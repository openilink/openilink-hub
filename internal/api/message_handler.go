package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
)

func (s *Server) handleRetryMedia(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	msgIDStr := r.PathValue("msgId")
	msgID, err := strconv.ParseInt(msgIDStr, 10, 64)
	if err != nil {
		jsonError(w, "invalid message id", http.StatusBadRequest)
		return
	}

	msg, err := s.DB.GetMessage(msgID)
	if err != nil || msg.BotID != botID {
		jsonError(w, "message not found", http.StatusNotFound)
		return
	}

	if err := s.BotManager.RetryMediaDownload(msgID); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	jsonOK(w)
}

func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	var beforeID int64
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		if id, err := decodeMsgCursor(cursor); err == nil {
			beforeID = id
		} else {
			jsonError(w, "invalid cursor", http.StatusBadRequest)
			return
		}
	}

	msgs, err := s.DB.ListMessages(botID, limit+1, beforeID) // fetch one extra to check has_more
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}

	var nextCursor string
	if hasMore && len(msgs) > 0 {
		nextCursor = encodeMsgCursor(msgs[len(msgs)-1].ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"messages":    msgs,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func encodeMsgCursor(id int64) string {
	return encodeCursor(id)
}

func decodeMsgCursor(cursor string) (int64, error) {
	return decodeCursor(cursor)
}

func (s *Server) handleWebhookLogs(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	channelID := r.URL.Query().Get("channel_id")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	logs, err := s.DB.ListWebhookLogs(botID, channelID, limit)
	if err != nil {
		slog.Error("list webhook logs failed", "bot", botID, "channel", channelID, "err", err)
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if logs == nil {
		logs = []database.WebhookLog{}
	}
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleListTraces(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	traces, err := s.DB.ListTraces(botID, limit)
	if err != nil {
		slog.Error("list traces failed", "bot", botID, "err", err)
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if traces == nil {
		traces = []database.MessageTrace{}
	}
	json.NewEncoder(w).Encode(traces)
}
