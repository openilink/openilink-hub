package api

import (
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/web"
)

type Server struct {
	DB           *database.DB
	WebAuthn     *webauthn.WebAuthn
	SessionStore *auth.SessionStore
	BotManager   *bot.Manager
	Hub          *relay.Hub
	Config       *config.Config
	OAuthStates  *oauthStateStore
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// --- Public auth ---
	mux.HandleFunc("POST /api/auth/register", s.handlePasswordRegister)
	mux.HandleFunc("POST /api/auth/login", s.handlePasswordLogin)
	mux.HandleFunc("POST /api/auth/passkey/register/begin", s.handleRegisterBegin)
	mux.HandleFunc("POST /api/auth/passkey/register/finish", s.handleRegisterFinish)
	mux.HandleFunc("POST /api/auth/passkey/login/begin", s.handleLoginBegin)
	mux.HandleFunc("POST /api/auth/passkey/login/finish", s.handleLoginFinish)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)

	// --- OAuth ---
	mux.HandleFunc("GET /api/auth/oauth/providers", s.handleOAuthProviders)
	mux.HandleFunc("GET /api/auth/oauth/{provider}", s.handleOAuthRedirect)
	mux.HandleFunc("GET /api/auth/oauth/{provider}/callback", s.handleOAuthCallback)

	// --- Channel API (api_key auth) ---
	mux.HandleFunc("GET /api/ws", s.handleWebSocket)
	mux.HandleFunc("GET /api/channel/messages", s.handleChannelMessages)
	mux.HandleFunc("POST /api/channel/send", s.handleChannelSend)
	mux.HandleFunc("GET /api/channel/status", s.handleChannelStatus)

	// --- Protected routes ---
	protected := http.NewServeMux()

	// Profile
	protected.HandleFunc("GET /api/auth/me", s.handleMe)
	protected.HandleFunc("PUT /api/auth/profile", s.handleUpdateProfile)
	protected.HandleFunc("PUT /api/auth/password", s.handleChangePassword)

	// OAuth account binding (authenticated)
	protected.HandleFunc("GET /api/auth/linked-accounts", s.handleOAuthAccounts)
	protected.HandleFunc("GET /api/auth/linked-accounts/{provider}/bind", s.handleOAuthBind)
	protected.HandleFunc("DELETE /api/auth/linked-accounts/{provider}", s.handleOAuthUnbind)

	// Bots
	protected.HandleFunc("GET /api/bots", s.handleListBots)
	protected.HandleFunc("POST /api/bots/bind/start", s.handleBindStart)
	protected.HandleFunc("GET /api/bots/bind/status/{sessionID}", s.handleBindStatus)
	protected.HandleFunc("POST /api/bots/{id}/reconnect", s.handleReconnect)
	protected.HandleFunc("DELETE /api/bots/{id}", s.handleDeleteBot)

	// Channels (formerly sublevels)
	protected.HandleFunc("GET /api/channels", s.handleListChannels)
	protected.HandleFunc("POST /api/channels", s.handleCreateChannel)
	protected.HandleFunc("PUT /api/channels/{id}", s.handleUpdateChannel)
	protected.HandleFunc("DELETE /api/channels/{id}", s.handleDeleteChannel)
	protected.HandleFunc("POST /api/channels/{id}/rotate-key", s.handleRotateKey)

	// Bot stats, contacts, send
	protected.HandleFunc("GET /api/stats", s.handleStats)
	protected.HandleFunc("GET /api/bots/{id}/contacts", s.handleBotContacts)
	protected.HandleFunc("PUT /api/bots/{id}/name", s.handleUpdateBotName)
	protected.HandleFunc("POST /api/bots/{id}/send", s.handleBotSend)

	// Messages
	protected.HandleFunc("GET /api/messages", s.handleListMessages)

	// --- Admin: user management ---
	protected.HandleFunc("GET /api/users", s.requireAdmin(s.handleListUsers))
	protected.HandleFunc("POST /api/users", s.requireAdmin(s.handleCreateUser))
	protected.HandleFunc("PUT /api/users/{id}/role", s.requireAdmin(s.handleUpdateUserRole))
	protected.HandleFunc("PUT /api/users/{id}/status", s.requireAdmin(s.handleUpdateUserStatus))
	protected.HandleFunc("PUT /api/users/{id}/password", s.requireAdmin(s.handleResetUserPassword))
	protected.HandleFunc("DELETE /api/users/{id}", s.requireAdmin(s.handleDeleteUser))

	// --- Admin: system config ---
	protected.HandleFunc("GET /api/admin/config/oauth", s.requireAdmin(s.handleGetOAuthConfig))
	protected.HandleFunc("PUT /api/admin/config/oauth/{provider}", s.requireAdmin(s.handleSetOAuthConfig))
	protected.HandleFunc("DELETE /api/admin/config/oauth/{provider}", s.requireAdmin(s.handleDeleteOAuthConfig))
	protected.HandleFunc("GET /api/admin/config/ai", s.requireAdmin(s.handleGetAIConfig))
	protected.HandleFunc("PUT /api/admin/config/ai", s.requireAdmin(s.handleSetAIConfig))
	protected.HandleFunc("DELETE /api/admin/config/ai", s.requireAdmin(s.handleDeleteAIConfig))

	mux.Handle("/api/", auth.Middleware(s.DB)(protected))

	// Serve embedded frontend (production) or skip (dev mode uses vite)
	if handler := web.Handler(); handler != nil {
		mux.Handle("/", handler)
	}

	return cors(mux)
}
