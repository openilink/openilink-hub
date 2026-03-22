package api

import (
	"encoding/json"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
)

func (s *Server) handleListSublevels(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	subs, err := s.DB.ListSublevelsByUser(userID)
	if err != nil {
		http.Error(w, `{"error":"list failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (s *Server) handleCreateSublevel(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		BotID string `json:"bot_id"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BotID == "" || req.Name == "" {
		http.Error(w, `{"error":"bot_id and name required"}`, http.StatusBadRequest)
		return
	}

	// Verify bot belongs to user
	bot, err := s.DB.GetBot(req.BotID)
	if err != nil || bot.UserID != userID {
		http.Error(w, `{"error":"bot not found"}`, http.StatusNotFound)
		return
	}

	sub, err := s.DB.CreateSublevel(userID, req.BotID, req.Name)
	if err != nil {
		http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sub)
}

func (s *Server) handleDeleteSublevel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.DB.DeleteSublevel(id); err != nil {
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleRotateKey(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	newKey, err := s.DB.RotateSublevelKey(id)
	if err != nil {
		http.Error(w, `{"error":"rotate failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"api_key": newKey})
}
