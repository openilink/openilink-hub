package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

func signAdmin(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestAdminBindingSync_SnapshotAndDedupAndInvalidate(t *testing.T) {
	env := setupTestEnv(t)
	secret := "test-secret"
	t.Setenv("ADMIN_SYNC_SHARED_SECRET", secret)
	t.Setenv("AI_FULL_PROMPT_MAX_BYTES", "16")

	snapshot := map[string]any{
		"type": "binding_profile_snapshot",
		"data": map[string]any{
			"event_id":          "evt-snap-1",
			"event_time":        1715400000,
			"bot_id":            "bot-1",
			"sender_user_id":    "wx-1",
			"binding_id":        "bind-1",
			"system_prompt":     "sys",
			"user_prompt":       "user",
			"full_prompt":       "12345678901234567890",
			"prompt_version":    1,
			"source_updated_at": 100,
		},
	}
	body, _ := json.Marshal(snapshot)
	req, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("snapshot request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("snapshot status=%d", resp.StatusCode)
	}

	p, err := env.store.GetActivePromptProfile("bot-1", "wx-1")
	if err != nil || p == nil {
		t.Fatalf("GetActivePromptProfile err=%v profile=%v", err, p)
	}
	if len(p.FullPrompt) != 16 {
		t.Fatalf("expected truncated full_prompt length=16, got %d", len(p.FullPrompt))
	}

	// Dedup same event id
	req2, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("snapshot dedup request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("snapshot dedup status=%d", resp2.StatusCode)
	}

	inv := map[string]any{
		"type": "binding_invalidated",
		"data": map[string]any{
			"event_id":       "evt-inv-1",
			"event_time":     1715400001,
			"bot_id":         "bot-1",
			"sender_user_id": "wx-1",
			"binding_id":     "bind-1",
			"reason":         "rebind",
		},
	}
	invBody, _ := json.Marshal(inv)
	req3, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(invBody))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("X-Admin-Signature", signAdmin(secret, invBody))
	resp3, err := http.DefaultClient.Do(req3)
	if err != nil {
		t.Fatalf("invalidate request: %v", err)
	}
	defer resp3.Body.Close()
	if resp3.StatusCode != http.StatusOK {
		t.Fatalf("invalidate status=%d", resp3.StatusCode)
	}

	active, err := env.store.GetActivePromptProfile("bot-1", "wx-1")
	if err != nil {
		t.Fatalf("GetActivePromptProfile after invalidate: %v", err)
	}
	if active != nil {
		t.Fatalf("expected no active profile after invalidate, got %#v", active)
	}

	// Ensure outbox receives mirror events
	claimed, err := env.store.ClaimPendingSyncOutboxEvents(store.ClaimOutboxOptions{Limit: 20})
	if err != nil {
		t.Fatalf("ClaimPendingSyncOutboxEvents: %v", err)
	}
	if len(claimed) < 2 {
		t.Fatalf("expected >=2 outbox events, got %d", len(claimed))
	}
}

func TestAdminBindingSync_PrebindAndDedup(t *testing.T) {
	env := setupTestEnv(t)
	secret := "test-secret"
	t.Setenv("ADMIN_SYNC_SHARED_SECRET", secret)

	bot, err := env.store.CreateBot(env.user.ID, "prebind-bot", "ilink", "provider-bot-1", json.RawMessage(`{"bot_id":"provider-bot-1","bot_token":"t"}`))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	if bot == nil || bot.ID == "" {
		t.Fatalf("CreateBot returned nil")
	}

	prebind := map[string]any{
		"type": "binding_prebind",
		"data": map[string]any{
			"event_id":   "evt-prebind-1",
			"event_time": 1715400002,
			"binding_id": "pb-1",
			"role_id":    "101",
			"bot_id":     bot.ID,
			"session_id": "session-1",
		},
	}
	body, _ := json.Marshal(prebind)
	req, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("prebind request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prebind status=%d", resp.StatusCode)
	}

	row, err := env.store.GetLatestPendingWechatBinding(bot.ID, "provider-bot-1", time.Now())
	if err != nil {
		t.Fatalf("GetLatestPendingWechatBinding: %v", err)
	}
	if row == nil {
		t.Fatalf("expected pending binding row")
	}
	if row.BindingID != "pb-1" {
		t.Fatalf("binding id mismatch: %s", row.BindingID)
	}

	req2, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("prebind dedup request: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("prebind dedup status=%d", resp2.StatusCode)
	}
}

func TestAdminBindingSync_PrebindAcceptsProviderBotID(t *testing.T) {
	env := setupTestEnv(t)
	secret := "test-secret"
	t.Setenv("ADMIN_SYNC_SHARED_SECRET", secret)

	bot, err := env.store.CreateBot(env.user.ID, "prebind-bot", "ilink", "provider-bot-1", json.RawMessage(`{"bot_id":"provider-bot-1","bot_token":"t"}`))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	prebind := map[string]any{
		"type": "binding_prebind",
		"data": map[string]any{
			"event_id":   "evt-prebind-provider-id",
			"event_time": 1715400002,
			"binding_id": "pb-provider-id",
			"role_id":    "101",
			"bot_id":     "provider-bot-1",
			"session_id": "session-1",
		},
	}
	body, _ := json.Marshal(prebind)
	req, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("prebind request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prebind with provider bot id should be accepted, status=%d", resp.StatusCode)
	}

	row, err := env.store.GetLatestPendingWechatBinding(bot.ID, "provider-bot-1", time.Now())
	if err != nil {
		t.Fatalf("GetLatestPendingWechatBinding: %v", err)
	}
	if row == nil {
		t.Fatalf("expected pending binding row")
	}
	if row.BotID != bot.ID {
		t.Fatalf("bot id mismatch: %s", row.BotID)
	}
	if row.ProviderBotID != "provider-bot-1" {
		t.Fatalf("provider bot id mismatch: %s", row.ProviderBotID)
	}
}

func TestAdminBindingSync_SignatureAndBlankPrompt(t *testing.T) {
	env := setupTestEnv(t)
	secret := "test-secret"
	t.Setenv("ADMIN_SYNC_SHARED_SECRET", secret)
	os.Unsetenv("AI_FULL_PROMPT_MAX_BYTES")

	payload := map[string]any{
		"type": "binding_profile_snapshot",
		"data": map[string]any{
			"event_id":          "evt-blank",
			"bot_id":            "bot-1",
			"sender_user_id":    "wx-1",
			"binding_id":        "bind-1",
			"full_prompt":       "   ",
			"prompt_version":    1,
			"source_updated_at": 100,
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Signature", "bad-sign")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request bad-sign: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 bad signature, got %d", resp.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodPost, env.ts.URL+"/api/internal/admin/sync/binding", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Admin-Signature", signAdmin(secret, body))
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("request blank prompt: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 blank prompt, got %d", resp2.StatusCode)
	}
}
