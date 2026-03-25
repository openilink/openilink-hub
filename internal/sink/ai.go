package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/openilink/openilink-hub/internal/ai"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

const typingTimeout = 30 * time.Second

// AI calls an OpenAI-compatible chat completion API and sends the reply
// back through the bot. It also manages typing indicators.
type AI struct {
	Store store.Store
}

func (s *AI) Name() string { return "ai" }

func (s *AI) Handle(d Delivery) {
	if !d.Channel.AIConfig.Enabled || d.MsgType != "text" || d.Content == "" {
		return
	}
	s.reply(d)
}

func (s *AI) reply(d Delivery) {
	cfg := s.resolveConfig(d.Channel.AIConfig)
	if cfg.APIKey == "" {
		slog.Warn("ai reply skipped: no api key", "channel", d.Channel.ID, "source", d.Channel.AIConfig.Source)
		return
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

	reply, err := ai.Complete(ctx, cfg, s.Store, d.Channel.ID, sender, d.Content)

	if typingTicket != "" {
		d.Provider.SendTyping(ctx, sender, typingTicket, false)
	}

	if err != nil {
		slog.Error("ai completion failed", "channel", d.Channel.ID, "err", err)
		return
	}
	if reply == "" {
		return
	}

	_, err = d.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: sender,
		Text:      reply,
	})
	if err != nil {
		slog.Error("ai reply send failed", "channel", d.Channel.ID, "err", err)
		return
	}

	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": reply}})
	s.Store.SaveMessage(&store.Message{
		BotID:       d.BotDBID,
		Direction:   "outbound",
		ToUserID:    sender,
		MessageType: 2, // BOT
		ItemList:    itemList,
	})
}

func (s *AI) resolveConfig(cfg store.AIConfig) store.AIConfig {
	if cfg.Source != "builtin" {
		return cfg
	}
	global, _ := s.Store.ListConfigByPrefix("ai.")
	if global["ai.api_key"] == "" {
		return cfg
	}
	cfg.BaseURL = global["ai.base_url"]
	cfg.APIKey = global["ai.api_key"]
	cfg.Model = global["ai.model"]
	if cfg.SystemPrompt == "" {
		cfg.SystemPrompt = global["ai.system_prompt"]
	}
	if cfg.MaxHistory <= 0 {
		if v := global["ai.max_history"]; v != "" {
			fmt.Sscanf(v, "%d", &cfg.MaxHistory)
		}
	}
	return cfg
}
