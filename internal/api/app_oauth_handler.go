package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
)

// GET /api/apps/{id}/oauth/authorize?bot_id=xxx&state=xxx&code_challenge=xxx
// Called when user confirms the install. Creates installation, generates code, redirects to oauth_redirect_url.
func (s *Server) handleAppOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")
	botID := r.URL.Query().Get("bot_id")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")

	if botID == "" || state == "" {
		jsonError(w, "bot_id and state required", http.StatusBadRequest)
		return
	}

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.OAuthRedirectURL == "" {
		jsonError(w, "app has no oauth_redirect_url configured", http.StatusBadRequest)
		return
	}

	// Verify the user owns the bot
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Generate temporary code
	codeBytes := make([]byte, 32)
	_, _ = rand.Read(codeBytes)
	code := hex.EncodeToString(codeBytes)

	if err := s.Store.CreateOAuthCode(code, appID, botID, state, codeChallenge); err != nil {
		slog.Error("create oauth code failed", "app", appID, "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Redirect to app's oauth_redirect_url with code and state
	redirectURL := app.OAuthRedirectURL + "?code=" + code + "&state=" + state
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// POST /api/apps/{id}/oauth/exchange
// App exchanges a temporary code for installation credentials.
// Supports PKCE (code_verifier) for auth. If no PKCE was used at authorize time,
// the code itself is sufficient (single-use, short-lived).
func (s *Server) handleAppOAuthExchange(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("id")

	var req struct {
		Code         string `json:"code"`
		CodeVerifier string `json:"code_verifier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" {
		jsonError(w, "code required", http.StatusBadRequest)
		return
	}

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}

	// Exchange code
	codeAppID, botID, codeChallenge, exchangeErr := s.Store.ExchangeOAuthCode(req.Code)
	if exchangeErr != nil {
		jsonError(w, "invalid or expired code", http.StatusBadRequest)
		return
	}
	if codeAppID != appID {
		jsonError(w, "code does not match app", http.StatusBadRequest)
		return
	}

	// PKCE verification
	if codeChallenge != "" {
		if req.CodeVerifier == "" {
			jsonError(w, "code_verifier required", http.StatusBadRequest)
			return
		}
		// Verify SHA256(code_verifier) == code_challenge (S256 method)
		h := sha256.Sum256([]byte(req.CodeVerifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		if computed != codeChallenge {
			jsonError(w, "invalid code_verifier", http.StatusUnauthorized)
			return
		}
	}
	// If no code_challenge was set and no code_verifier provided, allow it
	// (the code itself is single-use and expires quickly)

	// Create or get existing installation
	inst, err := s.Store.InstallApp(appID, botID)
	if err != nil {
		// Might already exist — try to find it
		installations, _ := s.Store.ListInstallationsByApp(appID)
		for _, i := range installations {
			if i.BotID == botID {
				inst = &i
				break
			}
		}
		if inst == nil {
			slog.Error("install app via oauth failed", "app", appID, "bot", botID, "err", err)
			jsonError(w, "install failed", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"installation_id": inst.ID,
		"app_token":       inst.AppToken,
		"webhook_secret":  app.WebhookSecret,
		"bot_id":          inst.BotID,
	})
}

// GET /api/apps/{id}/oauth/setup-redirect?bot_id=xxx
// Starts the OAuth install flow by redirecting to the app's oauth_setup_url.
func (s *Server) handleAppOAuthSetupRedirect(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")
	botID := r.URL.Query().Get("bot_id")

	if botID == "" {
		jsonError(w, "bot_id required", http.StatusBadRequest)
		return
	}

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.OAuthSetupURL == "" {
		jsonError(w, "app has no oauth_setup_url", http.StatusBadRequest)
		return
	}

	// Verify user owns the bot
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Generate state
	stateBytes := make([]byte, 16)
	_, _ = rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Build hub callback URL
	scheme := "https"
	if r.TLS == nil {
		scheme = r.Header.Get("X-Forwarded-Proto")
		if scheme == "" {
			scheme = "http"
		}
	}
	hubURL := scheme + "://" + r.Host

	// Redirect to app's oauth_setup_url
	setupURL := app.OAuthSetupURL +
		"?hub=" + hubURL +
		"&app_id=" + appID +
		"&bot_id=" + botID +
		"&state=" + state
	http.Redirect(w, r, setupURL, http.StatusFound)
}
