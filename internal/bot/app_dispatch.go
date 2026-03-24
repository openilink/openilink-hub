package bot

import (
	"context"
	"log/slog"
	"strings"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
)

// deliverToApps dispatches a message to matching App installations.
// It handles both slash commands and event subscriptions.
func (m *Manager) deliverToApps(inst *Instance, msg provider.InboundMessage, p parsedMessage) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("deliverToApps panic", "bot", inst.DBID, "err", r)
		}
	}()

	content := p.content
	slog.Debug("deliverToApps", "bot", inst.DBID, "content", content, "msg_type", p.msgType)

	// Check for @handle mention → route to specific app installation
	if m.tryDeliverMention(inst, msg, p, content) {
		return
	}

	// Check for slash command: /command args
	if m.tryDeliverCommand(inst, msg, p, content) {
		return
	}

	// Deliver as generic event to subscribed apps
	eventType := "message." + p.msgType
	installations, err := m.appDisp.MatchEvent(inst.DBID, eventType)
	if err != nil {
		slog.Error("app match event failed", "bot", inst.DBID, "err", err)
		return
	}

	if len(installations) == 0 {
		return
	}

	event := appdelivery.NewEvent(eventType, map[string]any{
		"message_id": msg.ExternalID,
		"sender":     map[string]any{"id": msg.Sender, "name": msg.Sender},
		"group":      groupInfo(msg),
		"content":    content,
		"msg_type":   p.msgType,
		"items":      p.relayItems,
	})

	for i := range installations {
		result := m.appDisp.DeliverWithRetry(&installations[i], event)
		if result != nil && result.Reply != "" {
			m.sendAppReply(inst, msg.Sender, result.Reply)
		}
	}
}

// tryDeliverMention checks if the message starts with @handle and routes to that installation.
func (m *Manager) tryDeliverMention(inst *Instance, msg provider.InboundMessage, p parsedMessage, content string) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "@") {
		return false
	}
	// Extract handle: @echo-work hello → handle="echo-work", text="hello"
	parts := strings.SplitN(trimmed[1:], " ", 2)
	handle := strings.ToLower(parts[0])
	if handle == "" {
		return false
	}

	text := ""
	if len(parts) > 1 {
		text = strings.TrimSpace(parts[1])
	}

	installation, err := m.appDisp.DB.GetInstallationByHandle(inst.DBID, handle)
	if err != nil || installation == nil || !installation.Enabled || installation.RequestURL == "" {
		return false
	}

	// @handle /command args → deliver as command to this specific installation
	if strings.HasPrefix(text, "/") {
		cmdParts := strings.SplitN(text[1:], " ", 2)
		command := "/" + strings.ToLower(cmdParts[0])
		cmdArgs := ""
		if len(cmdParts) > 1 {
			cmdArgs = strings.TrimSpace(cmdParts[1])
		}
		event := appdelivery.NewEvent("command", map[string]any{
			"command": command,
			"text":    cmdArgs,
			"sender":  map[string]any{"id": msg.Sender, "name": msg.Sender},
			"group":   groupInfo(msg),
			"handle":  handle,
		})
		result := m.appDisp.DeliverWithRetry(installation, event)
		if result != nil && result.Reply != "" {
			m.sendAppReply(inst, msg.Sender, result.Reply)
		}
		return true
	}

	// @handle text → deliver as message to this specific installation
	event := appdelivery.NewEvent("message.text", map[string]any{
		"sender":  map[string]any{"id": msg.Sender, "name": msg.Sender},
		"group":   groupInfo(msg),
		"content": text,
		"handle":  handle,
	})

	result := m.appDisp.DeliverWithRetry(installation, event)
	if result != nil && result.Reply != "" {
		m.sendAppReply(inst, msg.Sender, result.Reply)
	}
	return true
}

// tryDeliverCommand checks if the message is a /command or @command and delivers it.
func (m *Manager) tryDeliverCommand(inst *Instance, msg provider.InboundMessage, p parsedMessage, content string) bool {
	installations, command, args, err := m.appDisp.MatchCommand(inst.DBID, content)
	if err != nil {
		slog.Error("app match command failed", "bot", inst.DBID, "err", err)
		return false
	}
	if len(installations) == 0 {
		return false
	}

	event := appdelivery.NewEvent("command", map[string]any{
		"command": "/" + command,
		"text":    args,
		"sender":  map[string]any{"id": msg.Sender, "name": msg.Sender},
		"group":   groupInfo(msg),
	})

	for i := range installations {
		result := m.appDisp.DeliverWithRetry(&installations[i], event)
		if result != nil && result.Reply != "" {
			m.sendAppReply(inst, msg.Sender, result.Reply)
		}
	}
	return true
}

// sendAppReply sends a text reply from an App via the bot.
func (m *Manager) sendAppReply(inst *Instance, to, text string) {
	if text == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientID, err := inst.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: to,
		Text:      text,
	})
	if err != nil {
		slog.Error("app reply send failed", "bot", inst.DBID, "to", to, "err", err)
		return
	}
	slog.Info("app reply sent", "bot", inst.DBID, "to", to, "client_id", clientID)
}

func groupInfo(msg provider.InboundMessage) any {
	if msg.GroupID == "" {
		return nil
	}
	return map[string]any{"id": msg.GroupID, "name": msg.GroupID}
}
