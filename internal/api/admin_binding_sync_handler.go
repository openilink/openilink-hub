package api

import (
	"database/sql"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

type adminSyncEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type bindingProfileSnapshotPayload struct {
	EventID         string `json:"event_id"`
	EventTime       int64  `json:"event_time"`
	BotID           string `json:"bot_id"`
	SenderUserID    string `json:"sender_user_id"`
	BindingID       string `json:"binding_id"`
	SystemPrompt    string `json:"system_prompt"`
	UserPrompt      string `json:"user_prompt"`
	FullPrompt      string `json:"full_prompt"`
	PromptVersion   int64  `json:"prompt_version"`
	SourceUpdatedAt int64  `json:"source_updated_at"`
	GatewayStatus   string `json:"gateway_status"`
}

type bindingInvalidatedPayload struct {
	EventID      string `json:"event_id"`
	EventTime    int64  `json:"event_time"`
	BotID        string `json:"bot_id"`
	SenderUserID string `json:"sender_user_id"`
	BindingID    string `json:"binding_id"`
	Reason       string `json:"reason"`
}

type bindingPrebindPayload struct {
	EventID   string `json:"event_id"`
	EventTime int64  `json:"event_time"`
	BindingID string `json:"binding_id"`
	RoleID    string `json:"role_id"`
	BotID     string `json:"bot_id"`
	SessionID string `json:"session_id"`
}

func (s *Server) handleAdminBindingSync(w http.ResponseWriter, r *http.Request) {
	secret := strings.TrimSpace(os.Getenv("ADMIN_SYNC_SHARED_SECRET"))
	if secret == "" {
		jsonError(w, "admin sync disabled", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1*1024*1024))
	if err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if !verifyAdminSyncSignature(secret, body, r.Header.Get("X-Admin-Signature")) {
		jsonError(w, "invalid signature", http.StatusForbidden)
		return
	}

	var env adminSyncEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		jsonError(w, "invalid json", http.StatusBadRequest)
		return
	}

	fullPromptMax := parseEnvInt("AI_FULL_PROMPT_MAX_BYTES", 8192)

	switch env.Type {
	case "binding_prebind":
		var p bindingPrebindPayload
		if err := json.Unmarshal(env.Data, &p); err != nil {
			jsonError(w, "invalid prebind payload", http.StatusBadRequest)
			return
		}
		if p.EventID == "" || p.BotID == "" || p.BindingID == "" || p.RoleID == "" || p.SessionID == "" {
			jsonError(w, "missing required fields", http.StatusBadRequest)
			return
		}
		created, err := s.Store.CreateAdminSyncInboxEvent(p.EventID)
		if err != nil {
			jsonError(w, "dedup failed", http.StatusInternalServerError)
			return
		}
		if !created {
			jsonOK(w)
			return
		}
		bot, err := s.Store.GetBot(p.BotID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				bot, err = s.Store.FindBotByProviderID("ilink", p.BotID)
				if err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						jsonError(w, "bot not found for id/provider id", http.StatusNotFound)
						return
					}
					jsonError(w, "resolve bot by provider id failed", http.StatusInternalServerError)
					return
				}
			} else {
				jsonError(w, "resolve bot by id failed", http.StatusInternalServerError)
				return
			}
		}
		if bot == nil || bot.ID == "" {
			jsonError(w, "bot not found for id/provider id", http.StatusNotFound)
			return
		}
		providerBotID := strings.TrimSpace(bot.ProviderID)
		if providerBotID == "" {
			providerBotID = p.BotID
		}
		expiresAt := time.Now().Add(10 * time.Minute)
		_, _, err = s.Store.CreateWechatPendingBinding(store.WechatPendingBindingCreateInput{
			EventID:       p.EventID,
			ProviderBotID: providerBotID,
			BotID:         bot.ID,
			BindingID:     p.BindingID,
			RoleID:        p.RoleID,
			SessionID:     p.SessionID,
			Status:        store.WechatPendingBindingStatusPending,
			ExpiresAt:     expiresAt,
		})
		if err != nil {
			jsonError(w, "create wechat pending binding failed", http.StatusInternalServerError)
			return
		}
		jsonOK(w)
	case "binding_profile_snapshot":
		var p bindingProfileSnapshotPayload
		if err := json.Unmarshal(env.Data, &p); err != nil {
			jsonError(w, "invalid snapshot payload", http.StatusBadRequest)
			return
		}
		if p.EventID == "" || p.BotID == "" || p.SenderUserID == "" || p.BindingID == "" {
			jsonError(w, "missing required fields", http.StatusBadRequest)
			return
		}
		created, err := s.Store.CreateAdminSyncInboxEvent(p.EventID)
		if err != nil {
			jsonError(w, "dedup failed", http.StatusInternalServerError)
			return
		}
		if !created {
			jsonOK(w)
			return
		}

		prompt := store.NormalizePromptByMaxBytes(p.FullPrompt, fullPromptMax)
		if store.IsBlankPrompt(prompt.Value) {
			jsonError(w, "full_prompt is blank", http.StatusBadRequest)
			return
		}

		profile, changed, err := s.Store.UpsertPromptProfile(store.PromptProfileUpsertInput{
			BotID:           p.BotID,
			SenderUserID:    p.SenderUserID,
			BindingID:       p.BindingID,
			SystemPrompt:    p.SystemPrompt,
			UserPrompt:      p.UserPrompt,
			FullPrompt:      prompt.Value,
			PromptVersion:   p.PromptVersion,
			SourceUpdatedAt: p.SourceUpdatedAt,
			Status:          store.PromptProfileStatusActive,
		})
		if err != nil {
			jsonError(w, "upsert prompt profile failed", http.StatusInternalServerError)
			return
		}
		if changed {
			payload, _ := json.Marshal(map[string]any{
				"event_id":          p.EventID,
				"event_type":        "binding_profile_snapshot",
				"bot_id":            p.BotID,
				"sender_user_id":    p.SenderUserID,
				"binding_id":        p.BindingID,
				"prompt_version":    profile.PromptVersion,
				"source_updated_at": profile.SourceUpdatedAt,
				"status":            profile.Status,
				"prompt_truncated":  prompt.Truncated,
			})
			_, _, _ = s.Store.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
				EventID:      p.EventID + ":prompt_profile_changed",
				EventType:    store.OutboxEventPromptProfileChange,
				PartitionKey: p.BotID,
				Payload:      payload,
			})
		}
		jsonOK(w)
	case "binding_invalidated":
		var p bindingInvalidatedPayload
		if err := json.Unmarshal(env.Data, &p); err != nil {
			jsonError(w, "invalid invalidated payload", http.StatusBadRequest)
			return
		}
		if p.EventID == "" || p.BotID == "" || p.SenderUserID == "" {
			jsonError(w, "missing required fields", http.StatusBadRequest)
			return
		}
		created, err := s.Store.CreateAdminSyncInboxEvent(p.EventID)
		if err != nil {
			jsonError(w, "dedup failed", http.StatusInternalServerError)
			return
		}
		if !created {
			jsonOK(w)
			return
		}
		changed, err := s.Store.InvalidatePromptProfile(p.BotID, p.SenderUserID, p.BindingID)
		if err != nil {
			jsonError(w, "invalidate prompt profile failed", http.StatusInternalServerError)
			return
		}
		if changed {
			payload, _ := json.Marshal(map[string]any{
				"event_id":        p.EventID,
				"event_type":      "binding_invalidated",
				"bot_id":          p.BotID,
				"sender_user_id":  p.SenderUserID,
				"binding_id":      p.BindingID,
				"reason":          p.Reason,
				"invalidation_at": time.Now().Unix(),
			})
			_, _, _ = s.Store.EnqueueSyncOutboxEvent(store.EnqueueOutboxInput{
				EventID:      p.EventID + ":binding_invalidated",
				EventType:    store.OutboxEventBindingInvalidated,
				PartitionKey: p.BotID,
				Payload:      payload,
			})
		}
		jsonOK(w)
	default:
		jsonError(w, "unsupported event type", http.StatusBadRequest)
	}
}

func verifyAdminSyncSignature(secret string, body []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	if strings.HasPrefix(signature, "sha256=") {
		signature = strings.TrimPrefix(signature, "sha256=")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func parseEnvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
