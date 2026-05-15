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
	"os"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/supamemory"
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
	Store                    store.Store
	AppDisp                  *appdelivery.Dispatcher
	Storage                  storage.Store
	BotManager               BotModelSyncer
	SupaMemory               *supamemory.Client
	promptCache              map[string]cachedRuntimePrompt
	UsageBillingV2Enabled    bool
	UsageBillingCharsPerUnit int
}

type runtimePromptMeta struct {
	Source     string
	Version    int64
	FullHash   string
	Truncated  bool
	RoleID     string
	UserID     string
	UserPrompt string
}

type cachedRuntimePrompt struct {
	SystemPrompt  string
	UserPrompt    string
	PromptVersion int64
	CachedAt      time.Time
}

const runtimePromptCacheTTL = 120 * time.Second

func (s *AI) writeRuntimeAudit(d Delivery, eventType string, detail map[string]any) {
	if s.SupaMemory == nil || strings.TrimSpace(eventType) == "" {
		return
	}
	traceID := ""
	if d.Tracer != nil {
		traceID = d.Tracer.TraceID()
	}
	sessionID := d.Message.ContextToken
	if strings.TrimSpace(sessionID) == "" {
		sessionID = d.Message.Sender
	}
	go func() {
		_ = s.SupaMemory.WriteAuditLog(context.Background(), supamemory.AuditLogInput{
			EventType: eventType,
			SessionID: sessionID,
			TraceID:   traceID,
			Detail:    detail,
		})
	}()
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
		s.writeRuntimeAudit(d, "openilink_hub_ai_skipped_no_api_key", map[string]any{
			"bot_id": d.BotDBID,
		})
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
	billingEnabled, billingCharsPerUnit, billingSource := s.resolveUsageBillingConfig(ctx)
	usageUnits := 1
	if billingEnabled {
		usageUnits = calculateUsageUnitsV2(d.Content, d.Message.Items, billingCharsPerUnit)
	}
	cfg, promptMeta := s.resolveRuntimePrompt(ctx, cfg, d.BotDBID, d.Message.Recipient, d.Message.ContextToken, sender)
	channelCode := normalizeChannelCode(d.Provider.Name())
	channelPrompt := buildChannelPrompt(channelCode)
	cfg.SystemPrompt = composeSystemWithChannelPrompt(cfg.SystemPrompt, channelPrompt)
	s.writeRuntimeAudit(d, "openilink_hub_ai_reply_start", map[string]any{
		"bot_id":          d.BotDBID,
		"provider_bot_id": d.Message.Recipient,
		"context_token":   d.Message.ContextToken,
		"sender":          sender,
		"model":           cfg.Model,
		"prompt_source":   promptMeta.Source,
		"billing_source":  billingSource,
		"billing_enabled": billingEnabled,
		"channel_code":    channelCode,
		"user_id":         promptMeta.UserID,
		"role_id":         promptMeta.RoleID,
	})
	memQuery := strings.TrimSpace(d.Content)
	if s.SupaMemory != nil && strings.TrimSpace(promptMeta.UserID) != "" {
		quota, err := s.SupaMemory.CheckMonthlyQuota(ctx, promptMeta.UserID)
		if err != nil {
			slog.Warn("ai quota check failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", err)
			s.writeRuntimeAudit(d, "openilink_hub_ai_quota_check_failed", map[string]any{
				"bot_id":  d.BotDBID,
				"user_id": promptMeta.UserID,
				"role_id": promptMeta.RoleID,
				"error":   err.Error(),
				"sender":  sender,
				"model":   cfg.Model,
			})
		} else if quota != nil && !quota.Allowed {
			if span != nil {
				span.SetAttr("quota.blocked", true)
				span.SetAttr("quota.plan_code", quota.PlanCode)
				span.SetAttr("quota.period_month", quota.PeriodMonth)
				span.SetAttr("quota.used", quota.Used)
				span.SetAttr("quota.limit", quota.MonthlyLimit)
			}
			notice := "本月聊天额度已用完，请升级订阅或下月再试。"
			if quota.MonthlyLimit > 0 {
				notice = fmt.Sprintf("本月聊天额度已用完（%d/%d），请升级订阅或下月再试。", quota.Used, quota.MonthlyLimit)
			}
			if _, sendErr := d.Provider.Send(ctx, provider.OutboundMessage{
				Recipient: sender,
				Text:      notice,
			}); sendErr != nil {
				slog.Warn("quota notice send failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", sendErr)
				s.writeRuntimeAudit(d, "openilink_hub_ai_quota_notice_send_failed", map[string]any{
					"bot_id":    d.BotDBID,
					"user_id":   promptMeta.UserID,
					"role_id":   promptMeta.RoleID,
					"sender":    sender,
					"plan_code": quota.PlanCode,
					"used":      quota.Used,
					"limit":     quota.MonthlyLimit,
					"period":    quota.PeriodMonth,
					"error":     sendErr.Error(),
				})
			}
			s.writeRuntimeAudit(d, "openilink_hub_ai_quota_blocked", map[string]any{
				"bot_id":    d.BotDBID,
				"user_id":   promptMeta.UserID,
				"role_id":   promptMeta.RoleID,
				"sender":    sender,
				"plan_code": quota.PlanCode,
				"used":      quota.Used,
				"limit":     quota.MonthlyLimit,
				"period":    quota.PeriodMonth,
			})
			if span != nil {
				span.End()
			}
			return
		} else if quota != nil && quota.MonthlyLimit > 0 && (quota.Used+usageUnits) > quota.MonthlyLimit {
			if span != nil {
				span.SetAttr("quota.blocked", true)
				span.SetAttr("quota.plan_code", quota.PlanCode)
				span.SetAttr("quota.period_month", quota.PeriodMonth)
				span.SetAttr("quota.used", quota.Used)
				span.SetAttr("quota.limit", quota.MonthlyLimit)
				span.SetAttr("quota.request_units", usageUnits)
			}
			notice := fmt.Sprintf("本次消息预计消耗 %d 次额度，当前剩余不足（已用 %d/%d），请升级订阅或下月再试。", usageUnits, quota.Used, quota.MonthlyLimit)
			if _, sendErr := d.Provider.Send(ctx, provider.OutboundMessage{
				Recipient: sender,
				Text:      notice,
			}); sendErr != nil {
				slog.Warn("quota notice send failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", sendErr)
				s.writeRuntimeAudit(d, "openilink_hub_ai_quota_notice_send_failed", map[string]any{
					"bot_id":        d.BotDBID,
					"user_id":       promptMeta.UserID,
					"role_id":       promptMeta.RoleID,
					"sender":        sender,
					"plan_code":     quota.PlanCode,
					"used":          quota.Used,
					"limit":         quota.MonthlyLimit,
					"period":        quota.PeriodMonth,
					"request_units": usageUnits,
					"error":         sendErr.Error(),
				})
			}
			s.writeRuntimeAudit(d, "openilink_hub_ai_quota_blocked_by_units", map[string]any{
				"bot_id":        d.BotDBID,
				"user_id":       promptMeta.UserID,
				"role_id":       promptMeta.RoleID,
				"sender":        sender,
				"plan_code":     quota.PlanCode,
				"used":          quota.Used,
				"limit":         quota.MonthlyLimit,
				"period":        quota.PeriodMonth,
				"request_units": usageUnits,
			})
			if span != nil {
				span.End()
			}
			return
		}
	}

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

	memories := s.resolveMemories(ctx, cfg, promptMeta, memQuery)

	// Create media resolver for history images
	var resolver ai.MediaResolver
	if s.Storage != nil {
		resolver = func(ctx context.Context, key string) ([]byte, error) {
			return s.Storage.Get(ctx, key)
		}
	}

	// Build messages for conversation context (reused across tool-call rounds)
	messages := ai.BuildMessages(ctx, cfg, s.Store, d.Channel.ID, sender, text, currentImages, resolver)
	messages = injectUserPromptMessage(messages, promptMeta.UserPrompt)
	if len(memories) > 0 {
		memPrompt := buildMemoryPrompt(memories)
		if memPrompt != "" {
			for i := range messages {
				if messages[i].Role == "system" {
					if content, ok := messages[i].Content.(string); ok {
						messages[i].Content = strings.TrimSpace(content + "\n\n" + memPrompt)
					}
					break
				}
			}
		}
	}
	if span != nil {
		span.SetAttr("prompt.source", promptMeta.Source)
		span.SetAttr("prompt.version", promptMeta.Version)
		if promptMeta.FullHash != "" {
			span.SetAttr("prompt.full_hash", promptMeta.FullHash)
		}
		span.SetAttr("prompt.truncated", promptMeta.Truncated)
		span.SetAttr("memory.hit_count", len(memories))
		if promptMeta.RoleID != "" {
			span.SetAttr("memory.role_id", promptMeta.RoleID)
		}
		if promptMeta.UserID != "" {
			span.SetAttr("memory.user_id", promptMeta.UserID)
		}
	}
	result, err := ai.CompleteMessages(ctx, cfg, messages, tools)
	if err != nil {
		slog.Error("ai completion failed", "bot", d.BotDBID, "err", err)
		s.writeRuntimeAudit(d, "openilink_hub_ai_completion_failed", map[string]any{
			"bot_id":      d.BotDBID,
			"user_id":     promptMeta.UserID,
			"role_id":     promptMeta.RoleID,
			"sender":      sender,
			"model":       cfg.Model,
			"memory_hits": len(memories),
			"error":       err.Error(),
		})
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
			s.writeRuntimeAudit(d, "openilink_hub_ai_continuation_failed", map[string]any{
				"bot_id":  d.BotDBID,
				"user_id": promptMeta.UserID,
				"role_id": promptMeta.RoleID,
				"sender":  sender,
				"model":   cfg.Model,
				"round":   round + 1,
				"error":   nextErr.Error(),
			})
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
		s.writeRuntimeAudit(d, "openilink_hub_ai_reply_empty", map[string]any{
			"bot_id":  d.BotDBID,
			"user_id": promptMeta.UserID,
			"role_id": promptMeta.RoleID,
			"sender":  sender,
			"model":   cfg.Model,
		})
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
		s.writeRuntimeAudit(d, "openilink_hub_ai_reply_send_failed", map[string]any{
			"bot_id":      d.BotDBID,
			"user_id":     promptMeta.UserID,
			"role_id":     promptMeta.RoleID,
			"sender":      sender,
			"model":       cfg.Model,
			"reply_chars": len([]rune(reply)),
			"error":       err.Error(),
		})
		if span != nil {
			span.SetStatus(store.StatusError, "send failed: "+err.Error())
			span.End()
		}
		return
	}

	s.writeRuntimeAudit(d, "openilink_hub_ai_reply_sent", map[string]any{
		"bot_id":            d.BotDBID,
		"user_id":           promptMeta.UserID,
		"role_id":           promptMeta.RoleID,
		"sender":            sender,
		"model":             cfg.Model,
		"reply_chars":       len([]rune(reply)),
		"usage_units":       usageUnits,
		"prompt_tokens":     totalPrompt,
		"completion_tokens": totalCompletion,
		"total_tokens":      totalTokens,
		"cached_tokens":     totalCached,
		"reasoning_tokens":  totalReasoning,
		"memory_hits":       len(memories),
	})

	if span != nil {
		span.End()
	}

	if s.SupaMemory != nil && strings.TrimSpace(promptMeta.UserID) != "" {
		if err := s.SupaMemory.BumpMonthlyUsage(ctx, promptMeta.UserID, usageUnits); err != nil {
			slog.Warn("ai usage bump failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", err)
			s.writeRuntimeAudit(d, "openilink_hub_ai_usage_bump_failed", map[string]any{
				"bot_id":      d.BotDBID,
				"user_id":     promptMeta.UserID,
				"role_id":     promptMeta.RoleID,
				"sender":      sender,
				"usage_units": usageUnits,
				"error":       err.Error(),
			})
		} else {
			traceID := ""
			if d.Tracer != nil {
				traceID = d.Tracer.TraceID()
			}
			sessionID := d.Message.ContextToken
			if strings.TrimSpace(sessionID) == "" {
				sessionID = d.Message.Sender
			}
			if eventErr := s.SupaMemory.WriteUsageEvent(ctx, supamemory.UsageEventInput{
				UserID:      promptMeta.UserID,
				PeriodMonth: time.Now().UTC().Format("2006-01"),
				Delta:       usageUnits,
				Source:      "openilink_hub_ai",
				SessionID:   sessionID,
				TraceID:     traceID,
				Detail: map[string]any{
					"bot_id":      d.BotDBID,
					"role_id":     promptMeta.RoleID,
					"sender":      sender,
					"usage_units": usageUnits,
					"msg_type":    d.MsgType,
					"text_chars":  len([]rune(strings.Join(strings.Fields(text), " "))),
				},
			}); eventErr != nil {
				slog.Warn("ai usage event write failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", eventErr)
				s.writeRuntimeAudit(d, "openilink_hub_ai_usage_event_write_failed", map[string]any{
					"bot_id":      d.BotDBID,
					"user_id":     promptMeta.UserID,
					"role_id":     promptMeta.RoleID,
					"sender":      sender,
					"usage_units": usageUnits,
					"error":       eventErr.Error(),
				})
			}
		}
	}

	// Save only the content (not thinking) to message history to avoid polluting context
	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": result.Content}})
	saveRes, _ := s.Store.SaveMessage(&store.Message{
		BotID:       d.BotDBID,
		Direction:   "outbound",
		ToUserID:    sender,
		MessageType: 2,
		ItemList:    itemList,
	})
	if saveRes.Inserted {
		s.enqueueOutboundOutbox(d.BotDBID, saveRes.ID, &store.Message{
			BotID:       d.BotDBID,
			Direction:   "outbound",
			ToUserID:    sender,
			MessageType: 2,
			ItemList:    itemList,
		})
	}
	if s.SupaMemory != nil && promptMeta.RoleID != "" && promptMeta.UserID != "" {
		inboundText := strings.TrimSpace(text)
		replyText := strings.TrimSpace(result.Content)
		go func() {
			bg := context.Background()
			if inboundText != "" {
				_ = s.SupaMemory.RecordMemory(bg, supamemory.RecordInput{
					UserID:  promptMeta.UserID,
					RoleID:  promptMeta.RoleID,
					Content: inboundText,
					Source:  "openilink_user",
				})
			}
			if replyText != "" {
				_ = s.SupaMemory.RecordMemory(bg, supamemory.RecordInput{
					UserID:  promptMeta.UserID,
					RoleID:  promptMeta.RoleID,
					Content: replyText,
					Source:  "openilink_assistant",
				})
			}
		}()
	}
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
			params = ensureObjectSchema(params)
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

// ensureObjectSchema normalises a tool parameters value into a valid
// OpenAI-compatible JSON Schema ("type":"object").  It handles:
//   - empty / literal "null"  → default empty-object schema
//   - bare properties map (no "type"/"properties" keys) → wrapped
//   - per-property "required":true → hoisted to top-level "required" array
//   - already well-formed → cleaned of per-property "required" if present
func ensureObjectSchema(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}

	// Determine whether this is already a schema (has "type") or a bare
	// properties map (keys are property names like "text", "size").
	_, hasType := m["type"]

	var propsRaw map[string]json.RawMessage
	if hasType {
		// Already a schema – extract properties to clean them
		if p, ok := m["properties"]; ok {
			json.Unmarshal(p, &propsRaw)
		}
	} else {
		// Bare properties map – the top-level keys are property names
		propsRaw = m
	}

	// Hoist per-property "required":true to a top-level array and strip it.
	var required []string
	cleanedProps := make(map[string]any, len(propsRaw))
	for name, propRaw := range propsRaw {
		var prop map[string]any
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			cleanedProps[name] = propRaw
			continue
		}
		if req, ok := prop["required"]; ok {
			if b, isBool := req.(bool); isBool && b {
				required = append(required, name)
			}
			delete(prop, "required")
		}
		cleanedProps[name] = prop
	}

	schema := map[string]any{
		"type":       "object",
		"properties": cleanedProps,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	out, err := json.Marshal(schema)
	if err != nil {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return out
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
		saveRes, _ := s.Store.SaveMessage(&store.Message{
			BotID: d.BotDBID, Direction: "outbound", ToUserID: sender, MessageType: 2,
			ItemList: itemList, MediaStatus: mediaStatus, MediaKeys: mediaKeys,
		})
		if saveRes.Inserted {
			s.enqueueOutboundOutbox(d.BotDBID, saveRes.ID, &store.Message{
				BotID:       d.BotDBID,
				Direction:   "outbound",
				ToUserID:    sender,
				MessageType: 2,
				ItemList:    itemList,
				MediaStatus: mediaStatus,
				MediaKeys:   mediaKeys,
			})
		}
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

	var cfg store.AIConfig
	cfg.Source = "builtin"
	cfg.BaseURL = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_BASE_URL")), global["ai.base_url"])
	cfg.APIKey = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_API_KEY")), global["ai.api_key"])
	cfg.Model = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_MODEL")), global["ai.model"])
	cfg.FallbackModel = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_FALLBACK_MODEL")), global["ai.fallback_model"])
	cfg.SystemPrompt = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_SYSTEM_PROMPT")), global["ai.system_prompt"])
	if cfg.APIKey == "" {
		return store.AIConfig{}
	}

	hideThinkingRaw := strings.ToLower(firstNonEmpty(strings.TrimSpace(os.Getenv("AI_HIDE_THINKING")), global["ai.hide_thinking"]))
	cfg.HideThinking = hideThinkingRaw == "true" || hideThinkingRaw == "1"
	stripMarkdownRaw := strings.ToLower(firstNonEmpty(strings.TrimSpace(os.Getenv("AI_STRIP_MARKDOWN")), global["ai.strip_markdown"]))
	cfg.StripMarkdown = stripMarkdownRaw == "true" || stripMarkdownRaw == "1"
	if v := firstNonEmpty(strings.TrimSpace(os.Getenv("AI_MAX_HISTORY")), global["ai.max_history"]); v != "" {
		fmt.Sscanf(v, "%d", &cfg.MaxHistory)
	}
	if v := firstNonEmpty(strings.TrimSpace(os.Getenv("AI_CUSTOM_HEADERS")), global["ai.custom_headers"]); v != "" {
		cfg.CustomHeaders = parseCustomHeaders(v)
	}
	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeChannelCode(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "ilink":
		return "wechat"
	case "wechat", "telegram", "discord":
		return strings.ToLower(strings.TrimSpace(providerName))
	default:
		return "generic"
	}
}

func buildChannelPrompt(channelCode string) string {
	switch strings.ToLower(strings.TrimSpace(channelCode)) {
	case "wechat":
		return "渠道约束（WeChat）：移动端优先，短段落表达，避免超长单段与复杂排版。"
	case "discord":
		return "渠道约束（Discord）：可结构化表达，支持简洁列表与代码块，结论先行。"
	case "telegram":
		return "渠道约束（Telegram）：首屏先给结论，再给下一步，避免长段堆叠。"
	default:
		return "渠道约束：输出简洁、可执行、适配即时聊天阅读。"
	}
}

func composeSystemWithChannelPrompt(systemPrompt, channelPrompt string) string {
	sys := strings.TrimSpace(systemPrompt)
	ch := strings.TrimSpace(channelPrompt)
	switch {
	case sys != "" && ch != "":
		return sys + "\n\n" + ch
	case sys != "":
		return sys
	default:
		return ch
	}
}

func injectUserPromptMessage(messages []ai.Message, userPrompt string) []ai.Message {
	up := strings.TrimSpace(userPrompt)
	if up == "" {
		return messages
	}
	injected := ai.Message{
		Role: "user",
		Content: strings.TrimSpace("用户长期偏好（系统注入）：\n" +
			up + "\n请在后续回复中遵循这些偏好。"),
	}
	if len(messages) == 0 {
		return []ai.Message{injected}
	}
	if messages[0].Role == "system" {
		out := make([]ai.Message, 0, len(messages)+1)
		out = append(out, messages[0], injected)
		out = append(out, messages[1:]...)
		return out
	}
	out := make([]ai.Message, 0, len(messages)+1)
	out = append(out, injected)
	out = append(out, messages...)
	return out
}

func (s *AI) resolveRuntimePrompt(ctx context.Context, cfg store.AIConfig, botID, providerBotID, contextToken, sender string) (store.AIConfig, runtimePromptMeta) {
	meta := runtimePromptMeta{Source: "global_fallback"}
	globalPrompt := cfg.SystemPrompt
	if s.SupaMemory == nil {
		return cfg, meta
	}
	contextToken = strings.TrimSpace(contextToken)
	sender = strings.TrimSpace(sender)
	if contextToken == "" && sender == "" {
		return cfg, meta
	}

	var (
		bctx *supamemory.BindingContext
		berr error
	)
	if contextToken != "" {
		bctx, berr = s.SupaMemory.ResolveBindingContext(ctx, providerBotID, contextToken)
	}
	if (berr != nil || bctx == nil) && sender != "" && sender != contextToken {
		bctx, berr = s.SupaMemory.ResolveBindingContext(ctx, providerBotID, sender)
	}
	if berr != nil || bctx == nil {
		cfg.SystemPrompt = globalPrompt
		return cfg, meta
	}
	meta.RoleID = strings.TrimSpace(bctx.RoleID)
	meta.UserID = strings.TrimSpace(bctx.UserID)
	if meta.UserID == "" || meta.RoleID == "" {
		cfg.SystemPrompt = globalPrompt
		return cfg, meta
	}

	cacheKey := meta.UserID + ":" + meta.RoleID
	if hit, ok := s.getRuntimePromptCache(cacheKey, time.Now()); ok && !store.IsBlankPrompt(hit.SystemPrompt) {
		cfg.SystemPrompt = hit.SystemPrompt
		meta.UserPrompt = hit.UserPrompt
		meta.Source = "cache"
		meta.Version = hit.PromptVersion
		meta.FullHash = store.HashPrefix(store.HashPrompt(strings.TrimSpace(hit.SystemPrompt+"\n\n"+hit.UserPrompt)), 12)
		return cfg, meta
	}

	prompt, err := s.SupaMemory.GetEffectiveFullPrompt(ctx, meta.UserID, meta.RoleID)
	if err != nil || prompt == nil {
		cfg.SystemPrompt = globalPrompt
		return cfg, meta
	}
	resolvedSystemPrompt := strings.TrimSpace(prompt.SystemPrompt)
	resolvedUserPrompt := strings.TrimSpace(prompt.UserPrompt)
	if store.IsBlankPrompt(resolvedSystemPrompt) && store.IsBlankPrompt(resolvedUserPrompt) {
		cfg.SystemPrompt = globalPrompt
		return cfg, meta
	}
	if store.IsBlankPrompt(resolvedSystemPrompt) {
		resolvedSystemPrompt = strings.TrimSpace(globalPrompt)
	}

	cfg.SystemPrompt = resolvedSystemPrompt
	meta.UserPrompt = resolvedUserPrompt
	meta.Source = "supabase_rpc"
	meta.Version = prompt.PromptVersion
	meta.FullHash = store.HashPrefix(store.HashPrompt(strings.TrimSpace(resolvedSystemPrompt+"\n\n"+resolvedUserPrompt)), 12)
	s.setRuntimePromptCache(cacheKey, cachedRuntimePrompt{
		SystemPrompt:  resolvedSystemPrompt,
		UserPrompt:    resolvedUserPrompt,
		PromptVersion: prompt.PromptVersion,
		CachedAt:      time.Now(),
	})
	return cfg, meta
}

func (s *AI) getRuntimePromptCache(key string, now time.Time) (cachedRuntimePrompt, bool) {
	if s == nil || key == "" || s.promptCache == nil {
		return cachedRuntimePrompt{}, false
	}
	row, ok := s.promptCache[key]
	if !ok || now.Sub(row.CachedAt) > runtimePromptCacheTTL {
		return cachedRuntimePrompt{}, false
	}
	return row, true
}

func (s *AI) setRuntimePromptCache(key string, row cachedRuntimePrompt) {
	if s == nil || key == "" {
		return
	}
	if s.promptCache == nil {
		s.promptCache = make(map[string]cachedRuntimePrompt)
	}
	s.promptCache[key] = row
}

func (s *AI) resolveMemories(ctx context.Context, cfg store.AIConfig, meta runtimePromptMeta, currentText string) []supamemory.MemoryRow {
	if s.SupaMemory == nil {
		return nil
	}
	if strings.TrimSpace(meta.RoleID) == "" || strings.TrimSpace(meta.UserID) == "" {
		return nil
	}
	rows, err := s.SupaMemory.SearchMemories(ctx, meta.UserID, meta.RoleID, currentText, supamemory.SearchOptions{
		EmbeddingBase:  cfg.BaseURL,
		EmbeddingKey:   cfg.APIKey,
		EmbeddingModel: "",
		CustomHeaders:  cfg.CustomHeaders,
	})
	if err != nil {
		return nil
	}
	return rows
}

func buildMemoryPrompt(rows []supamemory.MemoryRow) string {
	if len(rows) == 0 {
		return ""
	}
	var lines []string
	lines = append(lines, "以下是与当前用户相关的历史记忆（仅作参考，若与当前事实冲突以当前对话为准）：")
	for _, row := range rows {
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		if len([]rune(content)) > 140 {
			content = string([]rune(content)[:140])
		}
		lines = append(lines, "- "+content)
	}
	if len(lines) == 1 {
		return ""
	}
	return strings.Join(lines, "\n")
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

func (s *AI) enqueueOutboundOutbox(botID string, msgID int64, msg *store.Message) {
	if s.Store == nil || msgID <= 0 || msg == nil {
		return
	}
	payload, _ := json.Marshal(map[string]any{
		"message_db_id": msgID,
		"bot_id":        botID,
		"direction":     "outbound",
		"to_user_id":    msg.ToUserID,
		"item_list":     msg.ItemList,
		"media_status":  msg.MediaStatus,
		"media_keys":    msg.MediaKeys,
	})
	eventID := fmt.Sprintf("msg:%s:%s:%d", store.OutboxEventMessageOutbound, botID, msgID)
	if _, _, err := s.Store.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
		EventID:      eventID,
		EventType:    store.OutboxEventMessageOutbound,
		PartitionKey: botID,
		Payload:      payload,
	}); err != nil {
		slog.Warn("enqueue outbox failed", "event_id", eventID, "bot", botID, "err", err)
	}
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func (s *AI) resolveUsageBillingConfig(ctx context.Context) (enabled bool, charsPerUnit int, source string) {
	// 先读 Supabase 配置表，统一多实例配置源；失败时回退本地 env 注入值。
	if s.SupaMemory != nil {
		cfg, err := s.SupaMemory.GetUsageBillingConfig(ctx)
		if err == nil && cfg != nil {
			if cfg.TextCharsPerUnit <= 0 {
				cfg.TextCharsPerUnit = 180
			}
			return cfg.Enabled, cfg.TextCharsPerUnit, "supabase_dict_items"
		}
	}
	chars := s.UsageBillingCharsPerUnit
	if chars <= 0 {
		chars = 180
	}
	return s.UsageBillingV2Enabled, chars, "env_fallback"
}

func calculateUsageUnitsV2(text string, items []provider.MessageItem, charsPerUnit int) int {
	if charsPerUnit <= 0 {
		charsPerUnit = 180
	}
	textUnits := calculateTextUnits(text, charsPerUnit)
	fileUnits := estimateFileUnits(items)
	if textUnits < 1 {
		textUnits = 1
	}
	if fileUnits > textUnits {
		return fileUnits
	}
	return textUnits
}

func calculateTextUnits(text string, charsPerUnit int) int {
	if charsPerUnit <= 0 {
		charsPerUnit = 180
	}
	normalized := strings.Join(strings.Fields(text), " ")
	chars := len([]rune(normalized))
	if chars <= 0 {
		return 1
	}
	units := (chars + charsPerUnit - 1) / charsPerUnit
	if units <= 0 {
		return 1
	}
	return units
}

func estimateFileUnits(items []provider.MessageItem) int {
	maxUnits := 0
	for _, item := range items {
		itemType := strings.ToLower(strings.TrimSpace(item.Type))
		if itemType == "" || itemType == "text" {
			continue
		}
		units := 2
		switch itemType {
		case "image":
			units = 2
		case "voice", "audio":
			units = 2
		case "video":
			units = 4
		case "file", "document":
			units = 3
		default:
			units = 2
		}
		if units > maxUnits {
			maxUnits = units
		}
	}
	return maxUnits
}
