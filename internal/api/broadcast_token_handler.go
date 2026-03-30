package api

import (
	"encoding/json"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
)

// GET /api/broadcast-tokens
func (s *Server) handleListBroadcastTokens(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	tokens, err := s.Store.ListBroadcastTokensByUser(userID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	// Mask tokens in list response — full token is only returned on create/regenerate
	for i := range tokens {
		t := tokens[i].Token
		if len(t) > 11 {
			tokens[i].Token = t[:7] + "..." + t[len(t)-4:]
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

// POST /api/broadcast-tokens
func (s *Server) handleCreateBroadcastToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		Name   string   `json:"name"`
		BotIDs []string `json:"bot_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}

	// Deduplicate and validate bot_ids belong to user
	req.BotIDs = dedupStrings(req.BotIDs)
	if err := s.validateBotOwnership(userID, req.BotIDs); err != nil {
		if _, ok := err.(*botOwnershipError); ok {
			jsonError(w, err.Error(), http.StatusBadRequest)
		} else {
			jsonError(w, "failed to validate bots", http.StatusInternalServerError)
		}
		return
	}

	botIDsJSON, _ := json.Marshal(req.BotIDs)

	token, err := s.Store.CreateBroadcastToken(userID, req.Name, botIDsJSON)
	if err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(token)
}

// PUT /api/broadcast-tokens/{id}
func (s *Server) handleUpdateBroadcastToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	// Verify ownership
	token, err := s.Store.GetBroadcastToken(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if token.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name   string   `json:"name"`
		BotIDs []string `json:"bot_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}

	// Deduplicate and validate bot_ids belong to user
	req.BotIDs = dedupStrings(req.BotIDs)
	if err := s.validateBotOwnership(userID, req.BotIDs); err != nil {
		if _, ok := err.(*botOwnershipError); ok {
			jsonError(w, err.Error(), http.StatusBadRequest)
		} else {
			jsonError(w, "failed to validate bots", http.StatusInternalServerError)
		}
		return
	}

	botIDsJSON, _ := json.Marshal(req.BotIDs)

	if err := s.Store.UpdateBroadcastToken(id, req.Name, botIDsJSON); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}

// DELETE /api/broadcast-tokens/{id}
func (s *Server) handleDeleteBroadcastToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	// Verify ownership
	token, err := s.Store.GetBroadcastToken(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if token.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if err := s.Store.DeleteBroadcastToken(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}

// POST /api/broadcast-tokens/{id}/regenerate
func (s *Server) handleRegenerateBroadcastToken(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id := r.PathValue("id")

	// Verify ownership
	token, err := s.Store.GetBroadcastToken(id)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if token.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	newToken, err := s.Store.RegenerateBroadcastToken(id)
	if err != nil {
		jsonError(w, "regenerate failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": newToken})
}

// dedupStrings removes duplicate strings while preserving order.
func dedupStrings(ss []string) []string {
	seen := make(map[string]bool, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// validateBotOwnership checks that all given bot IDs belong to the user.
func (s *Server) validateBotOwnership(userID string, botIDs []string) error {
	if len(botIDs) == 0 {
		return nil
	}
	bots, err := s.Store.ListBotsByUser(userID)
	if err != nil {
		return err
	}
	owned := make(map[string]bool, len(bots))
	for _, b := range bots {
		owned[b.ID] = true
	}
	for _, id := range botIDs {
		if !owned[id] {
			return &botOwnershipError{id: id}
		}
	}
	return nil
}

type botOwnershipError struct {
	id string
}

func (e *botOwnershipError) Error() string {
	return "bot not found: " + e.id
}
