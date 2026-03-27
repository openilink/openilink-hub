package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

const typingTimeout = 30 * time.Second

// AI calls an OpenAI-compatible chat completion API and sends the reply
// back through the bot. Supports tool calling via installed App tools.
type AI struct {
	Store      store.Store
	AppDisp    *appdelivery.Dispatcher
}

func (s *AI) Name() string { return "ai" }

func (s *AI) Handle(d Delivery) {
	if !d.AIEnabled || d.MsgType != "text" || d.Content == "" {
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

	// Build messages and do initial completion
	messages := ai.BuildMessages(cfg, s.Store, d.Channel.ID, sender, d.Content)
	result, err := ai.Complete(ctx, cfg, s.Store, d.Channel.ID, sender, d.Content, tools)
	if err != nil {
		slog.Error("ai completion failed", "bot", d.BotDBID, "err", err)
		if span != nil {
			span.SetStatus(store.StatusError, err.Error())
			span.End()
		}
		s.stopTyping(d, typingTicket)
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

		// Accumulate token usage from this round
		if result.Usage != nil {
			totalPrompt += result.Usage.PromptTokens
			totalCompletion += result.Usage.CompletionTokens
			totalTokens += result.Usage.TotalTokens
			totalCached += result.Usage.CachedTokens
			totalReasoning += result.Usage.ReasoningTokens
		}
	}

	// Set token usage attributes on span
	if span != nil && totalTokens > 0 {
		span.SetAttr("ai.tokens.prompt", strconv.Itoa(totalPrompt))
		span.SetAttr("ai.tokens.completion", strconv.Itoa(totalCompletion))
		span.SetAttr("ai.tokens.total", strconv.Itoa(totalTokens))
		if totalCached > 0 {
			span.SetAttr("ai.tokens.cached", strconv.Itoa(totalCached))
		}
		if totalReasoning > 0 {
			span.SetAttr("ai.tokens.reasoning", strconv.Itoa(totalReasoning))
		}
	}

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
		span.End()
	}

	content := result.Reply
	if content == "" {
		content = fmt.Sprintf("tool returned HTTP %d with no content", result.StatusCode)
	}
	return ai.ToolCallResult{ID: tc.ID, Name: tc.Name, Content: content}
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
	cfg.HideThinking = global["ai.hide_thinking"] == "true"
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
