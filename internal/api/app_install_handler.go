package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
)

// POST /api/apps/{id}/install
func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		BotID  string `json:"bot_id"`
		Handle string `json:"handle"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BotID == "" {
		jsonError(w, "bot_id required", http.StatusBadRequest)
		return
	}

	handle := req.Handle

	// Verify user owns the bot
	bot, err := s.DB.GetBot(req.BotID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Check handle uniqueness on this bot (only if handle is set)
	if handle != "" {
		existing, _ := s.DB.GetInstallationByHandle(req.BotID, handle)
		if existing != nil {
			jsonError(w, "handle @"+handle+" already in use on this bot", http.StatusConflict)
			return
		}
	}

	inst, err := s.DB.InstallApp(app.ID, req.BotID)
	if err != nil {
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}

	// Set handle
	_ = s.DB.UpdateInstallation(inst.ID, inst.RequestURL, handle, inst.Config, inst.Enabled)
	inst.Handle = handle

	// Auto-notify App via redirect_url (for apps without setup_url)
	if app.SetupURL == "" && app.RedirectURL != "" {
		s.notifyAppInstalled(app, inst)
		// Re-read installation to get updated request_url
		if updated, err := s.DB.GetInstallation(inst.ID); err == nil {
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

	installations, err := s.DB.ListInstallationsByApp(app.ID)
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
		RequestURL *string          `json:"request_url"`
		Handle     *string          `json:"handle"`
		Config     json.RawMessage  `json:"config"`
		Enabled    *bool            `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	requestURL := inst.RequestURL
	if req.RequestURL != nil {
		requestURL = *req.RequestURL
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

	if err := s.DB.UpdateInstallation(inst.ID, requestURL, handle, cfg, enabled); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Reset url_verified if URL changed
	if req.RequestURL != nil && *req.RequestURL != inst.RequestURL {
		_ = s.DB.SetInstallationURLVerified(inst.ID, false)
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

	if err := s.DB.DeleteInstallation(inst.ID); err != nil {
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

	token, err := s.DB.RegenerateInstallationToken(inst.ID)
	if err != nil {
		jsonError(w, "regenerate failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"app_token": token})
}

// POST /api/apps/{id}/installations/{iid}/verify-url
func (s *Server) handleVerifyURL(w http.ResponseWriter, r *http.Request) {
	app := s.requireAppForInstall(w, r)
	if app == nil {
		return
	}
	inst := s.requireInstallation(w, r, app.ID)
	if inst == nil {
		return
	}

	if inst.RequestURL == "" {
		jsonError(w, "no request_url configured", http.StatusBadRequest)
		return
	}

	// Generate random challenge
	challengeBytes := make([]byte, 16)
	_, _ = rand.Read(challengeBytes)
	challenge := hex.EncodeToString(challengeBytes)

	// Send challenge to the request URL
	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(inst.RequestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("verify-url: request failed", "inst", inst.ID, "url", inst.RequestURL, "err", err)
		jsonError(w, "request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("verify-url: remote error", "inst", inst.ID, "url", inst.RequestURL, "status", resp.StatusCode)
		jsonError(w, "remote returned HTTP "+strconv.Itoa(resp.StatusCode), http.StatusBadGateway)
		return
	}

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("verify-url: invalid response", "inst", inst.ID, "url", inst.RequestURL, "err", err)
		jsonError(w, "invalid response from remote", http.StatusBadGateway)
		return
	}

	if result.Challenge != challenge {
		slog.Error("verify-url: challenge mismatch", "inst", inst.ID)
		jsonError(w, "challenge mismatch", http.StatusUnprocessableEntity)
		return
	}

	if err := s.DB.SetInstallationURLVerified(inst.ID, true); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "url_verified": true})
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

	logs, err := s.DB.ListEventLogs(inst.ID, limit)
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

	logs, err := s.DB.ListAPILogs(inst.ID, limit)
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

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	installations, err := s.DB.ListInstallationsByBot(botID)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if installations == nil {
		installations = []database.AppInstallation{}
	}
	json.NewEncoder(w).Encode(installations)
}

// notifyAppInstalled POSTs installation credentials to the App's redirect_url.
// The App responds with its request_url, which Hub auto-sets and verifies.
func (s *Server) notifyAppInstalled(app *database.App, inst *database.AppInstallation) {
	if app.RedirectURL == "" {
		return
	}
	payload, _ := json.Marshal(map[string]string{
		"installation_id": inst.ID,
		"app_token":       inst.AppToken,
		"signing_secret":  inst.SigningSecret,
		"bot_id":          inst.BotID,
		"handle":          inst.Handle,
		"hub_url":         s.Config.RPOrigin,
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(app.RedirectURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("notify app installed failed", "app", app.ID, "err", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		RequestURL string `json:"request_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.RequestURL == "" {
		return
	}

	// Auto-set request_url and verify
	_ = s.DB.UpdateInstallation(inst.ID, result.RequestURL, inst.Handle, inst.Config, inst.Enabled)
	s.autoVerifyURL(inst.ID, result.RequestURL)
}

// autoVerifyURL sends a challenge to verify the request_url.
func (s *Server) autoVerifyURL(instID, requestURL string) {
	challengeBytes := make([]byte, 16)
	_, _ = rand.Read(challengeBytes)
	challenge := hex.EncodeToString(challengeBytes)

	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(requestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("auto verify url failed", "inst", instID, "err", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}
	if result.Challenge == challenge {
		_ = s.DB.SetInstallationURLVerified(instID, true)
		slog.Info("auto verify url success", "inst", instID)
	}
}
