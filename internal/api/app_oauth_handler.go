package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
)

// GET /api/apps/{id}/oauth/authorize?bot_id=xxx&state=xxx
// Called when user confirms the install. Creates installation, generates code, redirects to redirect_url.
func (s *Server) handleAppOAuthAuthorize(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")
	botID := r.URL.Query().Get("bot_id")
	state := r.URL.Query().Get("state")

	if botID == "" || state == "" {
		jsonError(w, "bot_id and state required", http.StatusBadRequest)
		return
	}

	app, err := s.DB.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.RedirectURL == "" {
		jsonError(w, "app has no redirect_url configured", http.StatusBadRequest)
		return
	}

	// Verify the user owns the bot
	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Generate temporary code
	codeBytes := make([]byte, 32)
	_, _ = rand.Read(codeBytes)
	code := hex.EncodeToString(codeBytes)

	if err := s.DB.CreateOAuthCode(code, appID, botID, state); err != nil {
		slog.Error("create oauth code failed", "app", appID, "err", err)
		jsonError(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Redirect to app's redirect_url with code and state
	redirectURL := app.RedirectURL + "?code=" + code + "&state=" + state
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// POST /api/apps/{id}/oauth/exchange
// App exchanges a temporary code for installation credentials.
func (s *Server) handleAppOAuthExchange(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("id")

	var req struct {
		Code         string `json:"code"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Code == "" || req.ClientSecret == "" {
		jsonError(w, "code and client_secret required", http.StatusBadRequest)
		return
	}

	// Verify client_secret
	app, err := s.DB.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.ClientSecret != req.ClientSecret {
		jsonError(w, "invalid client_secret", http.StatusUnauthorized)
		return
	}

	// Exchange code
	codeAppID, botID, err := s.DB.ExchangeOAuthCode(req.Code)
	if err != nil {
		jsonError(w, "invalid or expired code", http.StatusBadRequest)
		return
	}
	if codeAppID != appID {
		jsonError(w, "code does not match app", http.StatusBadRequest)
		return
	}

	// Create or get existing installation
	inst, err := s.DB.InstallApp(appID, botID)
	if err != nil {
		// Might already exist — try to find it
		installations, _ := s.DB.ListInstallationsByApp(appID)
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
		"signing_secret":  inst.SigningSecret,
		"bot_id":          inst.BotID,
	})
}

// GET /api/apps/{id}/oauth/setup-redirect?bot_id=xxx
// Starts the OAuth install flow by redirecting to the app's setup_url.
func (s *Server) handleAppOAuthSetupRedirect(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")
	botID := r.URL.Query().Get("bot_id")

	if botID == "" {
		jsonError(w, "bot_id required", http.StatusBadRequest)
		return
	}

	app, err := s.DB.GetApp(appID)
	if err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.SetupURL == "" {
		jsonError(w, "app has no setup_url", http.StatusBadRequest)
		return
	}

	// Verify user owns the bot
	bot, err := s.DB.GetBot(botID)
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

	// Redirect to app's setup_url
	setupURL := app.SetupURL +
		"?hub=" + hubURL +
		"&app_id=" + appID +
		"&bot_id=" + botID +
		"&state=" + state
	http.Redirect(w, r, setupURL, http.StatusFound)
}
