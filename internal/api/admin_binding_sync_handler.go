package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/supamemory"
)

type adminSyncEnvelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
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

func (s *Server) resolveBotByIDOrProviderID(rawBotID string) (*store.Bot, error) {
	botID := strings.TrimSpace(rawBotID)
	if botID == "" {
		return nil, sql.ErrNoRows
	}
	bot, err := s.Store.GetBot(botID)
	if err == nil && bot != nil {
		return bot, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return s.Store.FindBotByProviderID("ilink", botID)
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
			slog.Error("admin sync prebind dedup failed", "event_id", p.EventID, "bot_id", p.BotID, "session_id", p.SessionID, "err", err)
			jsonError(w, "dedup failed", http.StatusInternalServerError)
			return
		}
		if !created {
			jsonOK(w)
			return
		}
		bot, err := s.resolveBotByIDOrProviderID(p.BotID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "bot not found for id/provider id", http.StatusNotFound)
				return
			}
			slog.Error("admin sync prebind resolve bot failed", "event_id", p.EventID, "bot_id", p.BotID, "session_id", p.SessionID, "err", err)
			jsonError(w, "resolve bot failed", http.StatusInternalServerError)
			return
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
			slog.Error("admin sync prebind create pending binding failed", "event_id", p.EventID, "bot_id", bot.ID, "provider_bot_id", providerBotID, "session_id", p.SessionID, "binding_id", p.BindingID, "role_id", p.RoleID, "err", err)
			jsonError(w, "create wechat pending binding failed", http.StatusInternalServerError)
			return
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
		bot, err := s.resolveBotByIDOrProviderID(p.BotID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				jsonError(w, "bot not found for id/provider id", http.StatusNotFound)
				return
			}
			jsonError(w, "resolve bot failed", http.StatusInternalServerError)
			return
		}
		if bot == nil || strings.TrimSpace(bot.ID) == "" {
			jsonError(w, "bot not found for id/provider id", http.StatusNotFound)
			return
		}
		localBotID := strings.TrimSpace(bot.ID)
		if s.SupaMemory != nil {
			_ = s.SupaMemory.WriteAuditLog(r.Context(), supamemory.AuditLogInput{
				EventType: "binding_invalidated",
				SessionID: p.SenderUserID,
				TraceID:   p.EventID,
				Detail: map[string]any{
					"event_id":        p.EventID,
					"bot_id":          localBotID,
					"provider_bot_id": strings.TrimSpace(p.BotID),
					"sender_user_id":  p.SenderUserID,
					"binding_id":      p.BindingID,
					"reason":          p.Reason,
					"invalidation_at": time.Now().Unix(),
				},
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
