package api

import (
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/push"
)

func (s *Server) handlePushWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
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
