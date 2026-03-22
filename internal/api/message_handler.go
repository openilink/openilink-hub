package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	botID := r.URL.Query().Get("bot_id")
	if botID == "" {
		http.Error(w, `{"error":"bot_id required"}`, http.StatusBadRequest)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var beforeID int64
	if b := r.URL.Query().Get("before"); b != "" {
		beforeID, _ = strconv.ParseInt(b, 10, 64)
	}

	msgs, err := s.DB.ListMessages(botID, limit, beforeID)
	if err != nil {
		http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}
