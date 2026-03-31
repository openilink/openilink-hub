package sink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
)

const typingTimeout = 30 * time.Second
const maxImageBytes = 20 * 1024 * 1024 // 20MB

// BotModelSyncer allows the AI sink to sync an in-memory bot's model after a
// /model switch without importing the bot package (which would create a cycle).
type BotModelSyncer interface {
	SetBotAIModel(botDBID, model string)
}

// AI calls an OpenAI-compatible chat completion API and sends the reply
// back through the bot. Supports tool calling via installed App tools.
type AI struct {
	Store      store.Store
	AppDisp    *appdelivery.Dispatcher
	Storage    storage.Store
	BotManager BotModelSyncer
}

func (s *AI) Name() string { return "ai" }

func (s *AI) Handle(d Delivery) {
	if !d.AIEnabled {
		return
	}
	if d.MsgType != "text" && d.MsgType != "image" {
		return
	}
	if d.MsgType == "text" && d.Content == "" {
		return
	}
	// Skip messages targeted at specific apps (commands and @mentions).
	// For image messages, d.Content may be a placeholder; the real caption
	// is checked after extraction in reply().
	if d.MsgType == "text" {
		trimmed := strings.TrimSpace(d.Content)
		if strings.HasPrefix(trimmed, "@") {
			return
		}
		if strings.HasPrefix(trimmed, "/") {
			s.handleCommand(d, trimmed)
			return
		}
	}
	s.reply(d)
}

func (s *AI) handleCommand(d Delivery, cmd string) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 || parts[0] != "/model" {
		return
	}

	global, _ := s.Store.ListConfigByPrefix("ai.")
	availableRaw := global["ai.available_models"]
	var available []string
	if availableRaw != "" {
		if err := json.Unmarshal([]byte(availableRaw), &available); err != nil {
			slog.Warn("ai: malformed ai.available_models config", "err", err)
		}
	}

	sendText := func(text string) {
		d.Provider.Send(context.Background(), provider.OutboundMessage{
			Recipient: d.Message.Sender,
			Text:      text,
		})
	}

	if len(parts) == 1 {
		// List models
		if len(available) == 0 {
			sendText("没有可用的模型列表，请联系管理员配置。")
			return
		}
		current := d.AIModel
		if current == "" {
			current = global["ai.model"]
		}
		var lines []string
		for _, m := range available {
			mark := "  "
			if m == current {
				mark = "✓ "
			}
			lines = append(lines, mark+m)
		}
		sendText("可用模型：\n" + strings.Join(lines, "\n"))
		return
	}

	// Switch model
	requested := parts[1]
	valid := false
	for _, m := range available {
		if m == requested {
			valid = true
			break
		}
	}
	if !valid {
		sendText("模型不在可用列表中，请用 /model 查看可用模型。")
		return
	}

	if err := s.Store.UpdateBotAIModel(d.BotDBID, requested); err != nil {
		sendText("切换失败，请稍后重试。")
		return
	}
	if s.BotManager != nil {
		s.BotManager.SetBotAIModel(d.BotDBID, requested)
	}
	sendText("已切换到模型：" + requested)
}

func (s *AI) resolveConfig(botModel string) store.AIConfig {
	cfg := s.resolveGlobalConfig()
	if botModel != "" {
		cfg.Model = botModel
	}
	return cfg
}

func (s *AI) reply(d Delivery) {
	cfg := s.resolveConfig(d.AIModel)
	if cfg.APIKey == "" {
		slog.Warn("ai reply skipped: no api key", "bot", d.BotDBID)
		return
	}

	// Start trace span
	var span *store.SpanBuilder
	if d.Tracer != nil && d.RootSpan != nil {
		span = d.Tracer.StartChild(d.RootSpan, "ai_completion", store.SpanKindClient, map[string]any{
			"ai.model":  cfg.Model,
			"ai.source": cfg.Source,
			"reply.to":  d.Message.Sender,
		})
	}

	ctx := context.Background()
	sender := d.Message.Sender

	// Typing indicator
	var typingTicket string
	if d.Message.ContextToken != "" {
		if bcfg, err := d.Provider.GetConfig(ctx, sender, d.Message.ContextToken); err == nil && bcfg.TypingTicket != "" {
			typingTicket = bcfg.TypingTicket
			d.Provider.SendTyping(ctx, sender, typingTicket, true)
			go func() {
				time.Sleep(typingTimeout)
				d.Provider.SendTyping(context.Background(), sender, typingTicket, false)
			}()
		}
	}

	// Collect tools from installed apps
	tools := s.collectTools(d.BotDBID)
	if span != nil && len(tools) > 0 {
		span.SetAttr("ai.tools_count", len(tools))
	}

	// Download images from current message if it's an image type
	var currentImages []ai.ImageData
	text := d.Content
	if d.MsgType == "image" {
		text = "" // extract real text from items, not the "[image]" placeholder
		for _, item := range d.Message.Items {
			if item.Type == "text" && item.Text != "" {
				text = item.Text
			}
			if item.Type == "image" && item.Media != nil && item.Media.EncryptQueryParam != "" {
				data, err := d.Provider.DownloadMedia(ctx, item.Media)
				if err != nil {
					slog.Warn("ai: download image failed", "bot", d.BotDBID, "err", err)
					continue
				}
				if len(data) == 0 {
					continue
				}
				if len(data) > maxImageBytes {
					slog.Warn("ai: image too large, skipping", "bot", d.BotDBID, "size", len(data))
					continue
				}
				currentImages = append(currentImages, ai.ImageData{
					Data:        data,
					ContentType: http.DetectContentType(data),
				})
			}
		}
		if len(currentImages) == 0 && text == "" {
			return
		}
		// Check extracted caption for command/@mention prefixes
		trimmed := strings.TrimSpace(text)
		if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "@") {
			return
		}
	}

	// Create media resolver for history images
	var resolver ai.MediaResolver
	if s.Storage != nil {
		resolver = func(ctx context.Context, key string) ([]byte, error) {
			return s.Storage.Get(ctx, key)
		}
	}

	// Build messages for conversation context (reused across tool-call rounds)
	messages := ai.BuildMessages(ctx, cfg, s.Store, d.Channel.ID, sender, text, currentImages, resolver)
	result, err := ai.CompleteMessages(ctx, cfg, messages, tools)
	if err != nil {
		slog.Error("ai completion failed", "bot", d.BotDBID, "err", err)
		if span != nil {
			span.SetStatus(store.StatusError, err.Error())
			span.End()
		}
		s.stopTyping(d, typingTicket)
		s.sendErrorNotice(d, sender)
		return
	}

	// Accumulate token usage across all rounds
	var totalPrompt, totalCompletion, totalTokens, totalCached, totalReasoning int
	if result.Usage != nil {
		totalPrompt += result.Usage.PromptTokens
		totalCompletion += result.Usage.CompletionTokens
		totalTokens += result.Usage.TotalTokens
		totalCached += result.Usage.CachedTokens
		totalReasoning += result.Usage.ReasoningTokens
	}

	// Build installationID → appName map for status messages
	toolAppNames := make(map[string]string)
	for _, t := range tools {
		if idx := strings.Index(t.Function.Name, "__"); idx >= 0 {
			instID := t.Function.Name[:idx]
			// Extract app name from description "[AppName] ..."
			desc := t.Function.Description
			if len(desc) > 1 && desc[0] == '[' {
				if end := strings.Index(desc, "]"); end > 0 {
					toolAppNames[instID] = desc[1:end]
				}
			}
		}
	}

	// Tool call loop
	for round := 0; round < ai.MaxToolRounds && len(result.ToolCalls) > 0; round++ {
		// Send status message to user about tool calls
		for _, tc := range result.ToolCalls {
			toolName := tc.Name
			appName := ""
			if idx := strings.Index(tc.Name, "__"); idx >= 0 {
				appName = toolAppNames[tc.Name[:idx]]
				toolName = tc.Name[idx+2:]
			}
			status := fmt.Sprintf("🔧 调用 %s ...", toolName)
			if appName != "" {
				status = fmt.Sprintf("🔧 调用 %s 的 %s ...", appName, toolName)
			}
			d.Provider.Send(ctx, provider.OutboundMessage{
				Recipient: sender, Text: status,
			})
		}

		// Record assistant's tool_calls in messages
		messages = ai.AppendAssistantToolCalls(messages, result.ToolCalls)

		// Execute each tool call
		var toolResults []ai.ToolCallResult
		for _, tc := range result.ToolCalls {
			toolResult := s.executeToolCall(ctx, d, tc, span)
			toolResults = append(toolResults, toolResult)
		}

		// If all tool results are handled directly (images already sent, or async),
		// skip LLM continuation — the user will receive results without LLM involvement.
		skipLLM := true
		for _, tr := range toolResults {
			if len(tr.Images) == 0 && !tr.Async {
				skipLLM = false
				break
			}
		}
		if skipLLM {
			s.setTokenUsage(span, d.RootSpan, totalPrompt, totalCompletion, totalTokens, totalCached, totalReasoning)
			if span != nil {
				span.SetAttr("reply.content", "(tool handled directly)")
				span.End()
			}
			s.stopTyping(d, typingTicket)
			return
		}

		// Strip images from async/image results before passing to LLM.
		// We keep all tool results (required by OpenAI tool_calls protocol) but
		// clear images so they don't get sent as multimodal content to the LLM.
		var llmResults []ai.ToolCallResult
		for _, tr := range toolResults {
			if tr.Async || len(tr.Images) > 0 {
				tr.Images = nil
			}
			llmResults = append(llmResults, tr)
		}

		// Continue conversation with tool results
		var nextErr error
		result, messages, nextErr = ai.ContinueWithToolResults(ctx, cfg, messages, llmResults, tools)
		if nextErr != nil {
			slog.Error("ai continuation failed", "bot", d.BotDBID, "round", round+1, "err", nextErr)
			if span != nil {
				span.SetStatus(store.StatusError, nextErr.Error())
				span.End()
			}
			s.stopTyping(d, typingTicket)
			s.sendErrorNotice(d, sender)
			return
		}

		// Accumulate token usage from this round
		if result.Usage != nil {
			totalPrompt += result.Usage.PromptTokens
			totalCompletion += result.Usage.CompletionTokens
			totalTokens += result.Usage.TotalTokens
			totalCached += result.Usage.CachedTokens
			totalReasoning += result.Usage.ReasoningTokens
		}
	}

	s.setTokenUsage(span, d.RootSpan, totalPrompt, totalCompletion, totalTokens, totalCached, totalReasoning)

	s.stopTyping(d, typingTicket)

	reply := result.Content
	thinking := result.Thinking

	// Handle thinking/reasoning content
	if thinking != "" {
		if span != nil {
			span.SetAttr("ai.thinking_length", len(thinking))
		}
		if !cfg.HideThinking {
			reply = "💭 " + thinking + "\n\n" + reply
		}
	}

	// StripMarkdown runs after thinking is prepended, so both the thinking
	// content and the main reply are stripped when HideThinking=false.
	if cfg.StripMarkdown {
		reply = ai.StripMarkdown(reply)
	}

	if reply == "" {
		if span != nil {
			span.SetAttr("reply.content", "(empty)")
			span.End()
		}
		return
	}

	if span != nil {
		span.SetAttr("reply.content", reply)
	}

	_, err = d.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: sender,
		Text:      reply,
	})
	if err != nil {
		slog.Error("ai reply send failed", "bot", d.BotDBID, "err", err)
		if span != nil {
			span.SetStatus(store.StatusError, "send failed: "+err.Error())
			span.End()
		}
		return
	}

	if span != nil {
		span.End()
	}

	// Save only the content (not thinking) to message history to avoid polluting context
	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": result.Content}})
	s.Store.SaveMessage(&store.Message{
		BotID:       d.BotDBID,
		Direction:   "outbound",
		ToUserID:    sender,
		MessageType: 2,
		ItemList:    itemList,
	})
}

// collectTools gathers all tools from enabled app installations on this bot.
func (s *AI) collectTools(botID string) []ai.Tool {
	if s.AppDisp == nil {
		return nil
	}
	installations, err := s.Store.ListInstallationsByBot(botID)
	if err != nil {
		slog.Error("ai: list installations failed", "bot", botID, "err", err)
		return nil
	}

	var tools []ai.Tool
	for _, inst := range installations {
		if !inst.Enabled {
			continue
		}
		app, err := s.Store.GetApp(inst.AppID)
		if err != nil {
			continue
		}
		var appTools []store.AppTool
		json.Unmarshal(app.Tools, &appTools)
		for _, t := range appTools {
			if t.Name == "" {
				continue
			}
			params := t.Parameters
			if len(params) == 0 {
				params = json.RawMessage(`{"type":"object","properties":{}}`)
			}
			// Use installation ID as prefix for unique routing
			tools = append(tools, ai.Tool{
				Type: "function",
				Function: ai.ToolFunction{
					Name:        inst.ID + "__" + t.Name,
					Description: fmt.Sprintf("[%s] %s", inst.AppName, t.Description),
					Parameters:  params,
				},
			})
		}
	}
	return tools
}

// executeToolCall delivers a tool call to the corresponding app and returns the result.
func (s *AI) executeToolCall(ctx context.Context, d Delivery, tc ai.ToolCallRequest, parentSpan *store.SpanBuilder) ai.ToolCallResult {
	// Parse "installationID__tool_name" format
	name := tc.Name
	instID := ""
	toolName := name
	if idx := strings.Index(name, "__"); idx >= 0 {
		instID = name[:idx]
		toolName = name[idx+2:]
	}

	// Create child span for this tool call
	var span *store.SpanBuilder
	if d.Tracer != nil && parentSpan != nil {
		span = d.Tracer.StartChild(parentSpan, "tool_call:"+toolName, store.SpanKindClient, map[string]any{
			"tool.name": toolName,
			"tool.args": string(tc.Arguments),
		})
	}

	// Parse arguments
	var args map[string]any
	json.Unmarshal(tc.Arguments, &args)

	// Find the installation by ID
	installation, err := s.Store.GetInstallation(instID)
	if err != nil || installation == nil || !installation.Enabled || installation.BotID != d.BotDBID {
		errMsg := fmt.Sprintf("tool %q not found", toolName)
		slog.Warn("ai tool call: installation not found", "bot", d.BotDBID, "inst", instID, "tool", toolName)
		if span != nil {
			span.EndWithError(errMsg)
		}
		return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: errMsg}
	}

	if span != nil {
		span.SetAttr("app.name", installation.AppName)
	}

	// Build event (same format as command events).
	// sender is the real user; sender.role indicates AI Agent initiated the call.
	senderInfo := map[string]any{"id": d.Message.Sender, "role": "agent"}
	var groupInfo any
	if d.Message.GroupID != "" {
		groupInfo = map[string]any{"id": d.Message.GroupID}
	}
	event := appdelivery.NewEvent("command", map[string]any{
		"command": toolName,
		"text":    "",
		"args":    args,
		"sender":  senderInfo,
		"group":   groupInfo,
	})
	if d.Tracer != nil {
		event.TraceID = d.Tracer.TraceID()
	}

	// Deliver to app
	result := s.AppDisp.DeliverWithRetry(installation, event)

	if result == nil {
		if span != nil {
			span.EndWithError("no response")
		}
		return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: "tool returned no response"}
	}

	if span != nil {
		span.SetAttr("http.status_code", result.StatusCode)
		span.SetAttr("tool.result", truncateStr(result.Reply, 500))
	}

	// Handle async replies: app will push the result later via Bot API.
	if result.ReplyAsync {
		if span != nil {
			span.SetAttr("tool.reply_async", true)
			span.End()
		}
		return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: "result pending, will be delivered asynchronously", Async: true}
	}

	// Handle image replies: send image to user directly.
	// When all tool results in a round contain images, the caller skips LLM continuation.
	if result.ReplyType == "image" {
		images := s.resolveToolMedia(ctx, d.BotDBID, result)
		// Only include images that were actually delivered to the user.
		delivered := s.sendMediaToUser(ctx, d, images)
		if span != nil {
			span.SetAttr("tool.reply_type", result.ReplyType)
			span.End()
		}
		content := result.Reply
		if content == "" && len(delivered) == 0 {
			content = fmt.Sprintf("tool returned HTTP %d with no content", result.StatusCode)
		}
		return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: content, Images: delivered}
	}

	if span != nil {
		span.End()
	}

	content := result.Reply
	if content == "" {
		content = fmt.Sprintf("tool returned HTTP %d with no content", result.StatusCode)
	}
	return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: content}
}

// sendMediaToUser sends resolved images directly to the user via the provider.
// Returns only images that were successfully sent.
func (s *AI) sendMediaToUser(ctx context.Context, d Delivery, images []ai.ImageData) []ai.ImageData {
	sender := d.Message.Sender
	var delivered []ai.ImageData
	for _, img := range images {
		ct := img.ContentType
		fileName := "image.jpg"
		if strings.HasPrefix(ct, "image/png") {
			fileName = "image.png"
		} else if strings.HasPrefix(ct, "image/gif") {
			fileName = "image.gif"
		} else if strings.HasPrefix(ct, "image/webp") {
			fileName = "image.webp"
		}
		_, err := d.Provider.Send(ctx, provider.OutboundMessage{
			Recipient: sender, Data: img.Data, FileName: fileName,
		})
		if err != nil {
			slog.Error("ai tool media: send to user failed", "bot", d.BotDBID, "err", err)
			continue
		}
		delivered = append(delivered, img)
		itemList, _ := json.Marshal([]map[string]any{{"type": "image", "file_name": fileName}})
		mediaStatus := ""
		mediaKeys := json.RawMessage(`{}`)
		if s.Storage != nil {
			ext := ".jpg"
			if strings.HasPrefix(ct, "image/png") {
				ext = ".png"
			} else if strings.HasPrefix(ct, "image/gif") {
				ext = ".gif"
			} else if strings.HasPrefix(ct, "image/webp") {
				ext = ".webp"
			}
			now := time.Now()
			key := fmt.Sprintf("%s/%s/ai_%d%s", d.BotDBID, now.Format("2006/01/02"), now.UnixMilli(), ext)
			if _, err := s.Storage.Put(ctx, key, ct, img.Data); err == nil {
				mediaStatus = "ready"
				mediaKeys, _ = json.Marshal(map[string]string{"0": key})
			}
		}
		s.Store.SaveMessage(&store.Message{
			BotID: d.BotDBID, Direction: "outbound", ToUserID: sender, MessageType: 2,
			ItemList: itemList, MediaStatus: mediaStatus, MediaKeys: mediaKeys,
		})
	}
	return delivered
}

// resolveToolMedia resolves image data from a tool's media reply (base64 or URL).
func (s *AI) resolveToolMedia(ctx context.Context, botID string, result *appdelivery.DeliveryResult) []ai.ImageData {
	var data []byte
	var err error

	if result.ReplyBase64 != "" {
		b64 := result.ReplyBase64
		if idx := strings.Index(b64, ","); idx > 0 && strings.HasPrefix(b64, "data:") {
			b64 = b64[idx+1:]
		}
		data, err = base64.StdEncoding.DecodeString(b64)
		if err != nil {
			// Retry without padding (common in browser/JS encoders)
			data, err = base64.RawStdEncoding.DecodeString(b64)
			if err != nil {
				slog.Error("ai tool media: base64 decode failed", "bot", botID, "err", err)
				return nil
			}
		}
		if len(data) > maxImageBytes {
			slog.Error("ai tool media: base64 data too large", "bot", botID, "size", len(data))
			return nil
		}
	} else if result.ReplyURL != "" {
		u, err := url.Parse(result.ReplyURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			slog.Error("ai tool media: invalid url scheme", "bot", botID, "url", result.ReplyURL)
			return nil
		}
		dlCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, result.ReplyURL, nil)
		if err != nil {
			slog.Error("ai tool media: bad url", "bot", botID, "url", result.ReplyURL, "err", err)
			return nil
		}
		resp, err := safeHTTPClient.Do(req)
		if err != nil {
			slog.Error("ai tool media: download failed", "bot", botID, "url", result.ReplyURL, "err", err)
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			slog.Error("ai tool media: download returned non-200", "bot", botID, "url", result.ReplyURL, "status", resp.StatusCode)
			return nil
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, int64(maxImageBytes)+1))
		if err != nil {
			slog.Error("ai tool media: read failed", "bot", botID, "err", err)
			return nil
		}
		if len(data) > maxImageBytes {
			slog.Error("ai tool media: download too large", "bot", botID, "size", len(data))
			return nil
		}
	} else {
		return nil
	}

	if len(data) == 0 {
		return nil
	}

	ct := http.DetectContentType(data)
	if !strings.HasPrefix(ct, "image/") {
		slog.Warn("ai tool media: not an image", "bot", botID, "ct", ct)
		return nil
	}

	return []ai.ImageData{{
		Data:        data,
		ContentType: ct,
	}}
}

func (s *AI) stopTyping(d Delivery, ticket string) {
	if ticket != "" {
		d.Provider.SendTyping(context.Background(), d.Message.Sender, ticket, false)
	}
}

// sendErrorNotice sends a user-visible error message when AI completion fails.
// The detailed error is already logged via slog.Error at each call site;
// only a generic message is shown to the user to avoid leaking internal URLs
// or API response bodies.
func (s *AI) sendErrorNotice(d Delivery, recipient string) {
	if _, sendErr := d.Provider.Send(context.Background(), provider.OutboundMessage{
		Recipient: recipient,
		Text:      "⚠️ AI 回复失败，请稍后重试。",
	}); sendErr != nil {
		slog.Error("ai error notice send failed", "bot", d.BotDBID, "err", sendErr)
	}
}

func (s *AI) resolveGlobalConfig() store.AIConfig {
	global, _ := s.Store.ListConfigByPrefix("ai.")
	if global["ai.api_key"] == "" {
		return store.AIConfig{}
	}
	var cfg store.AIConfig
	cfg.Source = "builtin"
	cfg.BaseURL = global["ai.base_url"]
	cfg.APIKey = global["ai.api_key"]
	cfg.Model = global["ai.model"]
	cfg.SystemPrompt = global["ai.system_prompt"]
	cfg.HideThinking = global["ai.hide_thinking"] == "true"
	cfg.StripMarkdown = global["ai.strip_markdown"] == "true"
	if v := global["ai.max_history"]; v != "" {
		fmt.Sscanf(v, "%d", &cfg.MaxHistory)
	}
	if v := global["ai.custom_headers"]; v != "" {
		cfg.CustomHeaders = parseCustomHeaders(v)
	}
	return cfg
}

// parseCustomHeaders parses custom headers from JSON. Supports both array
// format [["key","value"],...] (from frontend) and object format {"key":"value"}.
func parseCustomHeaders(raw string) map[string]string {
	// Try array format first: [["k","v"],...]
	var arr [][2]string
	if json.Unmarshal([]byte(raw), &arr) == nil {
		m := make(map[string]string, len(arr))
		for _, kv := range arr {
			if kv[0] != "" {
				m[kv[0]] = kv[1]
			}
		}
		if len(m) > 0 {
			return m
		}
		return nil
	}
	// Fall back to object format: {"k":"v",...}
	var m map[string]string
	if json.Unmarshal([]byte(raw), &m) == nil && len(m) > 0 {
		return m
	}
	return nil
}

// safeHTTPClient blocks connections to private/internal IPs at the dial level,
// preventing SSRF via redirects and DNS rebinding.
var safeHTTPClient = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("ssrf: invalid addr %q", addr)
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("ssrf: dns lookup failed: %w", err)
			}
			for _, ip := range ips {
				if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() {
					return nil, fmt.Errorf("ssrf: blocked private ip %s for host %s", ip.IP, host)
				}
			}
			// Connect to the first allowed IP
			d := &net.Dialer{}
			return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	},
}

func (s *AI) setTokenUsage(span, rootSpan *store.SpanBuilder, prompt, completion, total, cached, reasoning int) {
	if total <= 0 {
		return
	}
	for _, sp := range []*store.SpanBuilder{span, rootSpan} {
		if sp == nil {
			continue
		}
		sp.SetAttr("ai.tokens.prompt", prompt)
		sp.SetAttr("ai.tokens.completion", completion)
		sp.SetAttr("ai.tokens.total", total)
		if cached > 0 {
			sp.SetAttr("ai.tokens.cached", cached)
		}
		if reasoning > 0 {
			sp.SetAttr("ai.tokens.reasoning", reasoning)
		}
	}
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
