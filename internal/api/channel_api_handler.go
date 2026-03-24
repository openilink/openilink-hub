package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/openilink/openilink-hub/internal/database"
)

func encodeCursor(id int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("v1:%d", id)))
}

func decodeCursor(cursor string) (int64, error) {
	data, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return 0, err
	}
	var id int64
	_, err = fmt.Sscanf(string(data), "v1:%d", &id)
	return id, err
}

// authenticateChannel extracts and validates the channel API key from the request.
func (s *Server) authenticateChannel(r *http.Request) (*database.Channel, error) {
	key := r.URL.Query().Get("key")
	if key == "" {
		key = r.Header.Get("X-API-Key")
	}
	if key == "" {
		return nil, nil
	}
	ch, err := s.DB.GetChannelByAPIKey(key)
	if err != nil {
		return nil, err
	}
	if !ch.Enabled {
		return nil, nil
	}
	return ch, nil
}

// GET /api/v1/channels/messages?key=xxx&cursor=xxx&limit=50
func (s *Server) handleChannelMessages(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			jsonError(w, "invalid key", http.StatusUnauthorized)
		} else {
			jsonError(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	afterSeq := int64(0)
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		if id, err := decodeCursor(cursor); err == nil {
			afterSeq = id
		} else {
			jsonError(w, "invalid cursor", http.StatusBadRequest)
			return
		}
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	msgs, err := s.DB.GetMessagesSince(ch.BotID, afterSeq, limit)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	for i := range msgs {
		msgs[i].ItemList = marshalChannelItemList(msgs[i].ItemList, ch.APIKey)
	}

	// Update last_seq
	if len(msgs) > 0 {
		s.DB.UpdateChannelLastSeq(ch.ID, msgs[len(msgs)-1].ID)
	}

	// Build response with next_cursor
	var nextCursor string
	if len(msgs) == limit {
		nextCursor = encodeCursor(msgs[len(msgs)-1].ID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"messages":    msgs,
		"next_cursor": nextCursor,
	})
}

// POST /api/v1/channels/send?key=xxx
// Supports JSON (text) or multipart/form-data (media).
func (s *Server) handleChannelSend(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			jsonError(w, "invalid key", http.StatusUnauthorized)
		} else {
			jsonError(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	inst, ok := s.BotManager.GetInstance(ch.BotID)
	if !ok {
		bot, _ := s.DB.GetBot(ch.BotID)
		if bot != nil && bot.Status == "session_expired" {
			jsonError(w, "session expired", http.StatusConflict)
		} else {
			jsonError(w, "bot not connected", http.StatusServiceUnavailable)
		}
		return
	}

	msg, msgType, err := parseSendRequest(r)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Auto-fill context_token from latest message if not provided
	if msg.ContextToken == "" {
		msg.ContextToken = s.DB.GetLatestContextToken(ch.BotID)
	}

	clientID, err := inst.Send(context.Background(), msg)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

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
		key := fmt.Sprintf("%s/%s/out_%d%s", ch.BotID,
			time.Now().Format("2006/01/02"), time.Now().UnixMilli(), ext)
		if _, err := s.Store.Put(r.Context(), key, ct, msg.Data); err == nil {
			mediaStatus = "ready"
			mediaKeys, _ = json.Marshal(map[string]string{"0": key})
		}
	}

	s.DB.SaveMessage(&database.Message{
		BotID:       ch.BotID,
		Direction:   "outbound",
		ToUserID:    msg.Recipient,
		MessageType: 2,
		ItemList:    itemList,
		MediaStatus: mediaStatus,
		MediaKeys:   mediaKeys,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"client_id": clientID,
	})
}

// POST /api/v1/channels/typing?key=xxx
func (s *Server) handleChannelTyping(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			jsonError(w, "invalid key", http.StatusUnauthorized)
		} else {
			jsonError(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	var req struct {
		Ticket string `json:"ticket"`
		Status string `json:"status"` // "typing" or "cancel"
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	inst, ok := s.BotManager.GetInstance(ch.BotID)
	if !ok {
		jsonError(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	typing := req.Status != "cancel"
	if err := inst.Provider.SendTyping(context.Background(), "", req.Ticket, typing); err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	jsonOK(w)
}

// POST /api/v1/channels/config?key=xxx
func (s *Server) handleChannelConfig(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			jsonError(w, "invalid key", http.StatusUnauthorized)
		} else {
			jsonError(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	var req struct {
		ContextToken string `json:"context_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	inst, ok := s.BotManager.GetInstance(ch.BotID)
	if !ok {
		jsonError(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	cfg, err := inst.Provider.GetConfig(context.Background(), "", req.ContextToken)
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cfg)
}

// GET /api/v1/channels/status?key=xxx
func (s *Server) handleChannelStatus(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			jsonError(w, "invalid key", http.StatusUnauthorized)
		} else {
			jsonError(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	botStatus := "disconnected"
	if inst, ok := s.BotManager.GetInstance(ch.BotID); ok {
		botStatus = inst.Status()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"channel_id":   ch.ID,
		"channel_name": ch.Name,
		"bot_id":       ch.BotID,
		"bot_status":   botStatus,
	})
}
