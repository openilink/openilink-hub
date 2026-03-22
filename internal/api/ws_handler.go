package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
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

	sub, err := s.DB.GetSublevelByAPIKey(apiKey)
	if err != nil || !sub.Enabled {
		http.Error(w, `{"error":"invalid or disabled key"}`, http.StatusUnauthorized)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}

	conn := relay.NewConn(sub.ID, sub.BotDBID, ws, s.Hub)
	s.Hub.Register(conn)

	go conn.WritePump()
	conn.ReadPump() // blocks
}

// SetupUpstreamHandler creates the handler for messages from sub-level clients.
func (s *Server) SetupUpstreamHandler() relay.UpstreamHandler {
	return func(conn *relay.Conn, env relay.Envelope) {
		switch env.Type {
		case "send_text":
			var data relay.SendTextData
			if err := json.Unmarshal(env.Data, &data); err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", "invalid data"))
				return
			}

			inst, ok := s.BotManager.GetInstance(conn.BotDBID)
			if !ok {
				conn.Send(relay.NewAck(env.ReqID, false, "", "bot not connected"))
				return
			}

			clientID, err := inst.Client.Push(context.Background(), data.ToUserID, data.Text)
			if err != nil {
				conn.Send(relay.NewAck(env.ReqID, false, "", err.Error()))
				return
			}

			// Log outbound
			sublevelID := conn.SublevelID
			_ = s.DB.SaveMessage(conn.BotDBID, "outbound", data.ToUserID, 1, data.Text, &sublevelID)
			conn.Send(relay.NewAck(env.ReqID, true, clientID, ""))

		default:
			conn.Send(relay.NewEnvelope("error", relay.ErrorData{
				Code: "unknown_type", Message: "unknown message type: " + env.Type,
			}))
		}
	}
}
