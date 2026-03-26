package bridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/store"
)

func TestHandler_ForwardEvent(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	inst := &store.AppInstallation{
		ID:               "inst-123",
		BotID:            "bot-456",
		Config:           json.RawMessage(`{"forward_url":"` + ts.URL + `"}`),
		AppWebhookSecret: "test-secret",
	}
	event := &app.Event{
		Type:      "message.text",
		ID:        "evt-789",
		Timestamp: 1234567890,
		TraceID:   "tr-abc",
		Data:      map[string]any{"content": "hello"},
	}

	h := &Handler{}
	err := h.HandleEvent(inst, event)
	if err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}

	// Verify body was forwarded
	if len(receivedBody) == 0 {
		t.Fatal("expected non-empty body")
	}
	var envelope map[string]any
	if err := json.Unmarshal(receivedBody, &envelope); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if envelope["type"] != "event" {
		t.Errorf("type = %v, want 'event'", envelope["type"])
	}
	if envelope["installation_id"] != "inst-123" {
		t.Errorf("installation_id = %v", envelope["installation_id"])
	}

	// Verify headers
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", receivedHeaders.Get("Content-Type"))
	}
	if receivedHeaders.Get("X-Installation-Id") != "inst-123" {
		t.Errorf("X-Installation-Id = %q", receivedHeaders.Get("X-Installation-Id"))
	}
	if receivedHeaders.Get("X-Trace-Id") != "tr-abc" {
		t.Errorf("X-Trace-Id = %q", receivedHeaders.Get("X-Trace-Id"))
	}

	// Verify HMAC signature
	sig := receivedHeaders.Get("X-Signature")
	ts2 := receivedHeaders.Get("X-Timestamp")
	if !strings.HasPrefix(sig, "sha256=") {
		t.Fatalf("signature missing sha256= prefix: %q", sig)
	}
	mac := hmac.New(sha256.New, []byte("test-secret"))
	mac.Write([]byte(ts2 + ":"))
	mac.Write(receivedBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if sig != expectedSig {
		t.Errorf("signature mismatch:\n  got:  %s\n  want: %s", sig, expectedSig)
	}
}

func TestHandler_MissingForwardURL(t *testing.T) {
	inst := &store.AppInstallation{
		ID:     "inst-123",
		Config: json.RawMessage(`{}`),
	}
	event := &app.Event{
		Type: "message.text",
		ID:   "evt-789",
	}

	h := &Handler{}
	err := h.HandleEvent(inst, event)
	if err != nil {
		t.Fatalf("expected nil error for missing forward_url, got: %v", err)
	}
}

func TestHandler_ForwardError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	inst := &store.AppInstallation{
		ID:               "inst-123",
		BotID:            "bot-456",
		Config:           json.RawMessage(`{"forward_url":"` + ts.URL + `"}`),
		AppWebhookSecret: "test-secret",
	}
	event := &app.Event{
		Type: "message.text",
		ID:   "evt-789",
	}

	h := &Handler{}
	err := h.HandleEvent(inst, event)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code: %v", err)
	}
}

func TestHandler_InvalidURL(t *testing.T) {
	inst := &store.AppInstallation{
		ID:               "inst-123",
		BotID:            "bot-456",
		Config:           json.RawMessage(`{"forward_url":"http://192.0.2.1:1"}`),
		AppWebhookSecret: "test-secret",
	}
	event := &app.Event{
		Type: "message.text",
		ID:   "evt-789",
	}

	h := &Handler{}
	err := h.HandleEvent(inst, event)
	if err == nil {
		t.Fatal("expected error for unreachable URL")
	}
}
