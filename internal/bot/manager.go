package bot

import (
	"context"
	"encoding/json"
	"log/slog"
	"regexp"
	"strings"
	"sync"

	"github.com/openilink/openilink-hub/internal/ai"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
)

var mentionRe = regexp.MustCompile(`@(\S+)`)

// parseMentions extracts @handle mentions from text.
func parseMentions(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	var handles []string
	for _, m := range matches {
		handles = append(handles, m[1])
	}
	return handles
}

// Manager manages all active bot instances.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	db        *database.DB
	hub       *relay.Hub
}

func NewManager(db *database.DB, hub *relay.Hub) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		db:        db,
		hub:       hub,
	}
}

func (m *Manager) StartAll(ctx context.Context) {
	bots, err := m.db.GetAllBots()
	if err != nil {
		slog.Error("failed to load bots", "err", err)
		return
	}
	for _, b := range bots {
		if len(b.Credentials) == 0 || string(b.Credentials) == "{}" {
			continue
		}
		if err := m.StartBot(ctx, &b); err != nil {
			slog.Error("failed to start bot", "bot", b.ID, "err", err)
		}
	}
	slog.Info("started all bots", "count", len(bots))
}

func (m *Manager) StartBot(ctx context.Context, bot *database.Bot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.instances[bot.ID]; ok {
		old.Stop()
	}

	factory, ok := provider.Get(bot.Provider)
	if !ok {
		slog.Error("unknown provider", "provider", bot.Provider, "bot", bot.ID)
		return nil
	}

	p := factory()
	inst := NewInstance(bot.ID, p)

	err := p.Start(ctx, provider.StartOptions{
		Credentials: bot.Credentials,
		SyncState:   bot.SyncState,
		OnMessage: func(msg provider.InboundMessage) {
			m.onInbound(inst, msg)
		},
		OnStatus: func(status string) {
			_ = m.db.UpdateBotStatus(bot.ID, status)
			m.onStatusChange(inst, status)
		},
		OnSyncUpdate: func(state json.RawMessage) {
			_ = m.db.UpdateBotSyncState(bot.ID, state)
		},
	})
	if err != nil {
		return err
	}

	m.instances[bot.ID] = inst
	slog.Info("bot started", "bot", bot.ID, "provider", bot.Provider)
	return nil
}

func (m *Manager) StopBot(botDBID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[botDBID]; ok {
		inst.Stop()
		delete(m.instances, botDBID)
	}
}

func (m *Manager) GetInstance(botDBID string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[botDBID]
	return inst, ok
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		inst.Stop()
	}
	m.instances = make(map[string]*Instance)
}

// onStatusChange broadcasts bot status to all channels.
func (m *Manager) onStatusChange(inst *Instance, status string) {
	env := relay.NewEnvelope("bot_status", relay.BotStatusData{
		BotID:  inst.DBID,
		Status: status,
	})
	m.hub.Broadcast(inst.DBID, env)
}

// onInbound routes an inbound message with filtering.
func (m *Manager) onInbound(inst *Instance, msg provider.InboundMessage) {
	// Determine primary msg type and content for storage
	msgType := "text"
	content := ""
	for _, item := range msg.Items {
		switch item.Type {
		case "text":
			content = item.Text
		case "image", "voice", "file", "video":
			msgType = item.Type
			if content == "" {
				if item.Text != "" {
					content = item.Text
				} else if item.FileName != "" {
					content = item.FileName
				} else {
					content = "[" + item.Type + "]"
				}
			}
		}
	}

	payloadMap := map[string]any{"content": content}
	if msg.GroupID != "" {
		payloadMap["group_id"] = msg.GroupID
	}
	if msg.ContextToken != "" {
		payloadMap["context_token"] = msg.ContextToken
	}
	payload, _ := json.Marshal(payloadMap)

	dbMsg := &database.Message{
		BotID:     inst.DBID,
		Direction: "inbound",
		Sender:    msg.Sender,
		Recipient: msg.Recipient,
		MsgType:   msgType,
		Payload:   payload,
	}
	seqID, _ := m.db.SaveMessage(dbMsg)
	_ = m.db.IncrBotMsgCount(inst.DBID)

	// Build relay envelope
	items := make([]relay.MessageItem, len(msg.Items))
	for i, item := range msg.Items {
		items[i] = convertRelayItem(item)
	}

	env := relay.NewEnvelope("message", relay.MessageData{
		SeqID:        seqID,
		ExternalID:   msg.ExternalID,
		Sender:       msg.Sender,
		Recipient:    msg.Recipient,
		GroupID:      msg.GroupID,
		Timestamp:    msg.Timestamp,
		MessageState: msg.MessageState,
		Items:        items,
		ContextToken: msg.ContextToken,
		SessionID:    msg.SessionID,
	})

	// Load channels and route
	channels, err := m.db.ListChannelsByBot(inst.DBID)
	if err != nil {
		slog.Error("load channels failed", "bot", inst.DBID, "err", err)
		return
	}

	// If text contains @handle mentions, route only to mentioned channels
	mentioned := parseMentions(content)
	var matched []database.Channel
	if len(mentioned) > 0 {
		handleSet := make(map[string]bool, len(mentioned))
		for _, h := range mentioned {
			handleSet[strings.ToLower(h)] = true
		}
		for _, ch := range channels {
			if ch.Handle != "" && handleSet[strings.ToLower(ch.Handle)] {
				matched = append(matched, ch)
			}
		}
	} else {
		// No @mention — use regular filter rules
		for _, ch := range channels {
			if matchFilter(ch.FilterRule, msg.Sender, content, msgType) {
				matched = append(matched, ch)
			}
		}
	}

	// Deliver to matched channels + trigger AI auto-reply
	for _, ch := range matched {
		m.hub.SendTo(ch.ID, env)
		_ = m.db.UpdateChannelLastSeq(ch.ID, seqID)

		// AI auto-reply (only for text messages)
		if ch.AIConfig.Enabled && msgType == "text" && content != "" {
			go m.aiReply(inst, ch, msg.Sender, content)
		}
	}
}

// resolveAIConfig merges channel config with global builtin config if source is "builtin".
func (m *Manager) resolveAIConfig(cfg database.AIConfig) database.AIConfig {
	if cfg.Source != "builtin" {
		return cfg
	}
	// Load global AI config from system_config
	global, _ := m.db.ListConfigByPrefix("ai.")
	if global["ai.api_key"] == "" {
		return cfg // no global config, can't resolve
	}
	cfg.BaseURL = global["ai.base_url"]
	cfg.APIKey = global["ai.api_key"]
	cfg.Model = global["ai.model"]
	return cfg
}

// aiReply calls the AI completion API and sends the reply through the bot.
func (m *Manager) aiReply(inst *Instance, ch database.Channel, sender, text string) {
	resolved := m.resolveAIConfig(ch.AIConfig)
	if resolved.APIKey == "" {
		slog.Warn("ai reply skipped: no api key", "channel", ch.ID, "source", ch.AIConfig.Source)
		return
	}
	reply, err := ai.Complete(context.Background(), resolved, m.db, inst.DBID, sender, text)
	if err != nil {
		slog.Error("ai completion failed", "channel", ch.ID, "err", err)
		return
	}
	if reply == "" {
		return
	}

	// Send reply through bot
	_, err = inst.Send(context.Background(), provider.OutboundMessage{
		Recipient: sender,
		Text:      reply,
	})
	if err != nil {
		slog.Error("ai reply send failed", "channel", ch.ID, "err", err)
		return
	}

	// Save outbound message
	chID := ch.ID
	payload, _ := json.Marshal(map[string]string{"content": reply})
	m.db.SaveMessage(&database.Message{
		BotID:     inst.DBID,
		ChannelID: &chID,
		Direction: "outbound",
		Sender:    "",
		Recipient: sender,
		MsgType:   "text",
		Payload:   payload,
	})
}

// matchFilter checks if a message passes the channel's filter rule.
func matchFilter(rule database.FilterRule, sender, text, msgType string) bool {
	if len(rule.UserIDs) > 0 {
		found := false
		for _, uid := range rule.UserIDs {
			if uid == sender {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.MessageTypes) > 0 {
		found := false
		for _, mt := range rule.MessageTypes {
			if mt == msgType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.Keywords) > 0 {
		found := false
		lower := strings.ToLower(text)
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func convertRelayItem(item provider.MessageItem) relay.MessageItem {
	ri := relay.MessageItem{
		Type:     item.Type,
		Text:     item.Text,
		FileName: item.FileName,
	}
	if item.Media != nil {
		ri.Media = &relay.Media{
			URL:         item.Media.URL,
			AESKey:      item.Media.AESKey,
			FileSize:    item.Media.FileSize,
			MediaType:   item.Media.MediaType,
			PlayTime:    item.Media.PlayTime,
			PlayLength:  item.Media.PlayLength,
			ThumbWidth:  item.Media.ThumbWidth,
			ThumbHeight: item.Media.ThumbHeight,
		}
	}
	if item.RefMsg != nil {
		refItem := convertRelayItem(item.RefMsg.Item)
		ri.RefMsg = &relay.RefMsg{
			Title: item.RefMsg.Title,
			Item:  refItem,
		}
	}
	return ri
}
