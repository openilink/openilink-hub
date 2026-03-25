package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

func (s *Server) handleRetryMedia(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.Store.GetBot(botID)
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

	msg, err := s.Store.GetMessage(msgID)
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

	bot, err := s.Store.GetBot(botID)
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

	msgs, err := s.Store.ListMessages(botID, limit+1, beforeID) // fetch one extra to check has_more
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

	// Include send capability so the chat panel can update without extra requests
	status := bot.Status
	if inst, ok := s.BotManager.GetInstance(botID); ok {
		status = inst.Status()
	}
	canSend, sendReason := s.checkSendability(botID, status)

	resp := map[string]any{
		"messages":    msgs,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"can_send":    canSend,
	}
	if sendReason != "" {
		resp["send_disabled_reason"] = sendReason
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
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

	bot, err := s.Store.GetBot(botID)
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

	logs, err := s.Store.ListWebhookLogs(botID, channelID, limit)
	if err != nil {
		slog.Error("list webhook logs failed", "bot", botID, "channel", channelID, "err", err)
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if logs == nil {
		logs = []store.WebhookLog{}
	}
	json.NewEncoder(w).Encode(logs)
}

func (s *Server) handleListTraces(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.Store.GetBot(botID)
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

	spans, err := s.Store.ListRootSpans(botID, limit)
	if err != nil {
		slog.Error("list traces failed", "bot", botID, "err", err)
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if spans == nil {
		spans = []store.TraceSpan{}
	}
	json.NewEncoder(w).Encode(spans)
}

func (s *Server) handleGetTrace(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	botID := r.PathValue("id")

	bot, err := s.Store.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	traceID := r.PathValue("traceId")
	if traceID == "" {
		jsonError(w, "missing trace id", http.StatusBadRequest)
		return
	}

	spans, err := s.Store.ListSpansByTrace(traceID)
	if err != nil {
		slog.Error("get trace failed", "traceId", traceID, "err", err)
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if spans == nil {
		spans = []store.TraceSpan{}
	}
	json.NewEncoder(w).Encode(spans)
}
