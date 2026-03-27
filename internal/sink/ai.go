package sink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
)

const typingTimeout = 30 * time.Second

// AI calls an OpenAI-compatible chat completion API and sends the reply
// back through the bot. Supports tool calling via installed App tools.
type AI struct {
	Store      store.Store
	AppDisp    *appdelivery.Dispatcher
	Storage    *storage.Storage
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
	// Skip messages targeted at specific apps (commands and @mentions)
	trimmed := strings.TrimSpace(d.Content)
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "@") {
		return
	}
	s.reply(d)
}

func (s *AI) reply(d Delivery) {
	cfg := s.resolveGlobalConfig()
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
				data, err := d.Provider.DownloadMedia(ctx, item.Media.EncryptQueryParam, item.Media.AESKey)
				if err != nil {
					slog.Warn("ai: download image failed", "bot", d.BotDBID, "err", err)
					continue
				}
				if len(data) > 0 {
					currentImages = append(currentImages, ai.ImageData{
						Data:        data,
						ContentType: http.DetectContentType(data),
					})
				}
			}
		}
		if len(currentImages) == 0 && text == "" {
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

	// Build messages and do initial completion
	messages := ai.BuildMessages(cfg, s.Store, d.Channel.ID, sender, text, currentImages, resolver)
	result, err := ai.Complete(ctx, cfg, s.Store, d.Channel.ID, sender, text, tools, currentImages, resolver)
	if err != nil {
		slog.Error("ai completion failed", "bot", d.BotDBID, "err", err)
		if span != nil {
			span.SetStatus(store.StatusError, err.Error())
			span.End()
		}
		s.stopTyping(d, typingTicket)
		return
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

		// Continue conversation with tool results
		var nextErr error
		result, messages, nextErr = ai.ContinueWithToolResults(ctx, cfg, messages, toolResults, tools)
		if nextErr != nil {
			slog.Error("ai continuation failed", "bot", d.BotDBID, "round", round+1, "err", nextErr)
			if span != nil {
				span.SetStatus(store.StatusError, nextErr.Error())
				span.End()
			}
			s.stopTyping(d, typingTicket)
			return
		}
	}

	s.stopTyping(d, typingTicket)

	reply := result.Content
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

	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": reply}})
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

	// Build event (same format as command events)
	event := appdelivery.NewEvent("command", map[string]any{
		"command": toolName,
		"text":    "",
		"args":    args,
		"sender":  map[string]any{"id": "system", "name": "AI Agent"},
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

	// Handle image replies: send image to user AND pass to LLM as multimodal content
	if result.ReplyType == "image" {
		images := s.resolveToolMedia(ctx, d.BotDBID, result)
		// Send image directly to user so they see it immediately
		s.sendMediaToUser(ctx, d, images)
		if span != nil {
			span.SetAttr("tool.reply_type", result.ReplyType)
			span.End()
		}
		content := result.Reply
		if content == "" && len(images) == 0 {
			content = fmt.Sprintf("tool returned HTTP %d with no content", result.StatusCode)
		}
		return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: content, Images: images}
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
func (s *AI) sendMediaToUser(ctx context.Context, d Delivery, images []ai.ImageData) {
	sender := d.Message.Sender
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
		itemList, _ := json.Marshal([]map[string]any{{"type": "image", "file_name": fileName}})
		s.Store.SaveMessage(&store.Message{
			BotID: d.BotDBID, Direction: "outbound", ToUserID: sender, MessageType: 2, ItemList: itemList,
		})
	}
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
			slog.Error("ai tool media: base64 decode failed", "bot", botID, "err", err)
			return nil
		}
	} else if result.ReplyURL != "" {
		dlCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, result.ReplyURL, nil)
		if err != nil {
			slog.Error("ai tool media: bad url", "bot", botID, "url", result.ReplyURL, "err", err)
			return nil
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("ai tool media: download failed", "bot", botID, "url", result.ReplyURL, "err", err)
			return nil
		}
		defer resp.Body.Close()
		data, err = io.ReadAll(resp.Body)
		if err != nil {
			slog.Error("ai tool media: read failed", "bot", botID, "err", err)
			return nil
		}
	} else {
		return nil
	}

	if len(data) == 0 {
		return nil
	}

	return []ai.ImageData{{
		Data:        data,
		ContentType: http.DetectContentType(data),
	}}
}

func (s *AI) stopTyping(d Delivery, ticket string) {
	if ticket != "" {
		d.Provider.SendTyping(context.Background(), d.Message.Sender, ticket, false)
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
	if v := global["ai.max_history"]; v != "" {
		fmt.Sscanf(v, "%d", &cfg.MaxHistory)
	}
	return cfg
}


func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
