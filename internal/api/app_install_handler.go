package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

// POST /api/apps/{id}/install
func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		BotID  string          `json:"bot_id"`
		Handle string          `json:"handle"`
		Scopes json.RawMessage `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BotID == "" {
		jsonError(w, "bot_id required", http.StatusBadRequest)
		return
	}

	handle := req.Handle

	// Verify user owns the bot
	bot, err := s.Store.GetBot(req.BotID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Check handle uniqueness on this bot (only if handle is set)
	if handle != "" {
		existing, _ := s.Store.GetInstallationByHandle(req.BotID, handle)
		if existing != nil {
			jsonError(w, "handle @"+handle+" already in use on this bot", http.StatusConflict)
			return
		}
	}

	// If app has OAuth setup URL, don't create installation now — redirect to OAuth.
	// The OAuth exchange will create the installation.
	if app.OAuthSetupURL != "" {
		oauthRedirectURL := fmt.Sprintf("%s/api/apps/%s/oauth/setup?bot_id=%s", s.Config.RPOrigin, app.ID, req.BotID)
		slog.Info("install: redirecting to OAuth", "app", app.Slug, "bot", req.BotID, "url", oauthRedirectURL)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"needs_oauth":      true,
			"oauth_redirect":   oauthRedirectURL,
		})
		return
	}

	slog.Info("install: creating", "app", app.Slug, "bot", req.BotID, "handle", handle)

	inst, err := s.Store.InstallApp(app.ID, req.BotID)
	if err != nil {
		slog.Error("install: db insert failed", "app", app.ID, "bot", req.BotID, "err", err)
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}
	slog.Info("install: created", "inst", inst.ID, "app_token", inst.AppToken[:8]+"...")

	// Set handle and scopes — snapshot app scopes at install time (Slack model)
	scopes := req.Scopes
	if scopes == nil || string(scopes) == "" || string(scopes) == "[]" || string(scopes) == "null" {
		scopes = app.Scopes
	}
	if err := s.Store.UpdateInstallation(inst.ID, handle, inst.Config, scopes, inst.Enabled); err != nil {
		slog.Error("install: set handle failed", "inst", inst.ID, "err", err)
	}
	inst.Handle = handle
	inst.Scopes = scopes

	// Auto-notify App via oauth_redirect_url (for apps without oauth_setup_url)
	if app.OAuthRedirectURL != "" {
		slog.Info("install: notifying app", "inst", inst.ID, "oauth_redirect_url", app.OAuthRedirectURL)
		s.notifyAppInstalled(app, inst)
		if updated, err := s.Store.GetInstallation(inst.ID); err == nil {
			inst = updated
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inst)
}

// GET /api/apps/{id}/installations
func (s *Server) handleListInstallations(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}

	installations, err := s.Store.ListInstallationsByApp(app.ID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}

	// Mask tokens in list view — show only last 4 chars
	for i := range installations {
		tok := installations[i].AppToken
		if len(tok) > 4 {
			installations[i].AppToken = strings.Repeat("*", len(tok)-4) + tok[len(tok)-4:]
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if installations == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(installations)
}

// GET /api/apps/{id}/installations/{iid}
func (s *Server) handleGetInstallation(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(inst)
}

// PUT /api/apps/{id}/installations/{iid}
func (s *Server) handleUpdateInstallation(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	var req struct {
		Handle  *string         `json:"handle"`
		Config  json.RawMessage `json:"config"`
		Enabled *bool           `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	handle := inst.Handle
	if req.Handle != nil {
		handle = *req.Handle
	}
	cfg := inst.Config
	if req.Config != nil {
		cfg = req.Config
	}
	enabled := inst.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := s.Store.UpdateInstallation(inst.ID, handle, cfg, inst.Scopes, enabled); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}

// DELETE /api/apps/{id}/installations/{iid}
func (s *Server) handleDeleteInstallation(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	if err := s.Store.DeleteInstallation(inst.ID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// POST /api/apps/{id}/installations/{iid}/regenerate-token
func (s *Server) handleRegenerateToken(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	token, err := s.Store.RegenerateInstallationToken(inst.ID)
	if err != nil {
		jsonError(w, "regenerate failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"app_token": token})
}

// POST /api/apps/{id}/verify-url
func (s *Server) handleVerifyURL(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
	if app == nil {
		return
	}

	if app.WebhookURL == "" {
		jsonError(w, "no webhook_url configured", http.StatusBadRequest)
		return
	}

	// Generate random challenge
	challengeBytes := make([]byte, 16)
	_, _ = rand.Read(challengeBytes)
	challenge := hex.EncodeToString(challengeBytes)

	// Send challenge to the webhook URL
	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(app.WebhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("verify-url: request failed", "app", app.ID, "url", app.WebhookURL, "err", err)
		jsonError(w, "验证失败：无法连接到 "+app.WebhookURL+" ("+err.Error()+")", http.StatusUnprocessableEntity)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyStr := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK {
		slog.Error("verify-url: remote error", "app", app.ID, "url", app.WebhookURL, "status", resp.StatusCode, "body", bodyStr)
		msg := "验证失败：远端返回 HTTP " + strconv.Itoa(resp.StatusCode)
		if bodyStr != "" {
			msg += " — " + bodyStr
		}
		jsonError(w, msg, http.StatusUnprocessableEntity)
		return
	}

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("verify-url: invalid response", "app", app.ID, "url", app.WebhookURL, "body", bodyStr, "err", err)
		jsonError(w, "验证失败：远端返回了无效的响应", http.StatusUnprocessableEntity)
		return
	}

	if result.Challenge != challenge {
		slog.Error("verify-url: challenge mismatch", "app", app.ID)
		jsonError(w, "challenge mismatch", http.StatusUnprocessableEntity)
		return
	}

	if err := s.Store.SetAppWebhookVerified(app.ID, true); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "webhook_verified": true})
}

// GET /api/apps/{id}/installations/{iid}/event-logs
func (s *Server) handleAppEventLogs(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	logs, err := s.Store.ListEventLogs(inst.ID, limit)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if logs == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(logs)
}

// GET /api/apps/{id}/installations/{iid}/api-logs
func (s *Server) handleAppAPILogs(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	logs, err := s.Store.ListAPILogs(inst.ID, limit)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if logs == nil {
		w.Write([]byte("[]"))
		return
	}
	json.NewEncoder(w).Encode(logs)
}

// GET /api/bots/{id}/apps — list app installations on a bot
func (s *Server) handleListBotApps(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	installations, err := s.Store.ListInstallationsByBot(botID)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if installations == nil {
		installations = []store.AppInstallation{}
	}
	json.NewEncoder(w).Encode(installations)
}

// POST /api/apps/{id}/installations/{iid}/reauthorize
// Updates installation scopes to the current app scopes.
// This is the mechanism for users to grant new scopes after an app adds them.
func (s *Server) handleReauthorize(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	// Update installation scopes to current app scopes
	if err := s.Store.UpdateInstallation(inst.ID, inst.Handle, inst.Config, app.Scopes, inst.Enabled); err != nil {
		jsonError(w, "reauthorize failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// notifyAppInstalled POSTs installation credentials to the App's oauth_redirect_url.
// The App responds with its webhook_url, which Hub auto-sets and verifies.
func (s *Server) notifyAppInstalled(app *store.App, inst *store.AppInstallation) {
	if app.OAuthRedirectURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"installation_id": inst.ID,
		"app_token":       inst.AppToken,
		"webhook_secret":  app.WebhookSecret,
		"bot_id":          inst.BotID,
		"handle":          inst.Handle,
		"hub_url":         s.Config.RPOrigin,
	})

	slog.Info("notify: POST to oauth_redirect_url", "inst", inst.ID, "url", app.OAuthRedirectURL)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(app.OAuthRedirectURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("notify: request failed", "inst", inst.ID, "url", app.OAuthRedirectURL, "err", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	slog.Info("notify: response", "inst", inst.ID, "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		slog.Error("notify: non-200 response", "inst", inst.ID, "status", resp.StatusCode)
		return
	}

	var result struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.WebhookURL == "" {
		slog.Error("notify: no webhook_url in response", "inst", inst.ID, "body", string(body))
		return
	}

	slog.Info("notify: got webhook_url", "app", app.ID, "webhook_url", result.WebhookURL)

	// Auto-set webhook_url on the App and verify
	if err := s.Store.UpdateAppWebhookURL(app.ID, result.WebhookURL); err != nil {
		slog.Error("notify: update webhook_url failed", "app", app.ID, "err", err)
		return
	}
	s.autoVerifyURL(app.ID, result.WebhookURL)
}

// POST /api/bots/{id}/apps -- unified install endpoint
// Supports three modes:
//   - {"app_id": "uuid"} -- install existing app
//   - {"marketplace_slug": "github"} -- install from marketplace
//   - {"template_slug": "websocket-app", "scopes": [...]} -- install from template (creates App if needed)
func (s *Server) handleUnifiedInstall(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	// Verify user owns the bot
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	var req struct {
		AppID           string          `json:"app_id"`
		MarketplaceSlug string          `json:"marketplace_slug"`
		TemplateSlug    string          `json:"template_slug"`
		Handle          string          `json:"handle"`
		Scopes          json.RawMessage `json:"scopes"`
		// Template-only fields (used when creating app from template)
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Icon        string          `json:"icon"`
		Events      json.RawMessage `json:"events"`
		Readme      string          `json:"readme"`
		Guide       string          `json:"guide"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Check handle uniqueness
	if req.Handle != "" {
		if existing, _ := s.Store.GetInstallationByHandle(botID, req.Handle); existing != nil {
			jsonError(w, "handle @"+req.Handle+" already in use on this bot", http.StatusConflict)
			return
		}
	}

	var app *store.App

	if req.AppID != "" {
		// Mode 1: install existing app by ID
		app, err = s.Store.GetApp(req.AppID)
		if err != nil {
			jsonError(w, "app not found", http.StatusNotFound)
			return
		}
	} else if req.MarketplaceSlug != "" {
		// Mode 2: install from marketplace
		if s.Registry == nil {
			jsonError(w, "marketplace not configured", http.StatusBadRequest)
			return
		}
		regApp, err := s.Registry.GetApp(req.MarketplaceSlug)
		if err != nil || regApp == nil {
			jsonError(w, "app not found in marketplace", http.StatusNotFound)
			return
		}
		// Find or create local app record
		app, _ = s.Store.GetAppBySlug(req.MarketplaceSlug, regApp.RegistryURL)
		if app == nil {
			app, err = s.Store.CreateApp(&store.App{
				Name:             regApp.Name,
				Slug:             regApp.Slug,
				Description:      regApp.Description,
				IconURL:          regApp.IconURL,
				Homepage:         regApp.Homepage,
				WebhookURL:       regApp.WebhookURL,
				OAuthSetupURL:    regApp.OAuthSetupURL,
				OAuthRedirectURL: regApp.OAuthRedirectURL,
				Tools:            regApp.Tools,
				Events:           regApp.Events,
				Scopes:           regApp.Scopes,
				Registry:         regApp.RegistryURL,
				Version:          regApp.Version,
				Readme:           regApp.Readme,
				Guide:            regApp.Guide,
			})
			if err != nil {
				jsonError(w, "create marketplace app failed", http.StatusInternalServerError)
				return
			}
		}
	} else if req.TemplateSlug != "" {
		// Mode 3: install from template (reuse existing App or create)
		app, _ = s.Store.GetAppBySlug(req.TemplateSlug, "builtin")
		if app == nil {
			// First time this template is used on this Hub -- create the App
			app, err = s.Store.CreateApp(&store.App{
				OwnerID:     userID,
				Name:        req.Name,
				Slug:        req.TemplateSlug,
				Description: req.Description,
				Icon:        req.Icon,
				Events:      req.Events,
				Scopes:      req.Scopes,
				Registry:    "builtin",
				Readme:      req.Readme,
				Guide:       req.Guide,
			})
			if err != nil {
				jsonError(w, "create template app failed", http.StatusInternalServerError)
				return
			}
		}
	} else {
		jsonError(w, "app_id, marketplace_slug, or template_slug required", http.StatusBadRequest)
		return
	}

	// If app has OAuth setup URL, don't create installation — redirect to OAuth.
	if app.OAuthSetupURL != "" {
		oauthRedirectURL := fmt.Sprintf("%s/api/apps/%s/oauth/setup?bot_id=%s", s.Config.RPOrigin, app.ID, botID)
		slog.Info("unified install: redirecting to OAuth", "app", app.Slug, "bot", botID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"needs_oauth":    true,
			"oauth_redirect": oauthRedirectURL,
		})
		return
	}

	// Create installation
	inst, err := s.Store.InstallApp(app.ID, botID)
	if err != nil {
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}

	// Set handle and scopes — snapshot app scopes at install time (Slack model)
	handle := req.Handle
	scopes := req.Scopes
	if scopes == nil || string(scopes) == "" || string(scopes) == "[]" || string(scopes) == "null" {
		scopes = app.Scopes
	}
	if handle != "" || scopes != nil {
		s.Store.UpdateInstallation(inst.ID, handle, inst.Config, scopes, inst.Enabled)
		inst.Handle = handle
		inst.Scopes = scopes
	}

	// Auto-notify for apps with redirect URL
	if app.OAuthRedirectURL != "" {
		s.notifyAppInstalled(app, inst)
		if updated, _ := s.Store.GetInstallation(inst.ID); updated != nil {
			inst = updated
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inst)
}

// autoVerifyURL sends a challenge to verify the app's webhook_url.
func (s *Server) autoVerifyURL(appID, webhookURL string) {
	challengeBytes := make([]byte, 16)
	_, _ = rand.Read(challengeBytes)
	challenge := hex.EncodeToString(challengeBytes)

	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	slog.Info("auto-verify: POST challenge", "app", appID, "url", webhookURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("auto-verify: request failed", "app", appID, "url", webhookURL, "err", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	slog.Info("auto-verify: response", "app", appID, "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode != http.StatusOK {
		slog.Error("auto-verify: non-200", "app", appID, "status", resp.StatusCode)
		return
	}

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		slog.Error("auto-verify: invalid response", "app", appID, "err", err)
		return
	}
	if result.Challenge == challenge {
		_ = s.Store.SetAppWebhookVerified(appID, true)
		slog.Info("auto-verify: success", "app", appID)
	} else {
		slog.Error("auto-verify: challenge mismatch", "app", appID, "expected", challenge, "got", result.Challenge)
	}
}
