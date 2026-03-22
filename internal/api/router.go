package api

import (
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/relay"
)

type Server struct {
	DB           *database.DB
	WebAuthn     *webauthn.WebAuthn
	SessionStore *auth.SessionStore
	BotManager   *bot.Manager
	Hub          *relay.Hub
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Auth (public)
	mux.HandleFunc("POST /api/auth/register/begin", s.handleRegisterBegin)
	mux.HandleFunc("POST /api/auth/register/finish", s.handleRegisterFinish)
	mux.HandleFunc("POST /api/auth/login/begin", s.handleLoginBegin)
	mux.HandleFunc("POST /api/auth/login/finish", s.handleLoginFinish)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)

	// WebSocket (sub-level auth via api_key)
	mux.HandleFunc("GET /api/ws", s.handleWebSocket)

	// Protected routes
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/auth/me", s.handleMe)
	protected.HandleFunc("GET /api/bots", s.handleListBots)
	protected.HandleFunc("POST /api/bots/bind/start", s.handleBindStart)
	protected.HandleFunc("GET /api/bots/bind/status/{sessionID}", s.handleBindStatus)
	protected.HandleFunc("POST /api/bots/{id}/reconnect", s.handleReconnect)
	protected.HandleFunc("DELETE /api/bots/{id}", s.handleDeleteBot)
	protected.HandleFunc("GET /api/sublevels", s.handleListSublevels)
	protected.HandleFunc("POST /api/sublevels", s.handleCreateSublevel)
	protected.HandleFunc("DELETE /api/sublevels/{id}", s.handleDeleteSublevel)
	protected.HandleFunc("POST /api/sublevels/{id}/rotate-key", s.handleRotateKey)
	protected.HandleFunc("GET /api/messages", s.handleListMessages)

	mux.Handle("/api/", auth.Middleware(s.DB)(protected))

	return mux
}
