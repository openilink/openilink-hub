package bot

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/builtin"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

// deliverToApps dispatches a message to matching App installations.
func (m *Manager) deliverToApps(inst *Instance, msg provider.InboundMessage, p parsedMessage, tracer *store.Tracer, rootSpan *store.SpanBuilder) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("deliverToApps panic", "bot", inst.DBID, "err", r)
		}
	}()

	content := p.content

	// Check for @handle mention → route to specific app installation
	if m.tryDeliverMention(inst, msg, p, content, tracer, rootSpan) {
		return
	}

	// Check for slash command: /command args
	if m.tryDeliverCommand(inst, msg, p, content, tracer, rootSpan) {
		return
	}

	// Deliver as generic event to subscribed apps
	eventType := "message." + p.msgType
	installations, err := m.appDisp.MatchEvent(inst.DBID, eventType)
	if err != nil {
		rootSpan.AddEvent("match_event_error", map[string]any{"error": err.Error()})
		return
	}

	if len(installations) == 0 {
		rootSpan.AddEvent("match_event_none", map[string]any{"event_type": eventType})
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
	event.TraceID = tracer.TraceID()

	for i := range installations {
		// Builtin apps with handler: route to internal handler
		if installations[i].AppRegistry == "builtin" {
			if h := builtin.Get(installations[i].AppSlug); h != nil {
				span := tracer.StartChild(rootSpan, "builtin:"+installations[i].AppSlug, store.SpanKindInternal, map[string]any{
					"app.name": installations[i].AppName,
					"app.slug": installations[i].AppSlug,
				})
				if err := h.HandleEvent(&installations[i], event); err != nil {
					span.EndWithError(err.Error())
				} else {
					span.End()
				}
				continue
			}
			// No builtin handler — fall through to WebSocket/webhook delivery
		}

		// Check for active WebSocket connection — deliver via WS if connected
		// Try installation-level WS first, then app-level WS
		if m.appWSHub != nil {
			wsConn := m.appWSHub.Get(installations[i].ID)
			if wsConn == nil {
				wsConn = m.appWSHub.GetAppLevel(installations[i].AppID)
			}
			if wsConn != nil {
				span := tracer.StartChild(rootSpan, "ws:"+installations[i].AppSlug, store.SpanKindClient, map[string]any{
					"app.name":    installations[i].AppName,
					"app.slug":    installations[i].AppSlug,
					"delivery":    "websocket",
				})
				envelope := map[string]any{
					"type":            "event",
					"v":               1,
					"trace_id":        event.TraceID,
					"installation_id": installations[i].ID,
					"bot":             map[string]string{"id": installations[i].BotID},
					"event": map[string]any{
						"type":      event.Type,
						"id":        event.ID,
						"timestamp": event.Timestamp,
						"data":      event.Data,
					},
				}
				if err := wsConn.SendJSON(envelope); err != nil {
					span.EndWithError("ws send: " + err.Error())
					// Fall through to webhook delivery below
				} else {
					span.End()
					continue
				}
			}
		}

		// Webhook delivery (fallback when no WS connection)
		span := tracer.StartChild(rootSpan, "POST "+installations[i].AppWebhookURL, store.SpanKindClient, map[string]any{
			"app.name":    installations[i].AppName,
			"app.slug":    installations[i].AppSlug,
			"http.url":    installations[i].AppWebhookURL,
			"http.method": "POST",
		})
		result := m.appDisp.DeliverWithRetry(&installations[i], event)
		if result != nil {
			reply := result.Reply
			if result.ReplyURL != "" {
				reply = "[media] " + result.ReplyURL
			}
			span.SetAttr("http.status_code", result.StatusCode)
			span.SetAttr("http.response_body", reply)
			span.End()
		} else {
			span.EndWithError("no result")
		}
		m.sendAppResult(inst, msg.Sender, result, tracer, rootSpan)
	}
}

// tryDeliverMention checks if the message starts with @handle and routes to that installation.
func (m *Manager) tryDeliverMention(inst *Instance, msg provider.InboundMessage, p parsedMessage, content string, tracer *store.Tracer, rootSpan *store.SpanBuilder) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "@") {
		return false
	}
	parts := strings.SplitN(trimmed[1:], " ", 2)
	handle := strings.ToLower(parts[0])
	if handle == "" {
		return false
	}

	text := ""
	if len(parts) > 1 {
		text = strings.TrimSpace(parts[1])
	}

	installation, err := m.appDisp.Store.GetInstallationByHandle(inst.DBID, handle)
	if err != nil || installation == nil || !installation.Enabled {
		rootSpan.AddEvent("match_handle_miss", map[string]any{"handle": handle})
		return false
	}

	rootSpan.AddEvent("match_handle", map[string]any{"handle": handle, "app.name": installation.AppName})

	if strings.HasPrefix(text, "/") {
		cmdParts := strings.SplitN(text[1:], " ", 2)
		command := strings.ToLower(cmdParts[0])
		cmdArgs := ""
		if len(cmdParts) > 1 {
			cmdArgs = strings.TrimSpace(cmdParts[1])
		}
		event := appdelivery.NewEvent("command", map[string]any{
			"command": command, "text": cmdArgs,
			"sender": map[string]any{"id": msg.Sender, "name": msg.Sender},
			"group": groupInfo(msg), "handle": handle,
		})
		event.TraceID = tracer.TraceID()

		// Builtin apps: route to internal handler
		if installation.AppRegistry == "builtin" {
			m.deliverBuiltinMention(inst, installation, event, msg.Sender, tracer, rootSpan)
			return true
		}

		m.deliverToInstallation(inst, installation, event, msg.Sender, tracer, rootSpan)
		return true
	}

	event := appdelivery.NewEvent("message.text", map[string]any{
		"sender": map[string]any{"id": msg.Sender, "name": msg.Sender},
		"group": groupInfo(msg), "content": text, "handle": handle,
	})
	event.TraceID = tracer.TraceID()

	// Builtin apps: route to internal handler
	if installation.AppRegistry == "builtin" {
		m.deliverBuiltinMention(inst, installation, event, msg.Sender, tracer, rootSpan)
		return true
	}

	m.deliverToInstallation(inst, installation, event, msg.Sender, tracer, rootSpan)
	return true
}

// deliverToInstallation delivers an event to a single non-builtin installation,
// trying WebSocket first and falling back to webhook.
func (m *Manager) deliverToInstallation(inst *Instance, installation *store.AppInstallation, event *appdelivery.Event, sender string, tracer *store.Tracer, rootSpan *store.SpanBuilder) {
	// Try WebSocket delivery first (installation-level, then app-level)
	if m.appWSHub != nil {
		wsConn := m.appWSHub.Get(installation.ID)
		if wsConn == nil {
			wsConn = m.appWSHub.GetAppLevel(installation.AppID)
		}
		if wsConn != nil {
			span := tracer.StartChild(rootSpan, "ws:"+installation.AppSlug, store.SpanKindClient, map[string]any{
				"app.name": installation.AppName,
				"app.slug": installation.AppSlug,
				"delivery": "websocket",
			})
			envelope := map[string]any{
				"type":            "event",
				"v":               1,
				"trace_id":        event.TraceID,
				"installation_id": installation.ID,
				"bot":             map[string]string{"id": installation.BotID},
				"event": map[string]any{
					"type":      event.Type,
					"id":        event.ID,
					"timestamp": event.Timestamp,
					"data":      event.Data,
				},
			}
			if err := wsConn.SendJSON(envelope); err != nil {
				span.EndWithError("ws send: " + err.Error())
				// Fall through to webhook
			} else {
				span.End()
				return
			}
		}
	}

	// Webhook delivery (fallback)
	span := tracer.StartChild(rootSpan, "POST "+installation.AppWebhookURL, store.SpanKindClient, map[string]any{
		"app.name":    installation.AppName,
		"app.slug":    installation.AppSlug,
		"http.url":    installation.AppWebhookURL,
		"http.method": "POST",
	})
	result := m.appDisp.DeliverWithRetry(installation, event)
	if result != nil {
		span.SetAttr("http.status_code", result.StatusCode)
		span.SetAttr("http.response_body", result.Reply)
		span.End()
	} else {
		span.EndWithError("no result")
	}
	m.sendAppResult(inst, sender, result, tracer, rootSpan)
}

// deliverBuiltinMention handles event delivery for builtin apps via mention.
// If no builtin handler exists, falls through to WebSocket/webhook delivery.
func (m *Manager) deliverBuiltinMention(inst *Instance, installation *store.AppInstallation, event *appdelivery.Event, sender string, tracer *store.Tracer, rootSpan *store.SpanBuilder) {
	if h := builtin.Get(installation.AppSlug); h != nil {
		span := tracer.StartChild(rootSpan, "builtin:"+installation.AppSlug, store.SpanKindInternal, map[string]any{
			"app.name": installation.AppName,
			"app.slug": installation.AppSlug,
		})
		if err := h.HandleEvent(installation, event); err != nil {
			span.EndWithError(err.Error())
		} else {
			span.End()
		}
		return
	}
	// No builtin handler — deliver via WebSocket/webhook
	m.deliverToInstallation(inst, installation, event, sender, tracer, rootSpan)
}

// tryDeliverCommand checks if the message is a /command and delivers it.
func (m *Manager) tryDeliverCommand(inst *Instance, msg provider.InboundMessage, p parsedMessage, content string, tracer *store.Tracer, rootSpan *store.SpanBuilder) bool {
	installations, command, args, err := m.appDisp.MatchCommand(inst.DBID, content)
	if err != nil {
		rootSpan.AddEvent("match_command_error", map[string]any{"error": err.Error()})
		return false
	}
	if len(installations) == 0 {
		return false
	}

	rootSpan.AddEvent("match_command", map[string]any{
		"command": command,
		"apps":    fmt.Sprintf("%d", len(installations)),
		"args":    args,
	})

	event := appdelivery.NewEvent("command", map[string]any{
		"command": command, "text": args,
		"sender": map[string]any{"id": msg.Sender, "name": msg.Sender},
		"group": groupInfo(msg),
	})
	event.TraceID = tracer.TraceID()

	for i := range installations {
		// Builtin apps with handler: route to internal handler
		if installations[i].AppRegistry == "builtin" {
			if h := builtin.Get(installations[i].AppSlug); h != nil {
				span := tracer.StartChild(rootSpan, "builtin:"+installations[i].AppSlug, store.SpanKindInternal, map[string]any{
					"app.name": installations[i].AppName,
					"app.slug": installations[i].AppSlug,
				})
				if err := h.HandleEvent(&installations[i], event); err != nil {
					span.EndWithError(err.Error())
				} else {
					span.End()
				}
				continue
			}
			// No builtin handler — fall through to WebSocket/webhook delivery
		}

		m.deliverToInstallation(inst, &installations[i], event, msg.Sender, tracer, rootSpan)
	}
	return true
}

// sendAppResult sends a reply from an App via the bot and stores it as outbound.
func (m *Manager) sendAppResult(inst *Instance, to string, result *appdelivery.DeliveryResult, tracer *store.Tracer, rootSpan *store.SpanBuilder) {
	if result == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	contextToken := m.store.GetLatestContextToken(inst.DBID)

	switch result.ReplyType {
	case "image", "video", "file", "voice":
		span := tracer.StartChild(rootSpan, "send_reply", store.SpanKindClient, map[string]any{
			"reply.type": result.ReplyType,
			"reply.to":   to,
		})
		mediaKey := m.sendAppMedia(ctx, inst, to, contextToken, result)
		span.SetAttr("reply.content", result.ReplyName)
		if mediaKey != "" {
			span.SetAttr("reply.media_key", mediaKey)
		}
		span.End()
	default:
		if result.Reply == "" {
			return
		}
		span := tracer.StartChild(rootSpan, "send_reply", store.SpanKindClient, map[string]any{
			"reply.type":    "text",
			"reply.to":      to,
			"reply.content": result.Reply,
		})
		m.sendAppText(ctx, inst, to, contextToken, result.Reply)
		span.End()
	}
}

func (m *Manager) sendAppText(ctx context.Context, inst *Instance, to, contextToken, text string) {
	clientID, err := inst.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: to, Text: text, ContextToken: contextToken,
	})
	if err != nil {
		slog.Error("app reply send failed", "bot", inst.DBID, "to", to, "err", err)
		return
	}
	slog.Info("app reply sent", "bot", inst.DBID, "to", to, "client_id", clientID)

	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": text}})
	m.store.SaveMessage(&store.Message{
		BotID: inst.DBID, Direction: "outbound", ToUserID: to, MessageType: 2, ItemList: itemList,
	})
}

func (m *Manager) sendAppMedia(ctx context.Context, inst *Instance, to, contextToken string, result *appdelivery.DeliveryResult) string {
	var data []byte
	var err error

	if result.ReplyBase64 != "" {
		// Decode base64 (supports data URI prefix: data:image/png;base64,...)
		b64, mime := parseBase64(result.ReplyBase64)
		if mime != "" && result.ReplyName == "" {
			result.ReplyName = fileNameFromMIME(mime)
		}
		data, err = base64.StdEncoding.DecodeString(b64)
		if err != nil {
			slog.Error("app media base64 decode failed", "err", err)
			if result.Reply != "" {
				m.sendAppText(ctx, inst, to, contextToken, result.Reply)
			}
			return ""
		}
	} else if result.ReplyURL != "" {
		// Download media from URL
		dlCtx, dlCancel := context.WithTimeout(ctx, 8*time.Second)
		defer dlCancel()

		req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, result.ReplyURL, nil)
		if err != nil {
			slog.Error("app media download: bad url", "url", result.ReplyURL, "err", err)
			return ""
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			slog.Error("app media download failed", "url", result.ReplyURL, "err", err)
			return ""
		}
		defer resp.Body.Close()

		data, err = io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024)) // 20MB max
		if err != nil {
			slog.Error("app media read failed", "err", err)
			return ""
		}
		// Extract filename from Content-Type if not provided
		if result.ReplyName == "" {
			if ct := resp.Header.Get("Content-Type"); ct != "" {
				mime := strings.SplitN(ct, ";", 2)[0]
				mime = strings.TrimSpace(mime)
				result.ReplyName = fileNameFromMIME(mime)
			}
		}
	} else {
		// No media source, send text fallback
		if result.Reply != "" {
			m.sendAppText(ctx, inst, to, contextToken, result.Reply)
		}
		return ""
	}

	fileName := result.ReplyName
	if fileName == "" {
		// Auto-generate filename with correct extension based on type
		switch result.ReplyType {
		case "image":
			fileName = "image.png"
			// Detect actual format from data header
			if len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 {
				fileName = "image.jpg"
			} else if len(data) > 4 && string(data[:4]) == "GIF8" {
				fileName = "image.gif"
			} else if len(data) > 8 && string(data[1:4]) == "PNG" {
				fileName = "image.png"
			}
		case "video":
			fileName = "video.mp4"
		default:
			fileName = "file"
		}
	}

	clientID, err := inst.Provider.Send(ctx, provider.OutboundMessage{
		Recipient: to, ContextToken: contextToken, Data: data, FileName: fileName,
	})
	if err != nil {
		slog.Error("app media send failed", "bot", inst.DBID, "to", to, "err", err)
		return ""
	}
	slog.Info("app media sent", "bot", inst.DBID, "to", to, "type", result.ReplyType, "size", len(data), "client_id", clientID)

	mediaStatus := ""
	mediaKeys := json.RawMessage(`{}`)
	storageKey := ""
	if len(data) > 0 && m.storage != nil {
		ct := detectOutboundContentType(result.ReplyType)
		ext := detectOutboundExt(fileName, result.ReplyType)
		now := time.Now()
		var rnd [4]byte
		rand.Read(rnd[:])
		key := fmt.Sprintf("%s/%s/out_%d_%x%s", inst.DBID,
			now.Format("2006/01/02"), now.UnixMilli(), rnd, ext)
		if _, err := m.storage.Put(ctx, key, ct, data); err == nil {
			mediaStatus = "ready"
			mediaKeys, _ = json.Marshal(map[string]string{"0": key})
			storageKey = key
		} else {
			slog.Warn("app media: objectstore put failed", "key", key, "err", err)
		}
	}

	itemType := result.ReplyType
	itemList, _ := json.Marshal([]map[string]any{{"type": itemType, "file_name": fileName}})
	m.store.SaveMessage(&store.Message{
		BotID: inst.DBID, Direction: "outbound", ToUserID: to, MessageType: 2, ItemList: itemList,
		MediaStatus: mediaStatus, MediaKeys: mediaKeys,
	})

	return storageKey
}

// parseBase64 extracts pure base64 and MIME type from a string that may be
// a data URI (data:image/png;base64,iVBOR...) or plain base64.
func parseBase64(s string) (b64, mime string) {
	if strings.HasPrefix(s, "data:") {
		// data:image/png;base64,iVBOR...
		commaIdx := strings.Index(s, ",")
		if commaIdx > 0 {
			header := s[5:commaIdx] // "image/png;base64"
			b64 = s[commaIdx+1:]
			semicolonIdx := strings.Index(header, ";")
			if semicolonIdx > 0 {
				mime = header[:semicolonIdx]
			} else {
				mime = header
			}
			return
		}
	}
	return s, ""
}

// fileNameFromMIME returns a default filename for a MIME type.
func fileNameFromMIME(mime string) string {
	switch mime {
	case "image/png":
		return "image.png"
	case "image/jpeg":
		return "image.jpg"
	case "image/gif":
		return "image.gif"
	case "image/webp":
		return "image.webp"
	case "video/mp4":
		return "video.mp4"
	case "audio/mpeg":
		return "audio.mp3"
	case "application/pdf":
		return "file.pdf"
	default:
		if strings.HasPrefix(mime, "image/") {
			return "image." + strings.TrimPrefix(mime, "image/")
		}
		if strings.HasPrefix(mime, "video/") {
			return "video." + strings.TrimPrefix(mime, "video/")
		}
		return "file"
	}
}

func groupInfo(msg provider.InboundMessage) any {
	if msg.GroupID == "" {
		return nil
	}
	return map[string]any{"id": msg.GroupID, "name": msg.GroupID}
}

func detectOutboundContentType(msgType string) string {
	switch msgType {
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	case "voice":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}

func detectOutboundExt(filename, msgType string) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}
	switch msgType {
	case "image":
		return ".jpg"
	case "video":
		return ".mp4"
	case "voice":
		return ".wav"
	default:
		return ""
	}
}
