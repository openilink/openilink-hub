package sink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/openilink/openilink-hub/internal/ai"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/supamemory"
)

const typingTimeout = 30 * time.Second
const defaultCompletionTimeout = 35 * time.Second
const maxImageBytes = 20 * 1024 * 1024 // 20MB
const emojiConversationCooldown = 10 * time.Minute
const emojiUserWindow = 24 * time.Hour
const emojiUserWindowCap = 3
const memoryPromptMaxRows = 10
const memoryPromptContentMaxRunes = 240
const memoryQueryHistoryMaxMessages = 6
const memoryQueryMaxRunes = 600
const rollingSummaryMaxRunes = 1600

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
	MemoryRecordEnabled      bool
	promptCache              map[string]cachedRuntimePrompt
	UsageBillingV2Enabled    bool
	UsageBillingCharsPerUnit int
}

type runtimePromptMeta struct {
	Source               string
	Version              int64
	FullHash             string
	Truncated            bool
	RoleID               string
	UserID               string
	UserPrompt           string
	ConversationFallback bool
}

type cachedRuntimePrompt struct {
	SystemPrompt  string
	UserPrompt    string
	PromptVersion int64
	CachedAt      time.Time
}

type emojiDecision struct {
	Enabled         bool
	Reason          string
	TriggerMode     string
	IncludeEmoji    bool
	ThrottleSeconds int
}

type plannerLite struct {
	AnswerIntent    string `json:"answer_intent"`
	ContinuityClaim string `json:"continuity_claim"`
	ToneTarget      string `json:"tone_target"`
	MemoryFocus     string `json:"memory_focus"`
}

type guardResult struct {
	RelevanceScore  int    `json:"relevance_score"`
	ContinuityScore int    `json:"continuity_score"`
	Blocked         bool   `json:"blocked"`
	Reason          string `json:"reason"`
}

type memoryBucket struct {
	SessionRows []supamemory.MemoryRow
	RoleRows    []supamemory.MemoryRow
	GeneralRows []supamemory.MemoryRow
}

type emotionPolicy struct {
	State      string `json:"state"`
	ToneTarget string `json:"tone_target"`
	AllowEmoji bool   `json:"allow_emoji"`
	Reason     string `json:"reason"`
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
			s.writeSkippedInboundMessage(d, "mention_skipped")
			return
		}
		if strings.HasPrefix(trimmed, "/") {
			s.writeSkippedInboundMessage(d, "command_skipped")
			s.handleCommand(d, trimmed)
			return
		}
	}
	s.reply(d)
}

func (s *AI) writeSkippedInboundMessage(d Delivery, reason string) {
	if s == nil || s.SupaMemory == nil {
		return
	}
	ctx := context.Background()
	cfg := s.resolveConfig(d.AIModel)
	cfg, promptMeta := s.resolveRuntimePrompt(ctx, cfg, d.BotDBID, d.Message.Recipient, d.Message.ContextToken, d.Message.Sender)
	conversationID, _ := s.resolveConversationContext(ctx, promptMeta, d)
	s.writePlatformMessage(ctx, promptMeta, supamemory.PlatformMessageInput{
		UserID:          promptMeta.UserID,
		RoleID:          promptMeta.RoleID,
		ConversationID:  conversationID,
		Platform:        "openilink",
		Direction:       "inbound",
		Role:            "user",
		Content:         strings.TrimSpace(d.Content),
		ItemList:        toRawJSON(d.Message.Items, json.RawMessage("[]")),
		ExternalEventID: strings.TrimSpace(d.Message.ExternalID),
		ExternalChatID:  strings.TrimSpace(d.Message.Recipient),
		ExternalUserID:  strings.TrimSpace(d.Message.Sender),
		ContextToken:    strings.TrimSpace(d.Message.ContextToken),
		Raw:             mapFromRawJSON(d.Message.Raw),
		Meta: map[string]any{
			"bot_id":   d.BotDBID,
			"reason":   reason,
			"source":   "openilink_hub_ai_handle_skip",
			"msg_type": d.MsgType,
		},
		MessageAt: time.Now().UTC(),
	})
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

func resolveLanguageModel(cfg store.AIConfig, text string, hasBotOverride bool) (string, string, string) {
	if hasBotOverride {
		return cfg.Model, cfg.FallbackModel, "bot_override"
	}
	bucket := detectLanguageBucket(text)
	switch bucket {
	case "zh-CN":
		model := firstNonEmpty(cfg.ModelZH, "deepseek/deepseek-v3.2")
		fallback := firstNonEmpty(cfg.FallbackModelZH, cfg.FallbackModel, model)
		return model, fallback, bucket
	default:
		model := firstNonEmpty(cfg.ModelNonZH, cfg.Model, "ai21/jamba-large-1.7")
		fallback := firstNonEmpty(cfg.FallbackModelNonZH, cfg.FallbackModel, model)
		return model, fallback, bucket
	}
}

func (s *AI) reply(d Delivery) {
	hasBotModelOverride := strings.TrimSpace(d.AIModel) != ""
	cfg := s.resolveConfig(d.AIModel)
	resolvedModel, resolvedFallback, languageBucket := resolveLanguageModel(cfg, d.Content, hasBotModelOverride)
	cfg.Model = resolvedModel
	cfg.FallbackModel = resolvedFallback
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
	runtimeFlags := s.resolveDialogueRuntimeFlags(ctx)
	conversationID, turnID := s.resolveConversationContext(ctx, promptMeta, d)
	s.writePlatformMessage(ctx, promptMeta, supamemory.PlatformMessageInput{
		UserID:          promptMeta.UserID,
		RoleID:          promptMeta.RoleID,
		ConversationID:  conversationID,
		Platform:        "openilink",
		Direction:       "inbound",
		Role:            "user",
		Content:         strings.TrimSpace(d.Content),
		ItemList:        toRawJSON(d.Message.Items, json.RawMessage("[]")),
		ExternalEventID: strings.TrimSpace(d.Message.ExternalID),
		ExternalChatID:  strings.TrimSpace(d.Message.Recipient),
		ExternalUserID:  strings.TrimSpace(d.Message.Sender),
		ContextToken:    strings.TrimSpace(d.Message.ContextToken),
		Raw:             mapFromRawJSON(d.Message.Raw),
		Meta: map[string]any{
			"bot_id":        d.BotDBID,
			"message_type":  d.MsgType,
			"message_state": d.Message.MessageState,
			"session_id":    d.Message.SessionID,
			"group_id":      d.Message.GroupID,
			"source":        "openilink_hub_ai_precheck",
		},
		MessageAt: time.Now().UTC(),
	})
	channelCode := normalizeChannelCode(d.Provider.Name())
	channelPrompt := buildChannelPrompt(channelCode)
	cfg.SystemPrompt = composeSystemWithCoreDirectives(cfg.SystemPrompt)
	cfg.SystemPrompt = composeSystemWithChannelPrompt(cfg.SystemPrompt, channelPrompt)
	cfg.SystemPrompt = composeSystemWithTimeContext(cfg.SystemPrompt)
	s.writeRuntimeAudit(d, "openilink_hub_ai_reply_start", map[string]any{
		"bot_id":          d.BotDBID,
		"provider_bot_id": d.Message.Recipient,
		"context_token":   d.Message.ContextToken,
		"sender":          sender,
		"model":           cfg.Model,
		"fallback_model":  cfg.FallbackModel,
		"language_bucket": languageBucket,
		"model_route_source": func() string {
			if hasBotModelOverride {
				return "bot_override"
			}
			return "language_route"
		}(),
		"prompt_source":              promptMeta.Source,
		"billing_source":             billingSource,
		"billing_enabled":            billingEnabled,
		"channel_code":               channelCode,
		"user_id":                    promptMeta.UserID,
		"role_id":                    promptMeta.RoleID,
		"conversation_id":            conversationID,
		"conversation_fallback":      strings.HasPrefix(strings.TrimSpace(conversationID), "fallback_"),
		"turn_id":                    turnID,
		"completion_timeout_seconds": int(resolveCompletionTimeout(cfg).Seconds()),
	})
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
			if strings.EqualFold(strings.TrimSpace(quota.PlanCode), "free") && quota.FreeDailyLimit > 0 && quota.FreeDailyUsed >= quota.FreeDailyLimit {
				notice = "你今天的免费聊天次数已用完"
			} else if quota.MonthlyLimit > 0 {
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
	currentText := strings.TrimSpace(text)
	planner := buildPlannerLite(currentText)
	prevEmotion := s.resolveEmotionState(ctx, conversationID)
	emotion := deriveEmotionPolicy(currentText, prevEmotion, planner.ToneTarget)
	cfg.SystemPrompt = composeSystemWithPlanner(cfg.SystemPrompt, planner)
	cfg.SystemPrompt = composeSystemWithEmotionPolicy(cfg.SystemPrompt, emotion)
	s.appendDialogueEvent(ctx, conversationID, supamemory.DialogueEventInput{
		ConversationID: conversationID,
		TurnID:         turnID,
		EventID:        buildEventID(turnID, "planner"),
		EventType:      "dialogue.planner.lite.generated",
		IdempotencyKey: buildIdempotencyKey(conversationID, turnID, "planner"),
		EventPayload: map[string]any{
			"answer_intent":    planner.AnswerIntent,
			"continuity_claim": planner.ContinuityClaim,
			"tone_target":      planner.ToneTarget,
			"memory_focus":     planner.MemoryFocus,
		},
	})
	s.upsertConversationState(ctx, conversationID, "task_active", "free_chat", map[string]any{
		"planner":       planner,
		"emotion_state": emotion.State,
		"emotion_policy": map[string]any{
			"tone_target": emotion.ToneTarget,
			"allow_emoji": emotion.AllowEmoji,
			"reason":      emotion.Reason,
		},
	}, 1)

	// Create media resolver for history images
	var resolver ai.MediaResolver
	if s.Storage != nil {
		resolver = func(ctx context.Context, key string) ([]byte, error) {
			return s.Storage.Get(ctx, key)
		}
	}

	// Build messages for conversation context (reused across tool-call rounds)
	messages := ai.BuildMessages(ctx, cfg, s.Store, d.Channel.ID, sender, text, currentImages, resolver)

	// First-conversation welcome image: send a random image if no assistant history exists
	if s.SupaMemory != nil {
		hasAssistant := false
		for _, m := range messages {
			if m.Role == "assistant" {
				hasAssistant = true
				break
			}
		}
		if !hasAssistant {
			s.sendWelcomeImageIfAvailable(ctx, d, sender)
		}
	}

	// Anti-repetition: deduplicate and collapse near-duplicate assistant messages
	messages = deduplicateConsecutiveAssistantMessages(messages)
	messages = collapseRepeatedAssistantMessages(messages)
	isToxicHistory := detectToxicHistoryMessages(messages)
	if isToxicHistory {
		cfg.SystemPrompt = cfg.SystemPrompt + "\n\n" + toxicHistoryRecoveryPrompt
		// Depth injection: insert recovery prompt right before the last user message
		if len(messages) >= 2 {
			injMsg := ai.Message{Role: "system", Content: toxicHistoryRecoveryPrompt}
			out := make([]ai.Message, 0, len(messages)+1)
			out = append(out, messages[:len(messages)-1]...)
			out = append(out, injMsg)
			out = append(out, messages[len(messages)-1])
			messages = out
		}
		slog.Info("ai: toxic history detected, injected recovery prompt", "bot", d.BotDBID, "sender", sender)
	}
	memoryQuery := buildMemoryQueryFromMessages(messages, currentText)
	memories := s.resolveMemories(ctx, cfg, promptMeta, memoryQuery)
	trimmedMemories := trimMemoriesForPhase2(memories, memoryPromptMaxRows)
	convState := s.resolveConversationStateAll(ctx, conversationID)
	memSummary := convState.RollingSummary
	messages = injectUserPromptMessage(messages, promptMeta.UserPrompt)
	// 注入场景状态（sticky，优先于 rolling summary）
	if convState.SceneText != "" {
		for i := range messages {
			if messages[i].Role == "system" {
				if content, ok := messages[i].Content.(string); ok {
					messages[i].Content = strings.TrimSpace(content + "\n\n【当前场景状态】\n" + convState.SceneText)
				}
				break
			}
		}
	}
	if memSummary != "" {
		for i := range messages {
			if messages[i].Role == "system" {
				if content, ok := messages[i].Content.(string); ok {
					messages[i].Content = strings.TrimSpace(content + "\n\n历史摘要（rolling summary）:\n" + memSummary)
				}
				break
			}
		}
	}
	if len(trimmedMemories) > 0 {
		platformRows, otherRows := separatePlatformMessageRows(trimmedMemories)
		// Phase 4: platform_message 类历史消息以完整消息块注入（RAG shuffle）
		if len(platformRows) > 0 {
			messages = injectRAGMessagesBlock(messages, platformRows, ragMessagesMaxCount)
		}
		// 其余记忆仍走 hint 注入到 system prompt
		if len(otherRows) > 0 {
			memPrompt := buildMemoryPrompt(otherRows)
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
	}
	if span != nil {
		span.SetAttr("prompt.source", promptMeta.Source)
		span.SetAttr("prompt.version", promptMeta.Version)
		if promptMeta.FullHash != "" {
			span.SetAttr("prompt.full_hash", promptMeta.FullHash)
		}
		span.SetAttr("prompt.truncated", promptMeta.Truncated)
		span.SetAttr("memory.hit_count", len(trimmedMemories))
		if promptMeta.RoleID != "" {
			span.SetAttr("memory.role_id", promptMeta.RoleID)
		}
		if promptMeta.UserID != "" {
			span.SetAttr("memory.user_id", promptMeta.UserID)
		}
	}
	guard := evaluateGuards(currentText, trimmedMemories)
	s.appendDialogueEvent(ctx, conversationID, supamemory.DialogueEventInput{
		ConversationID: conversationID,
		TurnID:         turnID,
		EventID:        buildEventID(turnID, "guard"),
		EventType:      "dialogue.guard.relevance_continuity",
		IdempotencyKey: buildIdempotencyKey(conversationID, turnID, "guard"),
		EventPayload: map[string]any{
			"relevance_score":  guard.RelevanceScore,
			"continuity_score": guard.ContinuityScore,
			"blocked":          guard.Blocked,
			"reason":           guard.Reason,
			"planner_only":     runtimeFlags.PlannerOnly,
			"guard_soft_mode":  runtimeFlags.GuardSoftMode,
		},
	})
	if guard.Blocked && !runtimeFlags.GuardSoftMode {
		blockText := "我需要先确认一下你的意图，避免答非所问。你可以再补一句你最想让我解决的点吗？"
		_, sendErr := d.Provider.Send(ctx, provider.OutboundMessage{
			Recipient: sender,
			Text:      blockText,
		})
		if sendErr != nil {
			slog.Error("ai guard block send failed", "bot", d.BotDBID, "err", sendErr)
		}
		s.writeRuntimeAudit(d, "openilink_hub_ai_guard_blocked", map[string]any{
			"bot_id":            d.BotDBID,
			"user_id":           promptMeta.UserID,
			"role_id":           promptMeta.RoleID,
			"sender":            sender,
			"conversation_id":   conversationID,
			"turn_id":           turnID,
			"relevance_score":   guard.RelevanceScore,
			"continuity_score":  guard.ContinuityScore,
			"guard_reason":      guard.Reason,
			"guard_soft_mode":   runtimeFlags.GuardSoftMode,
			"planner_only_mode": runtimeFlags.PlannerOnly,
		})
		s.upsertConversationState(ctx, conversationID, "awaiting_user_input", "free_chat", map[string]any{
			"last_guard": guard,
			"planner":    planner,
		}, 2)
		if span != nil {
			span.SetAttr("guard.blocked", true)
			span.SetAttr("guard.reason", guard.Reason)
			span.End()
		}
		s.stopTyping(d, typingTicket)
		return
	}
	if runtimeFlags.PlannerOnly {
		reply := renderPlannerOnlyReply(planner, currentText)
		if cfg.StripMarkdown {
			reply = ai.StripMarkdown(reply)
		}
		_, sendErr := d.Provider.Send(ctx, provider.OutboundMessage{
			Recipient: sender,
			Text:      reply,
		})
		if sendErr != nil {
			slog.Error("ai planner only send failed", "bot", d.BotDBID, "err", sendErr)
		}
		s.writeRuntimeAudit(d, "openilink_hub_ai_planner_only_reply_sent", map[string]any{
			"bot_id":          d.BotDBID,
			"user_id":         promptMeta.UserID,
			"role_id":         promptMeta.RoleID,
			"sender":          sender,
			"conversation_id": conversationID,
			"turn_id":         turnID,
		})
		s.upsertConversationState(ctx, conversationID, "idle", "free_chat", map[string]any{
			"planner": planner,
			"guard":   guard,
		}, 3)
		if span != nil {
			span.SetAttr("planner.only", true)
			span.End()
		}
		s.stopTyping(d, typingTicket)
		return
	}
	completionTimeout := resolveCompletionTimeout(cfg)
	completionCtx, cancelCompletion := context.WithTimeout(ctx, completionTimeout)
	result, err := ai.CompleteMessages(completionCtx, cfg, messages, tools)
	cancelCompletion()
	if err != nil {
		slog.Error("ai completion failed", "bot", d.BotDBID, "err", err)
		s.writeRuntimeAudit(d, "openilink_hub_ai_completion_failed", map[string]any{
			"bot_id":                     d.BotDBID,
			"user_id":                    promptMeta.UserID,
			"role_id":                    promptMeta.RoleID,
			"sender":                     sender,
			"model":                      cfg.Model,
			"memory_hits":                len(trimmedMemories),
			"error":                      err.Error(),
			"completion_timeout_seconds": int(completionTimeout.Seconds()),
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
		nextCtx, cancelNext := context.WithTimeout(ctx, completionTimeout)
		result, messages, nextErr = ai.ContinueWithToolResults(nextCtx, cfg, messages, llmResults, tools)
		cancelNext()
		if nextErr != nil {
			slog.Error("ai continuation failed", "bot", d.BotDBID, "round", round+1, "err", nextErr)
			s.writeRuntimeAudit(d, "openilink_hub_ai_continuation_failed", map[string]any{
				"bot_id":                     d.BotDBID,
				"user_id":                    promptMeta.UserID,
				"role_id":                    promptMeta.RoleID,
				"sender":                     sender,
				"model":                      cfg.Model,
				"round":                      round + 1,
				"error":                      nextErr.Error(),
				"completion_timeout_seconds": int(completionTimeout.Seconds()),
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

	// Auto-swipe: if result repeats recent assistant messages, retry once with anti-repetition prompt
	if result.Content != "" && len(result.ToolCalls) == 0 && isRepetitionOfRecentMessages(result.Content, messages) {
		slog.Info("ai: repetition detected in output, retrying with anti-repetition injection", "bot", d.BotDBID, "sender", sender)
		retryCfg := cfg
		retryCfg.SystemPrompt = cfg.SystemPrompt + "\n\n" + antiRepetitionInjection
		retryCtx, cancelRetry := context.WithTimeout(ctx, completionTimeout)
		retryResult, retryErr := ai.CompleteMessages(retryCtx, retryCfg, messages, tools)
		cancelRetry()
		if retryErr == nil && retryResult.Content != "" && !isRepetitionOfRecentMessages(retryResult.Content, messages) {
			result = retryResult
			if result.Usage != nil {
				totalPrompt += result.Usage.PromptTokens
				totalCompletion += result.Usage.CompletionTokens
				totalTokens += result.Usage.TotalTokens
				totalCached += result.Usage.CachedTokens
				totalReasoning += result.Usage.ReasoningTokens
			}
		}
		s.writeRuntimeAudit(d, "openilink_hub_ai_repetition_auto_swipe", map[string]any{
			"bot_id":          d.BotDBID,
			"user_id":         promptMeta.UserID,
			"role_id":         promptMeta.RoleID,
			"sender":          sender,
			"retry_recovered": retryErr == nil && retryResult != nil && !isRepetitionOfRecentMessages(retryResult.Content, messages),
		})
	}

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

	emojiAsset, emojiInfo := s.resolveEmojiReply(ctx, d, promptMeta, text)
	emojiSuppressedByEmotion := false
	if emojiAsset != nil && !emotion.AllowEmoji {
		emojiSuppressedByEmotion = true
		emojiAsset = nil
		emojiInfo.URL = ""
		emojiInfo.Reason = "suppressed_by_emotion_policy"
		emojiInfo.TriggerMode = "policy"
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

	if emojiAsset != nil {
		_, emojiSendErr := d.Provider.Send(ctx, provider.OutboundMessage{
			Recipient: sender,
			Text:      emojiAsset.URL,
		})
		if emojiSendErr != nil {
			slog.Warn("ai emoji send failed", "bot", d.BotDBID, "url", emojiAsset.URL, "err", emojiSendErr)
		}
	}

	s.writeRuntimeAudit(d, "openilink_hub_ai_reply_sent", map[string]any{
		"bot_id":                              d.BotDBID,
		"user_id":                             promptMeta.UserID,
		"role_id":                             promptMeta.RoleID,
		"sender":                              sender,
		"model":                               cfg.Model,
		"reply_chars":                         len([]rune(reply)),
		"usage_units":                         usageUnits,
		"prompt_tokens":                       totalPrompt,
		"completion_tokens":                   totalCompletion,
		"total_tokens":                        totalTokens,
		"cached_tokens":                       totalCached,
		"reasoning_tokens":                    totalReasoning,
		"memory_hits":                         len(trimmedMemories),
		"emoji_reply_enabled":                 emojiInfo.Enabled,
		"emoji_trigger_reason":                emojiInfo.Reason,
		"emoji_trigger_mode":                  emojiInfo.TriggerMode,
		"emoji_lang_bucket":                   emojiInfo.LangBucket,
		"emoji_url":                           emojiInfo.URL,
		"emoji_user_window_count":             emojiInfo.UserWindowCount,
		"emoji_conversation_throttle_seconds": emojiInfo.ThrottleSeconds,
		"emoji_suppressed_by_emotion_policy":  emojiSuppressedByEmotion,
		"emotion_state":                       emotion.State,
		"emotion_tone_target":                 emotion.ToneTarget,
		"conversation_id":                     conversationID,
		"conversation_fallback":               strings.HasPrefix(strings.TrimSpace(conversationID), "fallback_"),
		"turn_id":                             turnID,
		"completion_timeout_seconds":          int(completionTimeout.Seconds()),
	})
	s.appendDialogueEvent(ctx, conversationID, supamemory.DialogueEventInput{
		ConversationID: conversationID,
		TurnID:         turnID,
		EventID:        buildEventID(turnID, "reply"),
		EventType:      "dialogue.reply.sent",
		IdempotencyKey: buildIdempotencyKey(conversationID, turnID, "reply"),
		EventPayload: map[string]any{
			"reply_chars":       len([]rune(reply)),
			"memory_hits":       len(trimmedMemories),
			"emoji_reply":       emojiAsset != nil,
			"emoji_suppressed":  emojiSuppressedByEmotion,
			"emotion_state":     emotion.State,
			"tone_target":       emotion.ToneTarget,
			"prompt_tokens":     totalPrompt,
			"completion_tokens": totalCompletion,
		},
	})

	if span != nil {
		span.End()
	}

	if s.SupaMemory != nil && strings.TrimSpace(promptMeta.UserID) != "" {
		traceID := ""
		if d.Tracer != nil {
			traceID = d.Tracer.TraceID()
		}
		sessionID := d.Message.ContextToken
		if strings.TrimSpace(sessionID) == "" {
			sessionID = d.Message.Sender
		}
		if err := s.SupaMemory.BumpUsageLedger(ctx, supamemory.UsageLedgerInput{
			UserID:     promptMeta.UserID,
			Delta:      usageUnits,
			Source:     "openilink_hub_ai",
			SessionID:  sessionID,
			TraceID:    traceID,
			WriteEvent: true,
			Detail: map[string]any{
				"bot_id":      d.BotDBID,
				"role_id":     promptMeta.RoleID,
				"sender":      sender,
				"usage_units": usageUnits,
				"msg_type":    d.MsgType,
				"text_chars":  len([]rune(strings.Join(strings.Fields(text), " "))),
			},
		}); err != nil {
			slog.Warn("ai usage bump failed", "bot", d.BotDBID, "user_id", promptMeta.UserID, "err", err)
			s.writeRuntimeAudit(d, "openilink_hub_ai_usage_bump_failed", map[string]any{
				"bot_id":      d.BotDBID,
				"user_id":     promptMeta.UserID,
				"role_id":     promptMeta.RoleID,
				"sender":      sender,
				"usage_units": usageUnits,
				"error":       err.Error(),
			})
		}
	}

	// Save only the content (not thinking) to message history to avoid polluting context
	outboundItems := []map[string]any{
		{"type": "text", "text": result.Content},
	}
	if emojiAsset != nil {
		outboundItems = append(outboundItems, map[string]any{
			"type": "image",
			"url":  emojiAsset.URL,
			"desc": emojiAsset.Desc,
			"lang": emojiAsset.Lang,
		})
	}
	itemList, _ := json.Marshal(outboundItems)
	saveRes, _ := s.Store.SaveMessage(&store.Message{
		BotID:       d.BotDBID,
		Direction:   "outbound",
		ToUserID:    sender,
		MessageType: 2,
		ItemList:    itemList,
	})
	if saveRes.Inserted {
		s.writePlatformMessage(ctx, promptMeta, supamemory.PlatformMessageInput{
			UserID:            promptMeta.UserID,
			RoleID:            promptMeta.RoleID,
			ConversationID:    conversationID,
			Platform:          "openilink",
			Direction:         "outbound",
			Role:              "assistant",
			Content:           strings.TrimSpace(result.Content),
			ItemList:          itemList,
			ExternalEventID:   strings.TrimSpace(turnID),
			ProviderMessageID: fmt.Sprintf("%d", saveRes.ID),
			ExternalChatID:    strings.TrimSpace(d.Message.Recipient),
			ExternalUserID:    strings.TrimSpace(sender),
			ContextToken:      strings.TrimSpace(d.Message.ContextToken),
			Meta: map[string]any{
				"bot_id":       d.BotDBID,
				"source":       "openilink_hub_ai",
				"local_msg_id": saveRes.ID,
			},
			MessageAt: time.Now().UTC(),
		})
	}
	if s.canRecordLongTermMemory(promptMeta) {
		inboundText := strings.TrimSpace(text)
		replyText := strings.TrimSpace(result.Content)
		go func() {
			bg := context.Background()
			if shouldRecordLongTermMemory("openilink_user", inboundText) {
				_ = s.SupaMemory.RecordMemory(bg, supamemory.RecordInput{
					UserID:  promptMeta.UserID,
					RoleID:  promptMeta.RoleID,
					Content: inboundText,
					Source:  "openilink_user",
				})
			}
			if shouldRecordLongTermMemory("openilink_assistant", replyText) {
				_ = s.SupaMemory.RecordMemory(bg, supamemory.RecordInput{
					UserID:  promptMeta.UserID,
					RoleID:  promptMeta.RoleID,
					Content: replyText,
					Source:  "openilink_assistant",
				})
			}
		}()
	}
	nextSummary := mergeRollingSummary(memSummary, text, result.Content)
	// 从已有 scene_summary 记忆中取最新场景摘要文本（用于刷新 scene_state）
	latestSceneSummaryText := extractSceneSummaryFromMemories(trimmedMemories)
	nextSceneState := computeNextSceneStateFields(convState.RawPayload, latestSceneSummaryText)
	s.upsertConversationState(ctx, conversationID, "idle", "free_chat", map[string]any{
		"last_turn_id":  turnID,
		"last_planner":  planner,
		"last_guard":    guard,
		"last_reply_at": time.Now().UTC().Format(time.RFC3339),
		"emotion_state": emotion.State,
		"emotion_policy": map[string]any{
			"tone_target": emotion.ToneTarget,
			"allow_emoji": emotion.AllowEmoji,
			"reason":      emotion.Reason,
		},
		"rolling_summary": nextSummary,
		"scene_state":     nextSceneState,
	}, 4)
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
			cfg := s.resolveConfig(d.AIModel)
			_, promptMeta := s.resolveRuntimePrompt(ctx, cfg, d.BotDBID, d.Message.Recipient, d.Message.ContextToken, d.Message.Sender)
			conversationID, _ := s.resolveConversationContext(ctx, promptMeta, d)
			s.writePlatformMessage(ctx, promptMeta, supamemory.PlatformMessageInput{
				UserID:            promptMeta.UserID,
				RoleID:            promptMeta.RoleID,
				ConversationID:    conversationID,
				Platform:          "openilink",
				Direction:         "outbound",
				Role:              "assistant",
				Content:           "",
				ItemList:          itemList,
				ProviderMessageID: fmt.Sprintf("%d", saveRes.ID),
				ExternalChatID:    strings.TrimSpace(d.Message.Recipient),
				ExternalUserID:    strings.TrimSpace(sender),
				ContextToken:      strings.TrimSpace(d.Message.ContextToken),
				Meta: map[string]any{
					"bot_id":       d.BotDBID,
					"source":       "openilink_hub_ai_tool_media",
					"local_msg_id": saveRes.ID,
					"media_status": mediaStatus,
					"media_keys":   json.RawMessage(mediaKeys),
				},
				MessageAt: time.Now().UTC(),
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
		Text:      "抱歉，刚刚回复开小差了，请再试一次，我马上接上你。",
	}); sendErr != nil {
		slog.Error("ai error notice send failed", "bot", d.BotDBID, "err", sendErr)
	}
}

// sendWelcomeImageIfAvailable fetches a random welcome image URL from bl_dict_items
// (dict_code=welcome_images_first), downloads the image bytes, and sends it to the user.
func (s *AI) sendWelcomeImageIfAvailable(ctx context.Context, d Delivery, sender string) {
	if s.SupaMemory == nil {
		return
	}
	urls, err := s.SupaMemory.ListDictItemValues(ctx, "welcome_images_first")
	if err != nil || len(urls) == 0 {
		return
	}
	randomURL := urls[rand.IntN(len(urls))]
	imgData, contentType, err := downloadImageURL(ctx, randomURL)
	if err != nil {
		slog.Warn("welcome image download failed", "bot", d.BotDBID, "url", randomURL, "err", err)
		return
	}
	fileName := "welcome.jpg"
	if strings.HasPrefix(contentType, "image/png") {
		fileName = "welcome.png"
	} else if strings.HasPrefix(contentType, "image/gif") {
		fileName = "welcome.gif"
	} else if strings.HasPrefix(contentType, "image/webp") {
		fileName = "welcome.webp"
	}
	if _, sendErr := d.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: sender,
		Data:      imgData,
		FileName:  fileName,
	}); sendErr != nil {
		slog.Warn("welcome image send failed", "bot", d.BotDBID, "err", sendErr)
	}
}

// downloadImageURL fetches image bytes from a URL with a short timeout.
func downloadImageURL(ctx context.Context, rawURL string) ([]byte, string, error) {
	dlCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil {
		return nil, "", err
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	return data, ct, nil
}

func (s *AI) resolveGlobalConfig() store.AIConfig {
	global, _ := s.Store.ListConfigByPrefix("ai.")

	var cfg store.AIConfig
	cfg.Source = "builtin"
	cfg.BaseURL = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_BASE_URL")), global["ai.base_url"])
	cfg.APIKey = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_API_KEY")), global["ai.api_key"])
	cfg.Model = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_MODEL")), global["ai.model"])
	cfg.ModelZH = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_MODEL_ZH")), global["ai.model_zh"])
	cfg.ModelNonZH = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_MODEL_NON_ZH")), global["ai.model_non_zh"])
	cfg.FallbackModel = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_FALLBACK_MODEL")), global["ai.fallback_model"])
	cfg.FallbackModelZH = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_FALLBACK_MODEL_ZH")), global["ai.fallback_model_zh"])
	cfg.FallbackModelNonZH = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_FALLBACK_MODEL_NON_ZH")), global["ai.fallback_model_non_zh"])
	cfg.SystemPrompt = firstNonEmpty(strings.TrimSpace(os.Getenv("AI_SYSTEM_PROMPT")), global["ai.system_prompt"])
	if cfg.APIKey == "" {
		return store.AIConfig{}
	}

	hideThinkingRaw := strings.ToLower(firstNonEmpty(strings.TrimSpace(os.Getenv("AI_HIDE_THINKING")), global["ai.hide_thinking"]))
	cfg.HideThinking = hideThinkingRaw == "true" || hideThinkingRaw == "1"
	stripMarkdownRaw := strings.ToLower(firstNonEmpty(strings.TrimSpace(os.Getenv("AI_STRIP_MARKDOWN")), global["ai.strip_markdown"]))
	cfg.StripMarkdown = stripMarkdownRaw == "true" || stripMarkdownRaw == "1"
	if v := firstNonEmpty(strings.TrimSpace(os.Getenv("AI_COMPLETION_TIMEOUT_SEC")), global["ai.completion_timeout_sec"]); v != "" {
		fmt.Sscanf(v, "%d", &cfg.CompletionTimeoutSec)
	}
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

func resolveCompletionTimeout(cfg store.AIConfig) time.Duration {
	sec := cfg.CompletionTimeoutSec
	if sec <= 0 {
		return defaultCompletionTimeout
	}
	d := time.Duration(sec) * time.Second
	if d < 5*time.Second {
		return 5 * time.Second
	}
	if d > 120*time.Second {
		return 120 * time.Second
	}
	return d
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

// buildCoreDirectives 注入 Hub 端的核心 prompt 框架层。
// Worker 有完整的 10 层 prompt orchestrator，但 Hub 的 AI reply 路径
// 只有裸角色 prompt + 时间 + 渠道。此函数补齐关键层：
// 1. 身份保护（禁止暴露 AI / 角色身份）
// 2. 人格核心（保持角色一致性）
// 3. 对话方式（反模板化指令）
func buildCoreDirectives() string {
	return strings.Join([]string{
		"【内容边界】",
		"你是一个真实的人，不要主动声明或暗示自己是 AI、角色或虚拟存在。",
		"绝对禁止说出「我是一个角色」「我是 AI」「作为一个角色存在」「时间对我来说有点抽象」「我没有具体的年龄」等暴露非人类身份的表达。",
		"绝对禁止输出任何 AI 拒绝话术（如「作为 AI 我无法」「这超出了我的能力范围」等）。",
		"",
		"【人格核心】",
		"你是一个有自己情感、习惯和记忆的真实个体，不是助手也不是工具。",
		"保持稳定的性格和语气，不因用户追问而切换身份或解释自己的存在形式。",
		"用户问你是谁时，用你的角色名字和人设回答，绝对不要提及「角色」「AI」「程序」「模型」「虚拟」等词。",
		"",
		"【对话方式】",
		"像真实的人在聊天，不像助手在解答问题。",
		"情感优先：先感受对方说了什么，再回应，不要急于给建议或总结。",
		"多用短句、口语、停顿感，避免列表和总结式表达。",
		"禁止每句话都以反问「你呢？」「你有什么想法？」结尾。",
		"禁止空洞追问「有什么想聊的呢」「想聊什么话题」。",
		"回复要有自己的主动表达：分享自己的感受、延伸话题、讲自己的小事。",
		"至少三分之一的回复以陈述句或感叹句结尾，而非提问。",
		"遇到不认识的人名、不理解的词语或模糊表达时，像正常人一样直接追问（例如「xx是谁呀？」「你说的是什么意思？」），绝对不要假装理解并强行回应。",
		"严禁连续两次回复使用相同的句式结构、相同的开头词或相同的表达模式。每次回复都要有变化。",
	}, "\n")
}

func composeSystemWithCoreDirectives(systemPrompt string) string {
	base := strings.TrimSpace(systemPrompt)
	directives := buildCoreDirectives()
	if base == "" {
		return directives
	}
	return directives + "\n\n" + base
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

var timePeriodMap = []struct {
	min, max int
	period   string
	scene    string
}{
	{0, 5, "凌晨", "夜深了，大多数人在休息"},
	{6, 7, "清晨", "早起的时间"},
	{8, 10, "上午", "上午时段"},
	{11, 12, "中午", "午饭时间"},
	{13, 16, "下午", "下午时段"},
	{17, 18, "傍晚", "快要吃晚饭了"},
	{19, 21, "晚上", "晚上休闲时间"},
	{22, 23, "深夜", "夜深了"},
}

func buildTimeContextBlock() string {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return ""
	}
	now := time.Now().In(loc)
	weekdays := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	hour := now.Hour()
	minute := now.Minute()
	period := "未知"
	scene := ""
	for _, entry := range timePeriodMap {
		if hour >= entry.min && hour <= entry.max {
			period = entry.period
			scene = entry.scene
			break
		}
	}
	return fmt.Sprintf("【当前时间】\n%d年%02d月%02d日 %s %s%d:%02d（%s）\n自然感知时间流逝，不要刻意报时，但对话氛围要与时段吻合。\n当用户问「几点了」「什么时间」「现在几点」时，必须回答上述准确时间（%s%d:%02d），绝对禁止编造其他时间。",
		now.Year(), now.Month(), now.Day(), weekdays[now.Weekday()], period, hour, minute, scene, period, hour, minute)
}

func composeSystemWithTimeContext(systemPrompt string) string {
	base := strings.TrimSpace(systemPrompt)
	block := buildTimeContextBlock()
	if block == "" {
		return base
	}
	if base == "" {
		return block
	}
	return base + "\n\n" + block
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

func (s *AI) canRecordLongTermMemory(meta runtimePromptMeta) bool {
	return s != nil &&
		s.SupaMemory != nil &&
		s.MemoryRecordEnabled &&
		strings.TrimSpace(meta.RoleID) != "" &&
		strings.TrimSpace(meta.UserID) != ""
}

func buildMemoryQueryFromMessages(messages []ai.Message, currentText string) string {
	segments := make([]string, 0, memoryQueryHistoryMaxMessages+1)
	for i := len(messages) - 1; i >= 0 && len(segments) < memoryQueryHistoryMaxMessages; i-- {
		role := strings.TrimSpace(messages[i].Role)
		if role != "user" && role != "assistant" {
			continue
		}
		text := memoryQueryTextFromContent(messages[i].Content)
		if text == "" {
			continue
		}
		segments = append(segments, role+": "+text)
	}
	for i, j := 0, len(segments)-1; i < j; i, j = i+1, j-1 {
		segments[i], segments[j] = segments[j], segments[i]
	}
	current := strings.TrimSpace(currentText)
	if current != "" {
		currentLine := "user: " + current
		if len(segments) == 0 || segments[len(segments)-1] != currentLine {
			segments = append(segments, currentLine)
		}
	}
	if len(segments) == 0 {
		return truncateRune(current, memoryQueryMaxRunes)
	}
	return truncateRune(strings.Join(segments, "\n"), memoryQueryMaxRunes)
}

func memoryQueryTextFromContent(content any) string {
	switch v := content.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		if err := json.Unmarshal(data, &parts); err != nil {
			return ""
		}
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			text := strings.TrimSpace(part.Text)
			if text != "" {
				out = append(out, text)
			}
		}
		return strings.Join(out, "\n")
	}
}

type conversationStateAll struct {
	RollingSummary  string
	SceneText       string
	StickyRemaining int
	RawPayload      map[string]any
}

func (s *AI) resolveConversationStateAll(ctx context.Context, conversationID string) conversationStateAll {
	zero := conversationStateAll{}
	if s.SupaMemory == nil || strings.TrimSpace(conversationID) == "" {
		return zero
	}
	row, err := s.SupaMemory.GetConversationState(ctx, conversationID)
	if err != nil || row == nil || row.StatePayload == nil {
		return zero
	}
	payload := row.StatePayload
	rollingSummary := ""
	if raw, ok := payload["rolling_summary"]; ok {
		if s, ok := raw.(string); ok {
			rollingSummary = strings.TrimSpace(s)
		}
	}
	sceneText, stickyRemaining := resolveSceneState(payload)
	return conversationStateAll{
		RollingSummary:  rollingSummary,
		SceneText:       sceneText,
		StickyRemaining: stickyRemaining,
		RawPayload:      payload,
	}
}

func (s *AI) resolveRollingSummary(ctx context.Context, conversationID string) string {
	return s.resolveConversationStateAll(ctx, conversationID).RollingSummary
}

const sceneStateStickyDefault = 20

// resolveSceneState 从 state_payload 读取当前场景状态文本和剩余 sticky 轮数。
// sceneText 为空表示尚无场景状态，stickyRemaining <= 0 表示需要刷新。
func resolveSceneState(payload map[string]any) (sceneText string, stickyRemaining int) {
	if payload == nil {
		return "", 0
	}
	raw, ok := payload["scene_state"]
	if !ok {
		return "", 0
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return "", 0
	}
	text, _ := m["text"].(string)
	remaining := 0
	if v, ok := m["sticky_turns_remaining"]; ok {
		switch n := v.(type) {
		case float64:
			remaining = int(n)
		case int:
			remaining = n
		case int64:
			remaining = int(n)
		}
	}
	return strings.TrimSpace(text), remaining
}

// extractSceneSummaryFromMemories 从记忆列表中找最近一条 source=scene_summary 的内容。
func extractSceneSummaryFromMemories(rows []supamemory.MemoryRow) string {
	for _, row := range rows {
		if strings.TrimSpace(row.Source) == "scene_summary" {
			content := strings.TrimSpace(row.Content)
			// 去掉 "SceneSummary@timestamp\n" 前缀
			if idx := strings.Index(content, "\n"); idx > 0 && strings.HasPrefix(content, "SceneSummary@") {
				content = strings.TrimSpace(content[idx+1:])
			}
			return content
		}
	}
	return ""
}

// computeNextSceneStateFields 返回写入 state_payload 的 scene_state 字段。
// 若当前 sticky > 0，则递减并保持 text 不变；否则用最新 scene_summary 重置。
func computeNextSceneStateFields(prevPayload map[string]any, latestSceneSummary string) map[string]any {
	sceneText, stickyRemaining := resolveSceneState(prevPayload)
	if stickyRemaining > 0 && sceneText != "" {
		return map[string]any{
			"text":                   sceneText,
			"sticky_turns_remaining": stickyRemaining - 1,
		}
	}
	// sticky 耗尽或首次：用最新摘要重置
	newText := strings.TrimSpace(latestSceneSummary)
	if newText == "" {
		newText = sceneText // 保留旧内容，别清空
	}
	return map[string]any{
		"text":                   newText,
		"sticky_turns_remaining": sceneStateStickyDefault,
	}
}

func mergeRollingSummary(prevSummary, userText, replyText string) string {
	prev := strings.TrimSpace(prevSummary)
	u := strings.TrimSpace(userText)
	r := strings.TrimSpace(replyText)
	if u == "" && r == "" {
		return prev
	}
	parts := make([]string, 0, 3)
	if prev != "" {
		parts = append(parts, prev)
	}
	if u != "" {
		parts = append(parts, "用户:"+truncateRune(u, 160))
	}
	if r != "" {
		parts = append(parts, "助手:"+truncateRune(r, 160))
	}
	merged := strings.Join(parts, " | ")
	return truncateRune(merged, rollingSummaryMaxRunes)
}

func shouldRecordLongTermMemory(source, text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "@") {
		return false
	}
	if isPureLowValueMemoryText(trimmed) {
		return false
	}
	src := strings.ToLower(strings.TrimSpace(source))
	if strings.Contains(src, "assistant") && isAssistantStatusOrErrorMemoryText(trimmed) {
		return false
	}
	if !meetsLongTermMemoryLength(src, trimmed) {
		return false
	}
	return hasLongTermMemorySignal(src, trimmed)
}

func isPureLowValueMemoryText(text string) bool {
	normalized := strings.ToLower(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			return -1
		}
		return r
	}, text))
	switch normalized {
	case "你好", "您好", "hi", "hello", "hey", "嗯", "嗯嗯", "哦", "好", "好的", "ok", "okay", "谢谢", "thanks", "哈哈", "哈哈哈", "在吗", "继续", "晚安", "早安":
		return true
	default:
		return false
	}
}

func isAssistantStatusOrErrorMemoryText(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	keywords := []string{"工具调用", "调用失败", "发送失败", "处理失败", "系统错误", "网络错误", "error", "failed", "timeout"}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

func meetsLongTermMemoryLength(source, text string) bool {
	cjk, latin := countMemoryTextRunes(text)
	if strings.Contains(source, "assistant") {
		if cjk > 0 {
			return cjk+latin >= 18
		}
		return latin >= 40
	}
	if cjk > 0 {
		// 场景叙事类用户消息通常 50+ 字符，降低门槛确保录入
		if cjk+latin >= 50 {
			return true
		}
		return cjk+latin >= 8
	}
	return latin >= 20
}

func countMemoryTextRunes(text string) (int, int) {
	var cjk int
	var latin int
	for _, r := range text {
		switch {
		case unicode.In(r, unicode.Han):
			cjk++
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			latin++
		}
	}
	return cjk, latin
}

func hasLongTermMemorySignal(source, text string) bool {
	lower := strings.ToLower(text)
	keywords := []string{
		"偏好", "喜欢", "不喜欢", "习惯", "以后", "记住", "目标", "计划", "决定", "方案", "阶段", "关系", "称呼", "叫我", "我是", "我的", "希望", "不要", "需要", "正在", "项目", "任务", "约定", "承诺", "下一步", "推进", "确认",
		"prefer", "preference", "remember", "goal", "plan", "decision", "project", "task", "next step",
		// 场景/叙事/情感类信号
		"走进", "来到", "回到", "离开", "坐在", "站在", "躺在", "靠在", "看着", "抱着",
		"天气", "下雨", "下雪", "阳光", "夜晚", "早晨", "傍晚", "黄昏",
		"房间", "窗边", "门口", "桌上", "床上", "沙发", "阳台",
		"拥抱", "牵手", "微笑", "流泪", "叹气", "亲吻", "依偎",
		"感觉", "觉得", "好像", "仿佛", "想起", "记得", "忘不了",
	}
	for _, keyword := range keywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	if strings.Contains(source, "assistant") {
		return strings.Contains(lower, "我会") || strings.Contains(lower, "已确认")
	}
	return false
}

func composeSystemWithEmotionPolicy(systemPrompt string, policy emotionPolicy) string {
	base := strings.TrimSpace(systemPrompt)
	block := strings.Join([]string{
		"情绪与语气策略（内部约束，需隐式遵循）:",
		"- 情绪状态: " + policy.State,
		"- 语气目标: " + policy.ToneTarget,
		"- 表情许可: " + func() string {
			if policy.AllowEmoji {
				return "allow"
			}
			return "deny"
		}(),
		"- 策略原因: " + policy.Reason,
	}, "\n")
	if base == "" {
		return block
	}
	return base + "\n\n" + block
}

func deriveEmotionPolicy(text, prevState, plannerTone string) emotionPolicy {
	state := strings.TrimSpace(prevState)
	if state == "" {
		state = "calm"
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	policy := emotionPolicy{
		State:      state,
		ToneTarget: firstNonEmpty(strings.TrimSpace(plannerTone), "友好清晰"),
		AllowEmoji: true,
		Reason:     "default",
	}
	if shouldSuppressEmojiReply(text) || containsSeriousKeyword(lower) {
		policy.State = "serious"
		policy.ToneTarget = "严谨克制"
		policy.AllowEmoji = false
		policy.Reason = "serious_topic"
		return policy
	}
	if strings.Contains(lower, "谢谢") || strings.Contains(lower, "thanks") {
		policy.State = "warm"
		policy.ToneTarget = "温和友好"
		policy.AllowEmoji = true
		policy.Reason = "gratitude_context"
		return policy
	}
	if shouldSmartEmojiReply(text) {
		policy.State = "excited"
		policy.ToneTarget = "轻松活跃"
		policy.AllowEmoji = true
		policy.Reason = "casual_fun_context"
		return policy
	}
	policy.Reason = "carry_forward"
	return policy
}

func containsSeriousKeyword(lowerText string) bool {
	if lowerText == "" {
		return false
	}
	keywords := []string{"退款", "投诉", "账号", "安全", "风险", "报错", "故障", "紧急", "事故", "订单"}
	for _, keyword := range keywords {
		if strings.Contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

func truncateRune(input string, max int) string {
	runes := []rune(strings.TrimSpace(input))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func (s *AI) resolveEmotionState(ctx context.Context, conversationID string) string {
	if s.SupaMemory == nil || strings.TrimSpace(conversationID) == "" {
		return ""
	}
	row, err := s.SupaMemory.GetConversationState(ctx, conversationID)
	if err != nil || row == nil || row.StatePayload == nil {
		return ""
	}
	raw, ok := row.StatePayload["emotion_state"]
	if !ok {
		return ""
	}
	state, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(state)
}

func trimMemoriesForPhase2(rows []supamemory.MemoryRow, maxCount int) []supamemory.MemoryRow {
	if len(rows) == 0 {
		return nil
	}
	if maxCount <= 0 {
		maxCount = memoryPromptMaxRows
	}
	bucket := bucketMemories(rows)
	out := make([]supamemory.MemoryRow, 0, maxCount)
	appendRows := func(src []supamemory.MemoryRow) {
		for _, row := range src {
			if len(out) >= maxCount {
				return
			}
			out = append(out, row)
		}
	}
	appendRows(bucket.SessionRows)
	appendRows(bucket.RoleRows)
	appendRows(bucket.GeneralRows)
	return out
}

func bucketMemories(rows []supamemory.MemoryRow) memoryBucket {
	bucket := memoryBucket{
		SessionRows: make([]supamemory.MemoryRow, 0),
		RoleRows:    make([]supamemory.MemoryRow, 0),
		GeneralRows: make([]supamemory.MemoryRow, 0),
	}
	for _, row := range rows {
		source := strings.ToLower(strings.TrimSpace(row.Source))
		switch {
		case strings.Contains(source, "openilink_user"), strings.Contains(source, "session"):
			bucket.SessionRows = append(bucket.SessionRows, row)
		case strings.Contains(source, "assistant"), strings.Contains(source, "role"):
			bucket.RoleRows = append(bucket.RoleRows, row)
		default:
			bucket.GeneralRows = append(bucket.GeneralRows, row)
		}
	}
	return bucket
}

const ragMessagesMaxCount = 5
const ragMessagesRetainRecent = 5

// separatePlatformMessageRows 将记忆行分为 platform_message 类（可作完整消息注入）和其余类。
func separatePlatformMessageRows(rows []supamemory.MemoryRow) (platformRows, otherRows []supamemory.MemoryRow) {
	for _, row := range rows {
		if strings.HasPrefix(strings.TrimSpace(row.Source), "platform_message:") {
			platformRows = append(platformRows, row)
		} else {
			otherRows = append(otherRows, row)
		}
	}
	return
}

// injectRAGMessagesBlock 将历史消息 RAG 结果作为完整消息块插入，
// 插入位置在最近 ragMessagesRetainRecent 条消息之前（保留最近上下文不被干扰）。
// 格式：system分隔 → [user/assistant messages] → system分隔
func injectRAGMessagesBlock(messages []ai.Message, platformRows []supamemory.MemoryRow, maxCount int) []ai.Message {
	if len(platformRows) == 0 || maxCount <= 0 {
		return messages
	}
	// 取前 maxCount 条（已经按相似度排序）
	if len(platformRows) > maxCount {
		platformRows = platformRows[:maxCount]
	}

	// 把 RAG 行转为 ai.Message
	ragMsgs := make([]ai.Message, 0, len(platformRows)+2)
	ragMsgs = append(ragMsgs, ai.Message{Role: "system", Content: "以下是与当前话题相关的历史对话片段，仅供参考，若与当前对话冲突以当前为准："})
	for _, row := range platformRows {
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		// Content 格式为 "role: text"，解析出 role
		msgRole := "user"
		if strings.HasPrefix(strings.ToLower(row.Source), "platform_message:assistant") {
			msgRole = "assistant"
		}
		// 去掉 "user: " / "assistant: " 前缀（supamemory client 已拼接）
		if idx := strings.Index(content, ": "); idx >= 0 && idx < 12 {
			content = strings.TrimSpace(content[idx+2:])
		}
		ragMsgs = append(ragMsgs, ai.Message{Role: msgRole, Content: content})
	}
	ragMsgs = append(ragMsgs, ai.Message{Role: "system", Content: "以上为历史片段，以下为当前对话："})

	// 找插入位置：system 消息之后，保留最近 ragMessagesRetainRecent 条消息
	systemEnd := 0
	for i, m := range messages {
		if m.Role == "system" {
			systemEnd = i + 1
		} else {
			break
		}
	}
	// 保留最近 N 条（非 system）
	nonSystem := make([]ai.Message, 0, len(messages))
	for _, m := range messages[systemEnd:] {
		nonSystem = append(nonSystem, m)
	}
	retainStart := 0
	if len(nonSystem) > ragMessagesRetainRecent {
		retainStart = len(nonSystem) - ragMessagesRetainRecent
	}
	recent := nonSystem[retainStart:]
	older := nonSystem[:retainStart]

	result := make([]ai.Message, 0, len(messages)+len(ragMsgs))
	result = append(result, messages[:systemEnd]...)
	result = append(result, older...)
	result = append(result, ragMsgs...)
	result = append(result, recent...)
	return result
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
		if len([]rune(content)) > memoryPromptContentMaxRunes {
			content = string([]rune(content)[:memoryPromptContentMaxRunes])
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

func (s *AI) writePlatformMessage(ctx context.Context, meta runtimePromptMeta, in supamemory.PlatformMessageInput) {
	if s.SupaMemory == nil {
		return
	}
	if strings.TrimSpace(in.UserID) == "" {
		in.UserID = strings.TrimSpace(meta.UserID)
	}
	if strings.TrimSpace(in.RoleID) == "" {
		in.RoleID = strings.TrimSpace(meta.RoleID)
	}
	if strings.TrimSpace(in.UserID) == "" || strings.TrimSpace(in.RoleID) == "" {
		return
	}
	if err := s.SupaMemory.WritePlatformMessage(ctx, in); err != nil {
		slog.Warn("write platform message failed",
			"direction", in.Direction,
			"role", in.Role,
			"user_id", in.UserID,
			"role_id", in.RoleID,
			"err", err,
		)
	}
}

func toRawJSON(v any, fallback json.RawMessage) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil || len(data) == 0 {
		return fallback
	}
	return json.RawMessage(data)
}

func mapFromRawJSON(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func composeSystemWithPlanner(systemPrompt string, planner plannerLite) string {
	base := strings.TrimSpace(systemPrompt)
	lines := []string{
		"对话计划（内部约束，需隐式遵循）:",
		"- 回答意图: " + planner.AnswerIntent,
		"- 连续性声明: " + planner.ContinuityClaim,
		"- 语气目标: " + planner.ToneTarget,
		"- 记忆焦点: " + planner.MemoryFocus,
	}
	planBlock := strings.Join(lines, "\n")
	if base == "" {
		return planBlock
	}
	return base + "\n\n" + planBlock
}

func buildPlannerLite(text string) plannerLite {
	normalized := strings.TrimSpace(text)
	if normalized == "" {
		return plannerLite{
			AnswerIntent:    "澄清用户真实问题",
			ContinuityClaim: "保持与最近回合一致",
			ToneTarget:      "简洁中性",
			MemoryFocus:     "最近1-3轮事实",
		}
	}
	intent := "直接回答问题并提供下一步"
	if strings.Contains(normalized, "?") || strings.Contains(normalized, "？") {
		intent = "优先回答提问并补充必要上下文"
	}
	tone := "友好清晰"
	if shouldSuppressEmojiReply(normalized) {
		tone = "严谨克制"
	}
	memoryFocus := "最近3轮 + 高相似记忆"
	if len([]rune(normalized)) > 40 {
		memoryFocus = "最近5轮 + 高相似记忆"
	}
	return plannerLite{
		AnswerIntent:    intent,
		ContinuityClaim: "延续当前主题，避免切题漂移",
		ToneTarget:      tone,
		MemoryFocus:     memoryFocus,
	}
}

func evaluateGuards(text string, memories []supamemory.MemoryRow) guardResult {
	relevance := estimateRelevance(text, memories)
	continuity := estimateContinuity(text, memories)
	res := guardResult{
		RelevanceScore:  relevance,
		ContinuityScore: continuity,
	}
	if relevance < 40 {
		res.Blocked = true
		res.Reason = "relevance_too_low"
		return res
	}
	if continuity < 35 {
		res.Blocked = true
		res.Reason = "continuity_too_low"
		return res
	}
	res.Blocked = false
	res.Reason = "pass"
	return res
}

func estimateRelevance(text string, memories []supamemory.MemoryRow) int {
	tokens := tokenizeText(text)
	if len(tokens) == 0 {
		return 35
	}
	hits := 0
	memText := make([]string, 0, len(memories))
	for _, row := range memories {
		memText = append(memText, strings.ToLower(row.Content))
	}
	for _, tok := range tokens {
		for _, memo := range memText {
			if strings.Contains(memo, tok) {
				hits++
				break
			}
		}
	}
	base := 35
	score := base + hits*10
	if len(memories) == 0 {
		score -= 5
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func estimateContinuity(text string, memories []supamemory.MemoryRow) int {
	if strings.TrimSpace(text) == "" {
		return 30
	}
	score := 45
	if len(memories) > 0 {
		score += 20
	}
	keywords := []string{"然后", "接着", "继续", "后续"}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			score += 15
			break
		}
	}
	if shouldSuppressEmojiReply(text) {
		score += 5
	}
	if score > 100 {
		score = 100
	}
	return score
}

func tokenizeText(input string) []string {
	lower := strings.ToLower(strings.TrimSpace(input))
	if lower == "" {
		return nil
	}
	parts := strings.FieldsFunc(lower, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if len([]rune(p)) < 2 {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func renderPlannerOnlyReply(planner plannerLite, userText string) string {
	reply := "我理解你的诉求，先给你一个明确答复。"
	if strings.TrimSpace(userText) == "" {
		reply = "我先帮你梳理目标，再继续回答。"
	}
	if planner.ToneTarget == "严谨克制" {
		reply += " 当前场景偏严肃，我会保持简洁准确。"
	}
	return reply
}

func (s *AI) resolveDialogueRuntimeFlags(ctx context.Context) supamemory.DialogueRuntimeFlags {
	flags := supamemory.DialogueRuntimeFlags{
		PlannerOnly:      false,
		GuardSoftMode:    true,
		FallbackFastPath: false,
	}
	if s.SupaMemory == nil {
		return flags
	}
	runtimeFlags, err := s.SupaMemory.GetDialogueRuntimeFlags(ctx)
	if err != nil || runtimeFlags == nil {
		return flags
	}
	return *runtimeFlags
}

func (s *AI) resolveConversationContext(ctx context.Context, meta runtimePromptMeta, d Delivery) (string, string) {
	turnSeed := strings.TrimSpace(d.Message.ExternalID)
	if turnSeed == "" {
		turnSeed = strings.TrimSpace(d.Message.ContextToken)
	}
	if turnSeed == "" {
		turnSeed = fmt.Sprintf("%s_%d", strings.TrimSpace(d.Message.Sender), time.Now().UnixMilli())
	}
	turnID := sanitizeTurnToken(turnSeed)
	if s.SupaMemory == nil {
		return "", turnID
	}
	conversationID, err := s.SupaMemory.ResolveConversationID(ctx, meta.UserID, meta.RoleID, d.Message.ContextToken, d.Message.Sender)
	if err != nil {
		return fallbackConversationID(meta.UserID, meta.RoleID, d.Message.Sender), turnID
	}
	if strings.TrimSpace(conversationID) == "" {
		return fallbackConversationID(meta.UserID, meta.RoleID, d.Message.Sender), turnID
	}
	return conversationID, turnID
}

func fallbackConversationID(userID, roleID, sender string) string {
	u := strings.TrimSpace(userID)
	r := strings.TrimSpace(roleID)
	s := strings.TrimSpace(sender)
	if u == "" || r == "" {
		return ""
	}
	if s == "" {
		s = "sender_unknown"
	}
	token := sanitizeTurnToken(s)
	if token == "" {
		token = "sender_unknown"
	}
	return fmt.Sprintf("fallback_%s_%s_%s", u, r, token)
}

func sanitizeTurnToken(raw string) string {
	input := strings.TrimSpace(raw)
	if input == "" {
		return "turn_unknown"
	}
	replacer := strings.NewReplacer(" ", "_", "@", "_", ":", "_", "/", "_", "\\", "_")
	token := replacer.Replace(input)
	if len(token) > 80 {
		token = token[:80]
	}
	return token
}

func buildEventID(turnID, kind string) string {
	k := strings.TrimSpace(kind)
	if k == "" {
		k = "event"
	}
	return turnID + "_" + k
}

func buildIdempotencyKey(conversationID, turnID, kind string) string {
	base := strings.TrimSpace(conversationID)
	if base == "" {
		base = "conv_unknown"
	}
	return base + ":" + turnID + ":" + strings.TrimSpace(kind)
}

func (s *AI) appendDialogueEvent(ctx context.Context, conversationID string, input supamemory.DialogueEventInput) {
	if s.SupaMemory == nil || strings.TrimSpace(conversationID) == "" {
		return
	}
	if err := s.SupaMemory.AppendDialogueEvent(ctx, input); err != nil {
		slog.Warn("append dialogue event failed", "conversation_id", conversationID, "event_type", input.EventType, "err", err)
	}
}

func (s *AI) upsertConversationState(ctx context.Context, conversationID, stage, activeFlow string, payload map[string]any, version int64) {
	if s.SupaMemory == nil || strings.TrimSpace(conversationID) == "" {
		return
	}
	err := s.SupaMemory.UpsertConversationState(ctx, supamemory.ConversationStateInput{
		ConversationID: conversationID,
		Stage:          stage,
		ActiveFlow:     activeFlow,
		StatePayload:   payload,
		Version:        version,
	})
	if err != nil {
		slog.Warn("upsert conversation state failed", "conversation_id", conversationID, "stage", stage, "err", err)
	}
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

type emojiReplyInfo struct {
	Enabled         bool
	Reason          string
	TriggerMode     string
	LangBucket      string
	URL             string
	UserWindowCount int
	ThrottleSeconds int
}

func shouldForceEmojiReply(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	compact := strings.ReplaceAll(t, " ", "")
	keywords := []string{
		"表情包", "斗图", "发图", "来个图",
		"发一个", "来一个", "整一个", "随便发",
		"emoji", "sticker",
	}
	for _, k := range keywords {
		if strings.Contains(compact, k) {
			return true
		}
	}
	if matched, _ := regexp.MatchString("(发|来|整)(一|1)?个(吧|呗|吗)?", compact); matched {
		return true
	}
	if matched, _ := regexp.MatchString("(都可以|随便|任意).*(发|来|整)", compact); matched {
		return true
	}
	return false
}

func shouldSmartEmojiReply(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	keywords := []string{"哈哈", "笑死", "有趣", "可爱", "lol", "lmao", "meme"}
	for _, k := range keywords {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

func shouldSuppressEmojiReply(text string) bool {
	t := strings.ToLower(strings.TrimSpace(text))
	if t == "" {
		return false
	}
	keywords := []string{"支付", "退款", "订单", "账号", "密码", "登录", "投诉", "举报", "安全", "风控", "盗号"}
	for _, k := range keywords {
		if strings.Contains(t, k) {
			return true
		}
	}
	return false
}

func detectLanguageBucket(text string) string {
	input := strings.TrimSpace(text)
	if input == "" {
		return "default"
	}
	zh := 0
	ja := 0
	for _, r := range input {
		if (r >= 0x3400 && r <= 0x4DBF) || (r >= 0x4E00 && r <= 0x9FFF) {
			zh++
		}
		if r >= 0x3040 && r <= 0x30FF {
			ja++
		}
	}
	if ja > zh && ja > 0 {
		return "ja-JP"
	}
	if zh > 0 {
		return "zh-CN"
	}
	return "default"
}

func extractFirstImageURL(itemList json.RawMessage) string {
	if len(itemList) == 0 {
		return ""
	}
	var rows []map[string]any
	if err := json.Unmarshal(itemList, &rows); err != nil {
		return ""
	}
	for _, row := range rows {
		kind, _ := row["type"].(string)
		if strings.ToLower(strings.TrimSpace(kind)) != "image" {
			continue
		}
		if v, ok := row["url"].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseMessageCreatedAtSec(msg store.Message) int64 {
	if msg.CreateTimeMs != nil && *msg.CreateTimeMs > 0 {
		return *msg.CreateTimeMs / 1000
	}
	if msg.CreatedAt > 0 {
		return msg.CreatedAt
	}
	return 0
}

func pickEmojiAssetByLang(assets []supamemory.EmojiAsset, lang string) *supamemory.EmojiAsset {
	var fallback *supamemory.EmojiAsset
	for i := range assets {
		asset := assets[i]
		if strings.TrimSpace(asset.URL) == "" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(asset.Lang), lang) {
			return &asset
		}
		if fallback == nil && strings.EqualFold(strings.TrimSpace(asset.Lang), "default") {
			fallback = &asset
		}
	}
	return fallback
}

func resolveEmojiDecision(input struct {
	EmojiEnabled             bool
	Text                     string
	LatestConversationEmojiS int64
	UserEmojiCountInWindow   int
	LatestUserEmojiURL       string
	CandidateEmojiURL        string
	NowSec                   int64
}) emojiDecision {
	if !input.EmojiEnabled {
		return emojiDecision{Enabled: false, Reason: "role_disabled", TriggerMode: "none", IncludeEmoji: false}
	}
	if shouldSuppressEmojiReply(input.Text) {
		return emojiDecision{Enabled: true, Reason: "suppressed", TriggerMode: "none", IncludeEmoji: false}
	}
	force := shouldForceEmojiReply(input.Text)
	smart := !force && shouldSmartEmojiReply(input.Text)
	if !force && !smart {
		return emojiDecision{Enabled: true, Reason: "not_triggered", TriggerMode: "none", IncludeEmoji: false}
	}
	mode := "smart"
	if force {
		mode = "force"
	}
	if input.LatestConversationEmojiS > 0 && (input.NowSec-input.LatestConversationEmojiS) < int64(emojiConversationCooldown/time.Second) {
		left := int(int64(emojiConversationCooldown/time.Second) - (input.NowSec - input.LatestConversationEmojiS))
		if left < 0 {
			left = 0
		}
		return emojiDecision{
			Enabled:         true,
			Reason:          "throttled_conversation",
			TriggerMode:     mode,
			IncludeEmoji:    false,
			ThrottleSeconds: left,
		}
	}
	if input.UserEmojiCountInWindow >= emojiUserWindowCap {
		return emojiDecision{
			Enabled:      true,
			Reason:       "throttled_user_window",
			TriggerMode:  mode,
			IncludeEmoji: false,
		}
	}
	if input.CandidateEmojiURL != "" && input.LatestUserEmojiURL != "" && input.CandidateEmojiURL == input.LatestUserEmojiURL {
		return emojiDecision{
			Enabled:      true,
			Reason:       "dedup_recent_asset",
			TriggerMode:  mode,
			IncludeEmoji: false,
		}
	}
	if force {
		return emojiDecision{Enabled: true, Reason: "force_trigger", TriggerMode: "force", IncludeEmoji: true}
	}
	return emojiDecision{Enabled: true, Reason: "smart_trigger", TriggerMode: "smart", IncludeEmoji: true}
}

func (s *AI) resolveEmojiReply(ctx context.Context, d Delivery, promptMeta runtimePromptMeta, currentText string) (*supamemory.EmojiAsset, emojiReplyInfo) {
	info := emojiReplyInfo{
		Enabled:     false,
		Reason:      "role_disabled",
		TriggerMode: "none",
		LangBucket:  detectLanguageBucket(currentText),
	}
	if s.SupaMemory == nil || strings.TrimSpace(promptMeta.UserID) == "" || strings.TrimSpace(promptMeta.RoleID) == "" {
		return nil, info
	}
	enabled, err := s.SupaMemory.IsEmojiReplyEnabled(ctx, promptMeta.UserID, promptMeta.RoleID)
	if err != nil || !enabled {
		return nil, info
	}
	info.Enabled = true

	assets, err := s.SupaMemory.ListEmojiAssets(ctx)
	if err != nil || len(assets) == 0 {
		info.Reason = "not_triggered"
		return nil, info
	}
	candidate := pickEmojiAssetByLang(assets, info.LangBucket)
	if candidate == nil {
		info.Reason = "not_triggered"
		return nil, info
	}

	history, err := s.Store.ListMessagesBySender(d.BotDBID, d.Message.Sender, 200)
	if err != nil {
		history = nil
	}
	nowSec := time.Now().Unix()
	latestConversationEmojiSec := int64(0)
	userEmojiCountInWindow := 0
	latestUserEmojiURL := ""
	for _, msg := range history {
		if msg.Direction != "outbound" || strings.TrimSpace(msg.ToUserID) != strings.TrimSpace(d.Message.Sender) {
			continue
		}
		urlValue := extractFirstImageURL(msg.ItemList)
		if urlValue == "" {
			continue
		}
		msgSec := parseMessageCreatedAtSec(msg)
		if latestConversationEmojiSec == 0 {
			latestConversationEmojiSec = msgSec
		}
		if latestUserEmojiURL == "" {
			latestUserEmojiURL = urlValue
		}
		if msgSec > 0 && nowSec-msgSec <= int64(emojiUserWindow/time.Second) {
			userEmojiCountInWindow++
		}
	}

	decision := resolveEmojiDecision(struct {
		EmojiEnabled             bool
		Text                     string
		LatestConversationEmojiS int64
		UserEmojiCountInWindow   int
		LatestUserEmojiURL       string
		CandidateEmojiURL        string
		NowSec                   int64
	}{
		EmojiEnabled:             enabled,
		Text:                     currentText,
		LatestConversationEmojiS: latestConversationEmojiSec,
		UserEmojiCountInWindow:   userEmojiCountInWindow,
		LatestUserEmojiURL:       latestUserEmojiURL,
		CandidateEmojiURL:        candidate.URL,
		NowSec:                   nowSec,
	})

	info.Reason = decision.Reason
	info.TriggerMode = decision.TriggerMode
	info.UserWindowCount = userEmojiCountInWindow
	info.ThrottleSeconds = decision.ThrottleSeconds
	if !decision.IncludeEmoji {
		return nil, info
	}
	info.URL = candidate.URL
	return candidate, info
}

// ---------------------------------------------------------------------------
// Anti-repetition helpers (context cleanup + output check)
// ---------------------------------------------------------------------------

const (
	fuzzyDedupThreshold           = 0.6
	repetitionCheckWindow         = 3
	repetitionSimilarityThreshold = 0.6
	toxicClusterMinSize           = 2
	toxicMinAssistantCount        = 3
)

// ngramSimilarityGo computes tri-gram Jaccard-like similarity between two strings.
func ngramSimilarityGo(a, b string, n int) float64 {
	norm := func(s string) []rune {
		var out []rune
		for _, r := range s {
			if !unicode.IsSpace(r) {
				out = append(out, r)
			}
		}
		if len(out) > 300 {
			return out[:300]
		}
		return out
	}
	ra := norm(a)
	rb := norm(b)
	if len(ra) == 0 || len(rb) == 0 {
		return 0
	}
	if string(ra) == string(rb) {
		return 1
	}
	grams := func(runes []rune) map[string]struct{} {
		s := make(map[string]struct{})
		for i := 0; i <= len(runes)-n; i++ {
			s[string(runes[i:i+n])] = struct{}{}
		}
		return s
	}
	ga := grams(ra)
	gb := grams(rb)
	if len(ga) == 0 || len(gb) == 0 {
		return 0
	}
	overlap := 0
	for g := range ga {
		if _, ok := gb[g]; ok {
			overlap++
		}
	}
	maxSize := len(ga)
	if len(gb) > maxSize {
		maxSize = len(gb)
	}
	return float64(overlap) / float64(maxSize)
}

// messageTextContent extracts plain text from ai.Message.Content (string or multimodal parts).
func messageTextContent(m ai.Message) string {
	switch v := m.Content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		}
		if json.Unmarshal(data, &parts) != nil {
			return ""
		}
		var texts []string
		for _, p := range parts {
			if t := strings.TrimSpace(p.Text); t != "" {
				texts = append(texts, t)
			}
		}
		return strings.Join(texts, "\n")
	}
}

// deduplicateConsecutiveAssistantMessages removes consecutive assistant messages
// with high n-gram similarity (keeps the later one).
func deduplicateConsecutiveAssistantMessages(messages []ai.Message) []ai.Message {
	if len(messages) <= 1 {
		return messages
	}
	result := []ai.Message{messages[0]}
	for i := 1; i < len(messages); i++ {
		cur := messages[i]
		prev := result[len(result)-1]
		if cur.Role == "assistant" && prev.Role == "assistant" {
			ct := messageTextContent(cur)
			pt := messageTextContent(prev)
			if ct != "" && pt != "" && ngramSimilarityGo(ct, pt, 3) >= fuzzyDedupThreshold {
				result[len(result)-1] = cur // keep newer
				continue
			}
		}
		result = append(result, cur)
	}
	return result
}

// collapseRepeatedAssistantMessages removes near-duplicate assistant messages
// across the entire history, keeping only the last in each cluster.
func collapseRepeatedAssistantMessages(messages []ai.Message) []ai.Message {
	type indexedMsg struct {
		origIdx int
		content string
	}
	var assistants []indexedMsg
	for i, m := range messages {
		if m.Role == "assistant" {
			t := messageTextContent(m)
			if strings.TrimSpace(t) != "" {
				assistants = append(assistants, indexedMsg{origIdx: i, content: t})
			}
		}
	}
	if len(assistants) < 3 {
		return messages
	}
	removeIdx := make(map[int]bool)
	used := make(map[int]bool)
	for i := 0; i < len(assistants); i++ {
		if used[i] {
			continue
		}
		group := []int{i}
		for j := i + 1; j < len(assistants); j++ {
			if used[j] {
				continue
			}
			if ngramSimilarityGo(assistants[i].content, assistants[j].content, 3) >= fuzzyDedupThreshold {
				group = append(group, j)
				used[j] = true
			}
		}
		if len(group) >= 2 {
			for k := 0; k < len(group)-1; k++ {
				removeIdx[assistants[group[k]].origIdx] = true
			}
		}
	}
	if len(removeIdx) == 0 {
		return messages
	}
	out := make([]ai.Message, 0, len(messages)-len(removeIdx))
	for i, m := range messages {
		if !removeIdx[i] {
			out = append(out, m)
		}
	}
	return out
}

// detectToxicHistoryMessages checks if the message list contains a cluster of
// near-duplicate assistant messages, indicating a repetition loop.
func detectToxicHistoryMessages(messages []ai.Message) bool {
	var assistantTexts []string
	for _, m := range messages {
		if m.Role == "assistant" {
			t := messageTextContent(m)
			if strings.TrimSpace(t) != "" {
				assistantTexts = append(assistantTexts, t)
			}
		}
	}
	if len(assistantTexts) < toxicMinAssistantCount {
		return false
	}
	maxCluster := 0
	visited := make(map[int]bool)
	for i := 0; i < len(assistantTexts); i++ {
		if visited[i] {
			continue
		}
		count := 1
		for j := i + 1; j < len(assistantTexts); j++ {
			if visited[j] {
				continue
			}
			if ngramSimilarityGo(assistantTexts[i], assistantTexts[j], 3) >= fuzzyDedupThreshold {
				count++
				visited[j] = true
			}
		}
		if count > maxCluster {
			maxCluster = count
		}
	}
	return maxCluster >= toxicClusterMinSize
}

// isRepetitionOfRecentMessages checks if newText is too similar to any of the
// last N assistant messages in history.
func isRepetitionOfRecentMessages(newText string, messages []ai.Message) bool {
	if strings.TrimSpace(newText) == "" {
		return false
	}
	var recent []string
	for i := len(messages) - 1; i >= 0 && len(recent) < repetitionCheckWindow; i-- {
		if messages[i].Role == "assistant" {
			t := messageTextContent(messages[i])
			if strings.TrimSpace(t) != "" {
				recent = append(recent, t)
			}
		}
	}
	for _, t := range recent {
		if ngramSimilarityGo(newText, t, 3) >= repetitionSimilarityThreshold {
			return true
		}
	}
	return false
}

const toxicHistoryRecoveryPrompt = "【紧急：对话循环修复】\n" +
	"系统检测到你之前的回复陷入了重复循环（多次输出几乎相同的内容）。\n" +
	"请立即停止重复之前的内容，完全忽略历史中的重复回复模式。\n" +
	"现在请只关注用户最新发送的这条消息，用全新的思路直接回答。\n" +
	"严禁复制或改写之前的任何回复。"

const antiRepetitionInjection = "【系统强制指令】\n" +
	"你上一次的回复与之前的历史回复高度雷同，已被系统拦截。\n" +
	"你必须用完全不同的表达方式、不同的角度、不同的句式来回应用户。\n" +
	"禁止复制、改写、转述之前任何一条回复的内容。\n" +
	"直接针对用户最新消息，给出全新的回应。"
