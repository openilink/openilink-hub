package api

import (
	"encoding/json"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	// Verify bot ownership
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	channels, err := s.Store.ListChannelsByBotIDs([]string{botID})
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

func (s *Server) handleCreateChannel(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	var req struct {
		Name       string               `json:"name"`
		Handle     string               `json:"handle"`
		FilterRule *store.FilterRule  `json:"filter_rule,omitempty"`
		AIConfig   *store.AIConfig   `json:"ai_config,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}

	// Verify bot ownership
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "bot not found", http.StatusNotFound)
		return
	}

	ch, err := s.Store.CreateChannel(botID, req.Name, req.Handle, req.FilterRule, req.AIConfig)
	if err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ch)
}

func (s *Server) handleUpdateChannel(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	// Verify bot ownership
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	ch, err := s.Store.GetChannel(cid)
	if err != nil || ch.BotID != botID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name          string              `json:"name"`
		Handle        *string             `json:"handle"`
		FilterRule    *store.FilterRule `json:"filter_rule"`
		AIConfig      *store.AIConfig  `json:"ai_config"`
		WebhookConfig *store.WebhookConfig `json:"webhook_config"`
		Enabled       *bool               `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := ch.Name
	if req.Name != "" {
		name = req.Name
	}
	handle := ch.Handle
	if req.Handle != nil {
		handle = *req.Handle
	}
	filter := &ch.FilterRule
	if req.FilterRule != nil {
		filter = req.FilterRule
	}
	ai := &ch.AIConfig
	if req.AIConfig != nil {
		ai = req.AIConfig
	}
	webhook := &ch.WebhookConfig
	if req.WebhookConfig != nil {
		webhook = req.WebhookConfig
	}
	enabled := ch.Enabled
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if err := s.Store.UpdateChannel(cid, name, handle, filter, ai, webhook, enabled); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleDeleteChannel(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	// Verify bot ownership
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	ch, err := s.Store.GetChannel(cid)
	if err != nil || ch.BotID != botID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if err := s.Store.DeleteChannel(cid); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func (s *Server) handleRotateKey(w http.ResponseWriter, r *http.Request) {
	cid := r.PathValue("cid")
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	// Verify bot ownership
	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	ch, err := s.Store.GetChannel(cid)
	if err != nil || ch.BotID != botID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	newKey, err := s.Store.RotateChannelKey(cid)
	if err != nil {
		jsonError(w, "rotate failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"api_key": newKey})
}
