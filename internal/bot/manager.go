package bot

import (
	"context"
	"log/slog"
	"sync"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/relay"
)

// Manager manages all active bot instances.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance // keyed by bot DB ID
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

// StartAll loads all bots from DB and starts monitoring.
func (m *Manager) StartAll(ctx context.Context) {
	bots, err := m.db.GetAllBots()
	if err != nil {
		slog.Error("failed to load bots", "err", err)
		return
	}
	for _, b := range bots {
		if b.BotToken == "" {
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

	// Stop existing instance if any
	if old, ok := m.instances[bot.ID]; ok {
		old.Stop()
	}

	inst := NewInstance(bot)
	inst.Start(ctx, m.db, m.onInbound)
	m.instances[bot.ID] = inst
	slog.Info("bot started", "bot", bot.ID, "ilink_bot_id", bot.BotID)
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

// onInbound routes an inbound message to all matching sub-level WebSocket clients.
func (m *Manager) onInbound(inst *Instance, msg ilink.WeixinMessage) {
	text := ilink.ExtractText(&msg)

	// Log to DB
	content := text
	msgType := 1
	for _, item := range msg.ItemList {
		switch item.Type {
		case ilink.ItemImage:
			msgType = 2
			if content == "" {
				content = "[image]"
			}
		case ilink.ItemVoice:
			msgType = 3
			if item.VoiceItem != nil && item.VoiceItem.Text != "" {
				content = item.VoiceItem.Text
			} else if content == "" {
				content = "[voice]"
			}
		case ilink.ItemFile:
			msgType = 4
			if item.FileItem != nil {
				content = item.FileItem.FileName
			}
		case ilink.ItemVideo:
			msgType = 5
			if content == "" {
				content = "[video]"
			}
		}
	}
	_ = m.db.SaveMessage(inst.DBID, "inbound", msg.FromUserID, msgType, content, nil)

	// Build relay envelope
	var items []relay.MessageItem
	for _, item := range msg.ItemList {
		switch item.Type {
		case ilink.ItemText:
			if item.TextItem != nil {
				items = append(items, relay.MessageItem{Type: "text", Text: item.TextItem.Text})
			}
		case ilink.ItemImage:
			items = append(items, relay.MessageItem{Type: "image"})
		case ilink.ItemVoice:
			mi := relay.MessageItem{Type: "voice"}
			if item.VoiceItem != nil {
				mi.Text = item.VoiceItem.Text
			}
			items = append(items, mi)
		case ilink.ItemFile:
			mi := relay.MessageItem{Type: "file"}
			if item.FileItem != nil {
				mi.FileName = item.FileItem.FileName
			}
			items = append(items, mi)
		case ilink.ItemVideo:
			items = append(items, relay.MessageItem{Type: "video"})
		}
	}

	env := relay.NewEnvelope("message", relay.MessageData{
		MessageID:    msg.MessageID,
		FromUserID:   msg.FromUserID,
		Timestamp:    msg.CreateTimeMs,
		Items:        items,
		ContextToken: msg.ContextToken,
		SessionID:    msg.SessionID,
	})

	m.hub.Broadcast(inst.DBID, env)
}
