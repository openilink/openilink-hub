package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	apiKey := r.URL.Query().Get("key")
	if apiKey == "" {
		http.Error(w, `{"error":"api key required"}`, http.StatusUnauthorized)
		return
	}

	ch, err := s.DB.GetChannelByAPIKey(apiKey)
	if err != nil || !ch.Enabled {
		http.Error(w, `{"error":"invalid or disabled key"}`, http.StatusUnauthorized)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}

	conn := relay.NewConn(ch.ID, ch.BotID, ws, s.Hub)
	s.Hub.Register(conn)

	// Send init message
	botStatus := "disconnected"
	if inst, ok := s.BotManager.GetInstance(ch.BotID); ok {
		botStatus = inst.Status()
	}
	conn.Send(relay.NewEnvelope("init", relay.InitData{
		ChannelID:   ch.ID,
		ChannelName: ch.Name,
		BotID:       ch.BotID,
		BotStatus:   botStatus,
	}))

	// Replay missed messages since last_seq
	if ch.LastSeq > 0 {
		missed, err := s.DB.GetMessagesSince(ch.BotID, ch.LastSeq, 100)
		if err == nil && len(missed) > 0 {
			for _, m := range missed {
				ts := m.CreatedAt * 1000
				if m.CreateTimeMs != nil {
					ts = *m.CreateTimeMs
				}
				env := relay.NewEnvelope("message", relay.MessageData{
					SeqID:     m.ID,
					Sender:    m.FromUserID,
					Timestamp: ts,
					Items:     parseRelayItems(m.ItemList, ch.APIKey),
				})
				conn.Send(env)
			}
			if err := s.DB.UpdateChannelLastSeq(ch.ID, missed[len(missed)-1].ID); err != nil {
				slog.Error("update channel last_seq failed", "channel", ch.ID, "err", err)
			}
		}
	}

	go conn.WritePump()
	conn.ReadPump() // blocks
}

// SetupUpstreamHandler creates the handler for messages from channel clients.
func (s *Server) SetupUpstreamHandler() relay.UpstreamHandler {
	return func(conn *relay.Conn, env relay.Envelope) {
		switch env.Type {
		case "send_text":
			var data relay.SendTextData
			if err := json.Unmarshal(env.Data, &data); err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", "invalid data"))
				return
			}

			inst, ok := s.BotManager.GetInstance(conn.BotID)
			if !ok {
				conn.Send(relay.NewAck(env.ReqID, false, "", "bot not connected"))
				return
			}

			ctxToken := s.DB.GetLatestContextToken(conn.BotID)
			clientID, err := inst.Send(context.Background(), provider.OutboundMessage{
				Recipient:    data.Recipient,
				Text:         data.Text,
				ContextToken: ctxToken,
			})
			if err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", err.Error()))
				return
			}

			itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": data.Text}})
			s.DB.SaveMessage(&database.Message{
				BotID:       conn.BotID,
				Direction:   "outbound",
				ToUserID:    data.Recipient,
				MessageType: 2,
				ItemList:    itemList,
			})
			conn.Send(relay.NewAck(env.ReqID, true, clientID, ""))

		case "send_typing":
			var data relay.SendTypingData
			if err := json.Unmarshal(env.Data, &data); err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", "invalid data"))
				return
			}

			inst, ok := s.BotManager.GetInstance(conn.BotID)
			if !ok {
				conn.Send(relay.NewAck(env.ReqID, false, "", "bot not connected"))
				return
			}

			typing := data.Status != "cancel"
			if err := inst.Provider.SendTyping(context.Background(), "", data.Ticket, typing); err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", err.Error()))
				return
			}
			conn.Send(relay.NewAck(env.ReqID, true, "", ""))

		case "get_config":
			var data struct {
				ContextToken string `json:"context_token"`
			}
			if err := json.Unmarshal(env.Data, &data); err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", "invalid data"))
				return
			}

			inst, ok := s.BotManager.GetInstance(conn.BotID)
			if !ok {
				conn.Send(relay.NewAck(env.ReqID, false, "", "bot not connected"))
				return
			}

			cfg, err := inst.Provider.GetConfig(context.Background(), "", data.ContextToken)
			if err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", err.Error()))
				return
			}
			conn.Send(relay.NewEnvelope("config", cfg))

		default:
			conn.Send(relay.NewEnvelope("error", relay.ErrorData{
				Code: "unknown_type", Message: "unknown message type: " + env.Type,
			}))
		}
	}
}
