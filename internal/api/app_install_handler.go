package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
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

	slog.Info("install: creating", "app", app.Slug, "bot", req.BotID, "handle", handle)

	inst, err := s.DB.InstallApp(app.ID, req.BotID)
	if err != nil {
		slog.Error("install: db insert failed", "app", app.ID, "bot", req.BotID, "err", err)
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}
	slog.Info("install: created", "inst", inst.ID, "app_token", inst.AppToken[:8]+"...")

	// Set handle
	if err := s.DB.UpdateInstallation(inst.ID, handle, inst.Config, inst.Enabled); err != nil {
		slog.Error("install: set handle failed", "inst", inst.ID, "err", err)
	}
	inst.Handle = handle

	// Auto-notify App via redirect_url (for apps without setup_url)
	if app.SetupURL == "" && app.RedirectURL != "" {
		slog.Info("install: notifying app", "inst", inst.ID, "redirect_url", app.RedirectURL)
		s.notifyAppInstalled(app, inst)
		// Re-read installation to get updated request_url
		if updated, err := s.DB.GetInstallation(inst.ID); err == nil {
			inst = updated
			slog.Info("install: after notify", "inst", inst.ID, "request_url", inst.AppRequestURL)
		}
	} else {
		slog.Info("install: no redirect_url, skipping auto-notify", "inst", inst.ID, "setup_url", app.SetupURL, "redirect_url", app.RedirectURL)
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

	if err := s.DB.UpdateInstallation(inst.ID, handle, cfg, enabled); err != nil {
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

// POST /api/apps/{id}/verify-url
func (s *Server) handleVerifyURL(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
	if app == nil {
		return
	}

	if app.RequestURL == "" {
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
	resp, err := client.Post(app.RequestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("verify-url: request failed", "app", app.ID, "url", app.RequestURL, "err", err)
		jsonError(w, "验证失败：无法连接到 "+app.RequestURL+" ("+err.Error()+")", http.StatusUnprocessableEntity)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyStr := strings.TrimSpace(string(body))

	if resp.StatusCode != http.StatusOK {
		slog.Error("verify-url: remote error", "app", app.ID, "url", app.RequestURL, "status", resp.StatusCode, "body", bodyStr)
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
		slog.Error("verify-url: invalid response", "app", app.ID, "url", app.RequestURL, "body", bodyStr, "err", err)
		jsonError(w, "验证失败：远端返回了无效的响应", http.StatusUnprocessableEntity)
		return
	}

	if result.Challenge != challenge {
		slog.Error("verify-url: challenge mismatch", "app", app.ID)
		jsonError(w, "challenge mismatch", http.StatusUnprocessableEntity)
		return
	}

	if err := s.DB.SetAppURLVerified(app.ID, true); err != nil {
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
		"signing_secret":  app.SigningSecret,
		"bot_id":          inst.BotID,
		"handle":          inst.Handle,
		"hub_url":         s.Config.RPOrigin,
	})

	slog.Info("notify: POST to redirect_url", "inst", inst.ID, "url", app.RedirectURL)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(app.RedirectURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("notify: request failed", "inst", inst.ID, "url", app.RedirectURL, "err", err)
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
		RequestURL string `json:"request_url"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.RequestURL == "" {
		slog.Error("notify: no request_url in response", "inst", inst.ID, "body", string(body))
		return
	}

	slog.Info("notify: got request_url", "app", app.ID, "request_url", result.RequestURL)

	// Auto-set request_url on the App and verify
	if err := s.DB.UpdateAppRequestURL(app.ID, result.RequestURL); err != nil {
		slog.Error("notify: update request_url failed", "app", app.ID, "err", err)
		return
	}
	s.autoVerifyURL(app.ID, result.RequestURL)
}

// autoVerifyURL sends a challenge to verify the app's request_url.
func (s *Server) autoVerifyURL(appID, requestURL string) {
	challengeBytes := make([]byte, 16)
	_, _ = rand.Read(challengeBytes)
	challenge := hex.EncodeToString(challengeBytes)

	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	slog.Info("auto-verify: POST challenge", "app", appID, "url", requestURL)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(requestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		slog.Error("auto-verify: request failed", "app", appID, "url", requestURL, "err", err)
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
		_ = s.DB.SetAppURLVerified(appID, true)
		slog.Info("auto-verify: success", "app", appID)
	} else {
		slog.Error("auto-verify: challenge mismatch", "app", appID, "expected", challenge, "got", result.Challenge)
	}
}
