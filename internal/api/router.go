package api

import (
	"net/http"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/registry"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/web"
)

type Server struct {
	Store        store.Store
	WebAuthn     *webauthn.WebAuthn
	SessionStore *auth.SessionStore
	BotManager   *bot.Manager
	Hub          *relay.Hub
	Config       *config.Config
	OAuthStates  *oauthStateStore
	ObjectStore  *storage.Storage // optional
	Registry     *registry.Client
	AppWSHub     *app.WSHub
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

	// --- iLink scan login: scan QR to register + login + bind bot ---
	mux.HandleFunc("POST /api/auth/scan/start", s.handleScanLoginStart)
	mux.HandleFunc("GET /api/auth/scan/status/{sessionID}", s.handleScanLoginStatus)

	// --- Public info ---
	mux.HandleFunc("GET /api/info", s.handleInfo)

	// --- Webhook plugins (public: list approved) ---
	mux.HandleFunc("GET /api/webhook-plugins", s.handleListPlugins)
	mux.HandleFunc("GET /api/webhook-plugins/{id}", s.handleGetPlugin)
	mux.HandleFunc("GET /api/webhook-plugins/skill.md", handlePluginSkill)
	mux.HandleFunc("GET /api/webhook-plugins/{id}/versions", s.handlePluginVersions)

	// --- App skill.md ---
	mux.HandleFunc("GET /api/apps/skill.md", handleAppSkill)

	// --- OAuth complete (popup callback page, no auth needed) ---
	mux.HandleFunc("GET /oauth/complete", s.handleOAuthComplete)

	// --- Media proxy (serves MinIO files through Hub) ---
	mux.HandleFunc("GET /api/v1/media/", s.handleMediaProxy)

	// --- Channel API (api_key auth) ---
	mux.HandleFunc("GET /api/v1/channels/connect", s.handleWebSocket)
	mux.HandleFunc("GET /api/v1/channels/messages", s.handleChannelMessages)
	mux.HandleFunc("POST /api/v1/channels/send", s.handleChannelSend)
	mux.HandleFunc("POST /api/v1/channels/typing", s.handleChannelTyping)
	mux.HandleFunc("POST /api/v1/channels/config", s.handleChannelConfig)
	mux.HandleFunc("GET /api/v1/channels/status", s.handleChannelStatus)
	mux.HandleFunc("GET /api/v1/channels/media", s.handleChannelMedia)

	// --- Registry public endpoint ---
	mux.HandleFunc("GET /api/registry/v1/apps.json", s.handleRegistryApps)

	// --- Protected routes ---
	protected := http.NewServeMux()

	// Profile
	protected.HandleFunc("GET /api/me", s.handleMe)
	protected.HandleFunc("PUT /api/me/profile", s.handleUpdateProfile)
	protected.HandleFunc("PUT /api/me/password", s.handleChangePassword)

	// My plugins
	protected.HandleFunc("GET /api/me/plugins", s.handleMyPlugins)

	// Passkey binding (authenticated)
	protected.HandleFunc("GET /api/me/passkeys", s.handleListPasskeys)
	protected.HandleFunc("POST /api/me/passkeys/register/begin", s.handlePasskeyBindBegin)
	protected.HandleFunc("POST /api/me/passkeys/register/finish", s.handlePasskeyBindFinish)
	protected.HandleFunc("DELETE /api/me/passkeys/{id}", s.handleDeletePasskey)

	// OAuth account binding (authenticated)
	protected.HandleFunc("GET /api/me/linked-accounts", s.handleOAuthAccounts)
	protected.HandleFunc("GET /api/me/linked-accounts/{provider}/bind", s.handleOAuthBind)
	protected.HandleFunc("DELETE /api/me/linked-accounts/{provider}", s.handleOAuthUnbind)

	// Bots
	protected.HandleFunc("GET /api/bots", s.handleListBots)
	protected.HandleFunc("POST /api/bots/bind/start", s.handleBindStart)
	protected.HandleFunc("GET /api/bots/bind/status/{sessionID}", s.handleBindStatus)
	protected.HandleFunc("POST /api/bots/{id}/reconnect", s.handleReconnect)
	protected.HandleFunc("DELETE /api/bots/{id}", s.handleDeleteBot)

	// Webhook logs
	protected.HandleFunc("GET /api/bots/{id}/webhook-logs", s.handleWebhookLogs)
	protected.HandleFunc("GET /api/bots/{id}/traces", s.handleListTraces)
	protected.HandleFunc("GET /api/bots/{id}/traces/{traceId}", s.handleGetTrace)

	// Bot app installations
	protected.HandleFunc("GET /api/bots/{id}/apps", s.handleListBotApps)

	// Channels (under bots)
	protected.HandleFunc("GET /api/bots/{id}/channels", s.handleListChannels)
	protected.HandleFunc("POST /api/bots/{id}/channels", s.handleCreateChannel)
	protected.HandleFunc("PUT /api/bots/{id}/channels/{cid}", s.handleUpdateChannel)
	protected.HandleFunc("DELETE /api/bots/{id}/channels/{cid}", s.handleDeleteChannel)
	protected.HandleFunc("POST /api/bots/{id}/channels/{cid}/rotate_key", s.handleRotateKey)

	// Bot operations
	protected.HandleFunc("PUT /api/bots/{id}", s.handleUpdateBot)
	protected.HandleFunc("PUT /api/bots/{id}/ai", s.handleSetBotAI)
	protected.HandleFunc("PUT /api/bots/{id}/ai_model", s.handleSetBotAIModel)
	protected.HandleFunc("POST /api/bots/{id}/send", s.handleBotSend)
	protected.HandleFunc("GET /api/bots/{id}/contacts", s.handleBotContacts)
	protected.HandleFunc("GET /api/bots/stats", s.handleStats)

	// Messages (under bots)
	protected.HandleFunc("GET /api/bots/{id}/messages", s.handleListMessages)
	protected.HandleFunc("POST /api/bots/{id}/messages/{msgId}/retry_media", s.handleRetryMedia)

	// --- Admin: user management ---
	protected.HandleFunc("GET /api/admin/users", s.requireAdmin(s.handleListUsers))
	protected.HandleFunc("POST /api/admin/users", s.requireAdmin(s.handleCreateUser))
	protected.HandleFunc("PUT /api/admin/users/{id}/role", s.requireAdmin(s.handleUpdateUserRole))
	protected.HandleFunc("PUT /api/admin/users/{id}/status", s.requireAdmin(s.handleUpdateUserStatus))
	protected.HandleFunc("PUT /api/admin/users/{id}/password", s.requireAdmin(s.handleResetUserPassword))
	protected.HandleFunc("DELETE /api/admin/users/{id}", s.requireAdmin(s.handleDeleteUser))

	// --- Apps ---
	protected.HandleFunc("POST /api/apps", s.handleCreateApp)
	protected.HandleFunc("GET /api/apps", s.handleListApps)
	protected.HandleFunc("GET /api/apps/{id}", s.handleGetApp)
	protected.HandleFunc("PUT /api/apps/{id}", s.handleUpdateApp)
	protected.HandleFunc("DELETE /api/apps/{id}", s.handleDeleteApp)
	protected.HandleFunc("POST /api/apps/{id}/install", s.handleInstallApp)
	protected.HandleFunc("POST /api/apps/{id}/request-listing", s.handleRequestListing)
	protected.HandleFunc("POST /api/apps/{id}/withdraw-listing", s.handleWithdrawListing)
	protected.HandleFunc("GET /api/apps/{id}/installations", s.handleListInstallations)
	protected.HandleFunc("GET /api/apps/{id}/installations/{iid}", s.handleGetInstallation)
	protected.HandleFunc("PUT /api/apps/{id}/installations/{iid}", s.handleUpdateInstallation)
	protected.HandleFunc("DELETE /api/apps/{id}/installations/{iid}", s.handleDeleteInstallation)
	protected.HandleFunc("POST /api/apps/{id}/installations/{iid}/regenerate-token", s.handleRegenerateToken)
	protected.HandleFunc("POST /api/apps/{id}/installations/{iid}/reauthorize", s.handleReauthorize)
	protected.HandleFunc("POST /api/apps/{id}/verify-url", s.handleVerifyURL)
	protected.HandleFunc("GET /api/apps/{id}/installations/{iid}/event-logs", s.handleAppEventLogs)
	protected.HandleFunc("GET /api/apps/{id}/installations/{iid}/api-logs", s.handleAppAPILogs)

	// --- App OAuth ---
	protected.HandleFunc("GET /api/apps/{id}/oauth/setup", s.handleAppOAuthSetupRedirect)
	protected.HandleFunc("GET /api/apps/{id}/oauth/authorize", s.handleAppOAuthAuthorize)

	// --- Marketplace ---
	protected.HandleFunc("GET /api/marketplace", s.handleMarketplace)
	protected.HandleFunc("GET /api/marketplace/builtin", s.handleBuiltinApps)
	protected.HandleFunc("POST /api/marketplace/sync/{slug}", s.handleMarketplaceSync)

	// --- Webhook plugins (authenticated actions) ---
	protected.HandleFunc("POST /api/webhook-plugins/submit", s.handleSubmitPlugin)
	protected.HandleFunc("POST /api/webhook-plugins/{id}/versions/{vid}/cancel", s.handleCancelVersion)
	protected.HandleFunc("POST /api/webhook-plugins/debug/request", s.handleDebugRequest)
	protected.HandleFunc("POST /api/webhook-plugins/debug/response", s.handleDebugResponse)
	protected.HandleFunc("POST /api/webhook-plugins/{id}/install", s.handleInstallPlugin)
	protected.HandleFunc("POST /api/webhook-plugins/{id}/install-to-channel", s.handleInstallPluginToChannel)

	// --- Admin: dashboard ---
	protected.HandleFunc("GET /api/admin/stats", s.requireAdmin(s.handleAdminStats))

	// --- Admin: webhook plugins ---
	protected.HandleFunc("PUT /api/admin/webhook-plugins/{id}/review", s.requireAdmin(s.handleReviewPlugin))
	protected.HandleFunc("DELETE /api/admin/webhook-plugins/{id}", s.requireAdmin(s.handleDeletePlugin))

	// --- Admin: apps ---
	protected.HandleFunc("GET /api/admin/apps", s.requireAdmin(s.handleAdminListApps))
	protected.HandleFunc("PUT /api/admin/apps/{id}/review-listing", s.requireAdmin(s.handleReviewListing))

	// --- Admin: registries ---
	protected.HandleFunc("GET /api/admin/registries", s.requireAdmin(s.handleListRegistries))
	protected.HandleFunc("POST /api/admin/registries", s.requireAdmin(s.handleCreateRegistry))
	protected.HandleFunc("PUT /api/admin/registries/{id}", s.requireAdmin(s.handleUpdateRegistry))
	protected.HandleFunc("DELETE /api/admin/registries/{id}", s.requireAdmin(s.handleDeleteRegistry))

	// --- Admin: system config ---
	protected.HandleFunc("GET /api/admin/config/oauth", s.requireAdmin(s.handleGetOAuthConfig))
	protected.HandleFunc("PUT /api/admin/config/oauth/{provider}", s.requireAdmin(s.handleSetOAuthConfig))
	protected.HandleFunc("DELETE /api/admin/config/oauth/{provider}", s.requireAdmin(s.handleDeleteOAuthConfig))
	protected.HandleFunc("GET /api/config/ai/available_models", s.handleGetAvailableModels)
	protected.HandleFunc("GET /api/admin/config/ai", s.requireAdmin(s.handleGetAIConfig))
	protected.HandleFunc("PUT /api/admin/config/ai", s.requireAdmin(s.handleSetAIConfig))
	protected.HandleFunc("DELETE /api/admin/config/ai", s.requireAdmin(s.handleDeleteAIConfig))
	protected.HandleFunc("GET /api/admin/config/registry", s.requireAdmin(s.handleGetRegistryConfig))
	protected.HandleFunc("PUT /api/admin/config/registry", s.requireAdmin(s.handleSetRegistryConfig))
	protected.HandleFunc("GET /api/admin/config/registration", s.requireAdmin(s.handleGetRegistrationConfig))
	protected.HandleFunc("PUT /api/admin/config/registration", s.requireAdmin(s.handleSetRegistrationConfig))

	// App OAuth exchange (no user auth — uses PKCE or single-use code)
	mux.HandleFunc("POST /api/apps/{id}/oauth/exchange", s.handleAppOAuthExchange)

	mux.Handle("/api/", auth.Middleware(s.Store)(protected))

	// --- Bot API (app_token auth) ---
	botAPI := http.NewServeMux()
	// New paths
	botAPI.HandleFunc("POST /bot/v1/message/send", s.handleBotAPISend)
	botAPI.HandleFunc("GET /bot/v1/contact", s.handleBotAPIContacts)
	botAPI.HandleFunc("GET /bot/v1/info", s.handleBotAPIBotInfo)
	// Keep old paths for backward compatibility
	botAPI.HandleFunc("POST /bot/v1/messages/send", s.handleBotAPISend)
	botAPI.HandleFunc("GET /bot/v1/contacts", s.handleBotAPIContacts)
	botAPI.HandleFunc("GET /bot/v1/bot", s.handleBotAPIBotInfo)
	botAPI.HandleFunc("PUT /bot/v1/app/tools", s.handleBotAPIUpdateTools)
	botAPI.HandleFunc("PUT /bot/v1/installation/tools", s.handleBotAPIUpdateInstallationTools)
	botAPI.HandleFunc("/bot/", s.handleBotAPINotFound)
	mux.Handle("/bot/", s.appTokenAuth(botAPI))

	// WebSocket endpoints (auth via query param, outside appTokenAuth)
	mux.HandleFunc("GET /bot/v1/ws", s.handleBotAPIWebSocket)          // per-installation
	mux.HandleFunc("GET /bot/v1/app/ws", s.handleAppLevelWebSocket)    // per-app (all installations)

	// Serve embedded frontend (production) or skip (dev mode uses vite)
	if handler := web.Handler(); handler != nil {
		mux.Handle("/", handler)
	}

	return recovery(requestLogger(cors(mux)))
}
