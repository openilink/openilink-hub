package bridge

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/store"
)

// Handler implements builtin.Handler for the Bridge app.
type Handler struct{}

type bridgeConfig struct {
	ForwardURL string `json:"forward_url"`
}

func (h *Handler) HandleEvent(inst *store.AppInstallation, event *app.Event) error {
	var cfg bridgeConfig
	if err := json.Unmarshal(inst.Config, &cfg); err != nil {
		// Attempt to unwrap double-encoded config (stored as a JSON string
		// instead of a JSON object due to a prior frontend bug, issue #197).
		var s string
		if json.Unmarshal(inst.Config, &s) == nil {
			json.Unmarshal([]byte(s), &cfg)
		}
	}
	if cfg.ForwardURL == "" {
		return nil // not configured, skip silently
	}

	// Build event envelope (same format as standard app webhook)
	envelope := map[string]any{
		"v":               1,
		"type":            "event",
		"trace_id":        event.TraceID,
		"installation_id": inst.ID,
		"bot":             map[string]string{"id": inst.BotID},
		"event":           event,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Sign with webhook_secret as HMAC key (same as standard webhook signing)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(inst.AppWebhookSecret))
	mac.Write([]byte(timestamp + ":"))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	// POST to forward_url
	req, err := http.NewRequest("POST", cfg.ForwardURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Installation-Id", inst.ID)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Trace-Id", event.TraceID)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("bridge: forward failed", "inst", inst.ID, "url", cfg.ForwardURL, "err", err)
		return fmt.Errorf("forward: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(io.LimitReader(resp.Body, 1024)) // drain

	if resp.StatusCode >= 400 {
		slog.Error("bridge: forward error", "inst", inst.ID, "url", cfg.ForwardURL, "status", resp.StatusCode)
		return fmt.Errorf("forward returned %d", resp.StatusCode)
	}

	slog.Info("bridge: forwarded", "inst", inst.ID, "url", cfg.ForwardURL, "status", resp.StatusCode)
	return nil
}
