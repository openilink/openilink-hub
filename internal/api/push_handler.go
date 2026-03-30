package api

import (
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/push"
)

func (s *Server) handlePushWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate Origin header against the configured RPOrigin to prevent CSWSH.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser clients
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			expected, err := url.Parse(s.Config.RPOrigin)
			if err != nil || expected.Host == "" {
				return false
			}
			return strings.EqualFold(u.Scheme, expected.Scheme) &&
				strings.EqualFold(u.Host, expected.Host)
		},
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("push ws upgrade failed", "err", err)
		return
	}

	c := push.NewConn(userID, ws, s.PushHub)
	s.PushHub.Register(c)

	go c.WritePump()
	c.ReadPump() // blocks until disconnect
}
