package app

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestDeliverWithRetry_FirstAttemptSucceeds(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(200)
		w.Write([]byte(`{"reply":"ok"}`))
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	event := NewEvent("message.text", nil)

	result := d.DeliverWithRetry(inst, event)
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
	if result.Reply != "ok" {
		t.Errorf("Reply = %q, want %q", result.Reply, "ok")
	}

	// Give a moment for any background goroutines to start (they shouldn't).
	time.Sleep(50 * time.Millisecond)
	if callCount.Load() != 1 {
		t.Errorf("server called %d times, want 1 (no retries)", callCount.Load())
	}
}

func TestDeliverWithRetry_FirstAttemptFails(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n <= 2 {
			// First two calls fail.
			w.WriteHeader(500)
			w.Write([]byte(`{"error":"fail"}`))
			return
		}
		// Third call succeeds.
		w.WriteHeader(200)
		w.Write([]byte(`{"reply":"recovered"}`))
	}))
	defer srv.Close()

	// Override retry delays to be fast for tests.
	origDelays := retryDelays
	retryDelays = []time.Duration{0, 10 * time.Millisecond, 10 * time.Millisecond}
	defer func() { retryDelays = origDelays }()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	event := NewEvent("message.text", nil)

	// First attempt fails synchronously, retries happen in background.
	result := d.DeliverWithRetry(inst, event)
	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Wait for background retries.
	time.Sleep(200 * time.Millisecond)

	// Should have been called 3 times: initial + 2 retries.
	count := callCount.Load()
	if count < 2 {
		t.Errorf("server called %d times, want >= 2 (initial + retries)", count)
	}
}

func TestDeliverWithRetry_AllRetriesFail(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	origDelays := retryDelays
	retryDelays = []time.Duration{0, 5 * time.Millisecond, 5 * time.Millisecond}
	defer func() { retryDelays = origDelays }()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result := d.DeliverWithRetry(inst, NewEvent("message.text", nil))
	if result == nil {
		t.Fatal("result should not be nil even on failure")
	}

	// Wait for all retries.
	time.Sleep(200 * time.Millisecond)

	// 1 initial + 2 retries = 3.
	count := callCount.Load()
	if count != 3 {
		t.Errorf("server called %d times, want 3", count)
	}
}

func TestDeliverWithRetry_TimeoutThenRecover(t *testing.T) {
	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			// First call times out.
			time.Sleep(500 * time.Millisecond)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	origDelays := retryDelays
	retryDelays = []time.Duration{0, 10 * time.Millisecond, 10 * time.Millisecond}
	defer func() { retryDelays = origDelays }()

	client := &http.Client{Timeout: 50 * time.Millisecond}
	d := newTestDispatcher(&mockLogDB{}, client)
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result := d.DeliverWithRetry(inst, NewEvent("message.text", nil))
	if result == nil {
		t.Fatal("result should not be nil")
	}

	// Wait for retries.
	time.Sleep(300 * time.Millisecond)

	count := callCount.Load()
	if count < 2 {
		t.Errorf("server called %d times, want >= 2", count)
	}
}

func TestDeliverRetryAttempt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"reply":"retry ok"}`))
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result, err := d.deliverRetryAttempt(inst, NewEvent("message.text", nil), 1)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Reply != "retry ok" {
		t.Errorf("Reply = %q, want %q", result.Reply, "retry ok")
	}
}
