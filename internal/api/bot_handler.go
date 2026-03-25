package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	ilinkProvider "github.com/openilink/openilink-hub/internal/provider/ilink"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
)

func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	bots, err := s.DB.ListBotsByUser(userID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}

	type botResp struct {
		ID                 string          `json:"id"`
		Name               string          `json:"name"`
		Provider           string          `json:"provider"`
		Status             string          `json:"status"`
		CanSend            bool            `json:"can_send"`
		SendDisabledReason string          `json:"send_disabled_reason,omitempty"`
		MsgCount           int64           `json:"msg_count"`
		ReminderHours      int             `json:"reminder_hours"`
		CreatedAt          int64           `json:"created_at"`
		Extra              json.RawMessage `json:"extra,omitempty"`
	}
	// Batch check context_token freshness to avoid N+1 queries
	botIDs := make([]string, len(bots))
	for i, b := range bots {
		botIDs[i] = b.ID
	}
	freshTokens := s.DB.BatchHasFreshContextToken(botIDs, contextTokenMaxAge)

	var result []botResp
	for _, b := range bots {
		status := b.Status
		if inst, ok := s.BotManager.GetInstance(b.ID); ok {
			status = inst.Status()
		}
		canSend, reason := checkSendStatus(status, freshTokens[b.ID])
		extra := extractPublicCredentials(b.Provider, b.Credentials)
		result = append(result, botResp{
			ID: b.ID, Name: b.Name, Provider: b.Provider,
			Status: status, CanSend: canSend, SendDisabledReason: reason,
			MsgCount: b.MsgCount, ReminderHours: b.ReminderHours,
			CreatedAt: b.CreatedAt, Extra: extra,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// extractPublicCredentials returns non-secret info from credentials for the API response.
func extractPublicCredentials(prov string, creds json.RawMessage) json.RawMessage {
	if prov == "ilink" {
		var c struct {
			BotID       string `json:"bot_id"`
			ILinkUserID string `json:"ilink_user_id"`
		}
		json.Unmarshal(creds, &c)
		data, _ := json.Marshal(map[string]string{
			"bot_id":        c.BotID,
			"ilink_user_id": c.ILinkUserID,
		})
		return data
	}
	return nil
}

const contextTokenMaxAge = 24 * time.Hour

// checkSendStatus is a pure function that determines send capability from pre-fetched data.
func checkSendStatus(status string, hasFreshToken bool) (bool, string) {
	if status == "session_expired" {
		return false, "会话已过期，请重新扫码绑定"
	}
	if status != "connected" {
		return false, "Bot 未连接"
	}
	if !hasFreshToken {
		return false, "超过 24 小时未收到消息，无法主动发送"
	}
	return true, ""
}

// checkSendability queries the DB and returns send capability for a single bot.
func (s *Server) checkSendability(botID, status string) (bool, string) {
	hasFresh := s.DB.HasFreshContextToken(botID, contextTokenMaxAge)
	return checkSendStatus(status, hasFresh)
}

func (s *Server) handleBindStart(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	sessionID, qrURL, err := ilinkProvider.StartBind(r.Context(), userID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sessionID,
		"qr_url":     qrURL,
	})
}

func (s *Server) handleBindStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")
	enableAI := r.URL.Query().Get("enable_ai") == "true"

	ilinkProvider.PendingBinds.Lock()
	entry, ok := ilinkProvider.PendingBinds.M[sessionID]
	ilinkProvider.PendingBinds.Unlock()
	if !ok {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}
	defer ws.Close()

	// Read pump: detect client disconnect
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}()

	sendEvent := func(event, data string) {
		var parsed json.RawMessage
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			parsed, _ = json.Marshal(data)
		}
		msg := map[string]any{"event": event}
		var fields map[string]any
		if json.Unmarshal(parsed, &fields) == nil {
			for k, v := range fields {
				msg[k] = v
			}
		}
		ws.WriteJSON(msg)
	}

	for {
		select {
		case <-done:
			return
		default:
		}

		result, err := ilinkProvider.PollBind(context.Background(), sessionID)
		if err != nil {
			sendEvent("error", `{"message":"poll failed"}`)
			return
		}

		switch result.Status {
		case "wait":
			sendEvent("status", `{"status":"wait"}`)
		case "scanned":
			sendEvent("status", `{"status":"scanned"}`)
		case "expired":
			j, _ := json.Marshal(map[string]string{"status": "refreshed", "qr_url": result.QRURL})
			sendEvent("status", string(j))
		case "confirmed":
			var creds struct {
				BotID       string `json:"bot_id"`
				ILinkUserID string `json:"ilink_user_id"`
			}
			json.Unmarshal(result.Credentials, &creds)

			var bot *database.Bot

			// 1. Match by provider_id (exact bot_id)
			if creds.BotID != "" {
				existing, _ := s.DB.FindBotByProviderID("ilink", creds.BotID)
				if existing != nil {
					if existing.UserID != entry.UserID {
						sendEvent("error", `{"message":"this account is already bound by another user"}`)
						return
					}
					s.BotManager.StopBot(existing.ID)
					if err := s.DB.UpdateBotCredentials(existing.ID, creds.BotID, result.Credentials); err != nil {
						slog.Error("rebind update failed", "err", err)
						sendEvent("error", `{"message":"rebind failed"}`)
						return
					}
					existing.Credentials = result.Credentials
					existing.Status = "connected"
					bot = existing
				}
			}

			// 2. Match by ilink_user_id (same WeChat user, new bot_id)
			if bot == nil && creds.ILinkUserID != "" {
				sibling, _ := s.DB.FindBotByCredential("ilink_user_id", creds.ILinkUserID)
				if sibling != nil && sibling.UserID == entry.UserID {
					s.BotManager.StopBot(sibling.ID)
					if err := s.DB.UpdateBotCredentials(sibling.ID, creds.BotID, result.Credentials); err != nil {
						slog.Error("rebind update failed", "err", err)
						sendEvent("error", `{"message":"rebind failed"}`)
						return
					}
					sibling.Credentials = result.Credentials
					sibling.ProviderID = creds.BotID
					sibling.Status = "connected"
					bot = sibling
				}
			}

			if bot == nil {
				var err error
				bot, err = s.DB.CreateBot(entry.UserID, "", "ilink", creds.BotID, result.Credentials)
				if err != nil {
					slog.Error("save bot failed", "err", err)
					sendEvent("error", `{"message":"save failed"}`)
					return
				}
				// Auto-create default channel for new bots only
				var aiCfg *database.AIConfig
				if enableAI {
					aiCfg = &database.AIConfig{Enabled: true, Source: "builtin"}
				}
				s.DB.CreateChannel(bot.ID, "默认", "", nil, aiCfg)
			}

			s.BotManager.StartBot(context.Background(), bot)

			j, _ := json.Marshal(map[string]string{"status": "connected", "bot_id": bot.ID})
			sendEvent("status", string(j))
			return
		}
	}
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if bot.Status == "session_expired" {
		jsonError(w, "session expired, please re-bind this bot", http.StatusConflict)
		return
	}

	s.BotManager.StartBot(r.Context(), bot)
	jsonOK(w)
}

func (s *Server) handleDeleteBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	s.BotManager.StopBot(botID)
	s.DB.DeleteBot(botID)
	jsonOK(w)
}

func (s *Server) handleUpdateBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var req struct {
		Name          *string `json:"name"`
		ReminderHours *int    `json:"reminder_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.Name != nil && *req.Name != "" {
		if err := s.DB.UpdateBotName(botID, *req.Name); err != nil {
			jsonError(w, "update failed", http.StatusInternalServerError)
			return
		}
	}
	if req.ReminderHours != nil {
		hours := *req.ReminderHours
		if hours < 0 || hours > 23 {
			jsonError(w, "reminder_hours must be 0-23", http.StatusBadRequest)
			return
		}
		if err := s.DB.UpdateBotReminder(botID, hours); err != nil {
			jsonError(w, "update failed", http.StatusInternalServerError)
			return
		}
	}
	jsonOK(w)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	stats, err := s.DB.GetBotStats(userID)
	if err != nil {
		jsonError(w, "stats failed", http.StatusInternalServerError)
		return
	}
	stats.ConnectedWS = s.Hub.ConnectedCount()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.DB.GetAdminStats()
	if err != nil {
		jsonError(w, "stats failed", http.StatusInternalServerError)
		return
	}
	stats.ConnectedWS = s.Hub.ConnectedCount()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// POST /api/bots/{id}/send
// Supports JSON body (text) or multipart/form-data (media).
// JSON: {"text": "hello", "recipient": "optional"}
// Multipart: file=@image.jpg, text=caption (optional), recipient=xxx (optional)
func (s *Server) handleBotSend(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	status := bot.Status
	inst, ok := s.BotManager.GetInstance(botID)
	if ok {
		status = inst.Status()
	}

	canSend, reason := s.checkSendability(botID, status)
	if !canSend {
		jsonError(w, reason, http.StatusConflict)
		return
	}
	if !ok {
		jsonError(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	msg, msgType, err := parseSendRequest(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auto-fill context_token from latest message if not provided
	if msg.ContextToken == "" {
		msg.ContextToken = s.DB.GetLatestContextToken(botID)
	}

	clientID, err := inst.Send(r.Context(), msg)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	// Save outbound message
	content := msg.Text
	if content == "" && msg.FileName != "" {
		content = msg.FileName
	}
	itemList, _ := json.Marshal([]map[string]any{{"type": msgType, "text": content}})

	mediaStatus := ""
	mediaKeys := json.RawMessage(`{}`)
	if len(msg.Data) > 0 && s.Store != nil {
		ct := detectContentType(msgType)
		ext := detectExt(msg.FileName, msgType)
		key := fmt.Sprintf("%s/%s/out_%d%s", botID,
			time.Now().Format("2006/01/02"), time.Now().UnixMilli(), ext)
		if _, err := s.Store.Put(r.Context(), key, ct, msg.Data); err == nil {
			mediaStatus = "ready"
			mediaKeys, _ = json.Marshal(map[string]string{"0": key})
		}
	}

	s.DB.SaveMessage(&database.Message{
		BotID:       botID,
		Direction:   "outbound",
		ToUserID:    msg.Recipient,
		MessageType: 2,
		ItemList:    itemList,
		MediaStatus: mediaStatus,
		MediaKeys:   mediaKeys,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "client_id": clientID})
}

func detectMediaType(filename, mime string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasPrefix(mime, "image/"),
		strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"),
		strings.HasSuffix(lower, ".png"), strings.HasSuffix(lower, ".gif"),
		strings.HasSuffix(lower, ".webp"):
		return "image"
	case strings.HasPrefix(mime, "video/"),
		strings.HasSuffix(lower, ".mp4"), strings.HasSuffix(lower, ".mov"),
		strings.HasSuffix(lower, ".avi"):
		return "video"
	case strings.HasPrefix(mime, "audio/"),
		strings.HasSuffix(lower, ".mp3"), strings.HasSuffix(lower, ".wav"),
		strings.HasSuffix(lower, ".ogg"):
		return "voice"
	default:
		return "file"
	}
}

func detectContentType(msgType string) string {
	switch msgType {
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	case "voice":
		return "audio/wav"
	default:
		return "application/octet-stream"
	}
}

func detectExt(filename, msgType string) string {
	if ext := filepath.Ext(filename); ext != "" {
		return ext
	}
	switch msgType {
	case "image":
		return ".jpg"
	case "video":
		return ".mp4"
	case "voice":
		return ".wav"
	default:
		return ""
	}
}

func parseSendRequest(r *http.Request) (provider.OutboundMessage, string, error) {
	ct := r.Header.Get("Content-Type")

	// Multipart: file upload
	if strings.HasPrefix(ct, "multipart/") {
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
			return provider.OutboundMessage{}, "", fmt.Errorf("parse multipart: %w", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			return provider.OutboundMessage{}, "", fmt.Errorf("file required for multipart")
		}
		defer file.Close()
		data, _ := io.ReadAll(file)

		msgType := detectMediaType(header.Filename, header.Header.Get("Content-Type"))

		return provider.OutboundMessage{
			Recipient: r.FormValue("recipient"),
			Text:      r.FormValue("text"),
			Data:      data,
			FileName:  header.Filename,
		}, msgType, nil
	}

	// JSON: text only
	var req struct {
		Recipient string `json:"recipient"`
		Text      string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		return provider.OutboundMessage{}, "", fmt.Errorf("text required")
	}
	return provider.OutboundMessage{
		Recipient: req.Recipient,
		Text:      req.Text,
	}, "text", nil
}

func (s *Server) handleBotContacts(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	contacts, err := s.DB.ListRecentContacts(botID, 100)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}
