package api

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
)

// POST /api/apps/{id}/install
func (s *Server) handleInstallApp(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
	if app == nil {
		return
	}
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		BotID string `json:"bot_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BotID == "" {
		jsonError(w, "bot_id required", http.StatusBadRequest)
		return
	}

	// Verify user owns the bot
	bot, err := s.DB.GetBot(req.BotID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	inst, err := s.DB.InstallApp(app.ID, req.BotID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			jsonError(w, "app already installed on this bot", http.StatusConflict)
			return
		}
		jsonError(w, "install failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(inst)
}

// GET /api/apps/{id}/installations
func (s *Server) handleListInstallations(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(inst.RequestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		jsonError(w, "request failed: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		jsonError(w, "remote returned HTTP "+strconv.Itoa(resp.StatusCode), http.StatusBadGateway)
		return
	}

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		jsonError(w, "invalid response from remote", http.StatusBadGateway)
		return
	}

	if result.Challenge != challenge {
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
	app := s.requireApp(w, r)
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
	app := s.requireApp(w, r)
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
