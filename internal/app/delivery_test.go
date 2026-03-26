package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

var errFake = errors.New("fake error")

// --- Mock DB for delivery tests ---

type mockLogDB struct {
	createLogCalled atomic.Int32
	updateDelivered atomic.Int32
	updateFailed    atomic.Int32
	lastLogID       int64
	createLogErr    error
}

func (m *mockLogDB) CreateEventLog(_ *store.AppEventLog) (int64, error) {
	m.createLogCalled.Add(1)
	if m.createLogErr != nil {
		return 0, m.createLogErr
	}
	m.lastLogID++
	return m.lastLogID, nil
}

func (m *mockLogDB) UpdateEventLogDelivered(_ int64, _ int, _ string, _ int) error {
	m.updateDelivered.Add(1)
	return nil
}

func (m *mockLogDB) UpdateEventLogFailed(_ int64, _ string, _ int, _ int) error {
	m.updateFailed.Add(1)
	return nil
}

func newTestDispatcher(mock *mockLogDB, client *http.Client) *Dispatcher {
	return &Dispatcher{
		Client: client,
		dbLog:  mock,
	}
}

// --- Tests ---

func TestComputeSignature(t *testing.T) {
	secret := "mysecret"
	timestamp := "1700000000"
	body := []byte(`{"hello":"world"}`)

	got := computeSignature(secret, timestamp, body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte(":"))
	mac.Write(body)
	want := hex.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Errorf("computeSignature() = %s, want %s", got, want)
	}
}

func TestComputeSignatureDifferentSecrets(t *testing.T) {
	body := []byte(`{"test":1}`)
	ts := "1700000000"
	sig1 := computeSignature("secret-a", ts, body)
	sig2 := computeSignature("secret-b", ts, body)
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestComputeSignatureDifferentTimestamps(t *testing.T) {
	body := []byte(`{"test":1}`)
	sig1 := computeSignature("same", "100", body)
	sig2 := computeSignature("same", "200", body)
	if sig1 == sig2 {
		t.Error("different timestamps should produce different signatures")
	}
}

func TestNewEvent(t *testing.T) {
	data := map[string]string{"key": "value"}
	evt := NewEvent("message.text", data)

	if evt.Type != "message.text" {
		t.Errorf("Type = %q, want %q", evt.Type, "message.text")
	}
	if !strings.HasPrefix(evt.ID, "evt_") {
		t.Errorf("ID = %q, expected evt_ prefix", evt.ID)
	}
	if evt.Timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}
	if evt.Data == nil {
		t.Error("Data should not be nil")
	}
}

func TestDeliverEvent_Success(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	mock := &mockLogDB{}
	d := newTestDispatcher(mock, srv.Client())

	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "test-secret", AppWebhookURL: srv.URL,
	}
	event := NewEvent("message.text", map[string]string{"text": "hello"})

	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		t.Fatalf("DeliverEvent() error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}

	// Verify envelope structure.
	var envelope eventEnvelope
	if err := json.Unmarshal(receivedBody, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if envelope.V != envelopeVersion {
		t.Errorf("envelope.V = %d, want %d", envelope.V, envelopeVersion)
	}
	if envelope.Type != "event" {
		t.Errorf("envelope.Type = %q, want %q", envelope.Type, "event")
	}
	if !strings.HasPrefix(envelope.TraceID, "tr_") {
		t.Errorf("TraceID = %q, expected tr_ prefix", envelope.TraceID)
	}
	if envelope.InstallationID != "inst-1" {
		t.Errorf("InstallationID = %q, want %q", envelope.InstallationID, "inst-1")
	}
	if envelope.Bot.ID != "bot-1" {
		t.Errorf("Bot.ID = %q, want %q", envelope.Bot.ID, "bot-1")
	}
	if envelope.Event == nil {
		t.Fatal("envelope.Event should not be nil")
	}
	if envelope.Event.Type != "message.text" {
		t.Errorf("Event.Type = %q, want %q", envelope.Event.Type, "message.text")
	}

	// Verify headers.
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should be application/json")
	}
	if receivedHeaders.Get("X-App-Id") != "app-1" {
		t.Errorf("X-App-Id = %q, want %q", receivedHeaders.Get("X-App-Id"), "app-1")
	}
	if receivedHeaders.Get("X-Installation-Id") != "inst-1" {
		t.Errorf("X-Installation-Id = %q", receivedHeaders.Get("X-Installation-Id"))
	}
	sig := receivedHeaders.Get("X-Signature")
	if !strings.HasPrefix(sig, "sha256=") {
		t.Errorf("X-Signature = %q, expected sha256= prefix", sig)
	}

	// Verify DB log calls.
	if mock.createLogCalled.Load() != 1 {
		t.Errorf("CreateEventLog called %d times, want 1", mock.createLogCalled.Load())
	}
	if mock.updateDelivered.Load() != 1 {
		t.Errorf("UpdateEventLogDelivered called %d times, want 1", mock.updateDelivered.Load())
	}
}

func TestDeliverEvent_CommandEnvelopeType(t *testing.T) {
	var receivedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	_, err := d.DeliverEvent(inst, NewEvent("command", map[string]string{"command": "help"}))
	if err != nil {
		t.Fatalf("DeliverEvent() error: %v", err)
	}

	var envelope eventEnvelope
	json.Unmarshal(receivedBody, &envelope)
	if envelope.Type != "event" {
		t.Errorf("envelope.Type = %q, want %q", envelope.Type, "event")
	}
}

func TestDeliverEvent_SyncReply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{
			"reply":      "Hello from app!",
			"reply_type": "markdown",
		})
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Reply != "Hello from app!" {
		t.Errorf("Reply = %q, want %q", result.Reply, "Hello from app!")
	}
	if result.ReplyType != "markdown" {
		t.Errorf("ReplyType = %q, want %q", result.ReplyType, "markdown")
	}
}

func TestDeliverEvent_SyncReplyDefaultType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"reply": "hi"})
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.ReplyType != "text" {
		t.Errorf("ReplyType = %q, want %q", result.ReplyType, "text")
	}
}

func TestDeliverEvent_Failure500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"internal"}`))
	}))
	defer srv.Close()

	mock := &mockLogDB{}
	d := newTestDispatcher(mock, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, expected to contain '500'", err.Error())
	}
	if result.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", result.StatusCode)
	}
	if mock.updateFailed.Load() != 1 {
		t.Errorf("UpdateEventLogFailed called %d times, want 1", mock.updateFailed.Load())
	}
}

func TestDeliverEvent_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	mock := &mockLogDB{}
	client := &http.Client{Timeout: 100 * time.Millisecond}
	d := newTestDispatcher(mock, client)
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	_, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if mock.updateFailed.Load() != 1 {
		t.Errorf("UpdateEventLogFailed called %d times, want 1", mock.updateFailed.Load())
	}
}

func TestDeliverEvent_NoRequestURL(t *testing.T) {
	d := newTestDispatcher(&mockLogDB{}, http.DefaultClient)
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookURL: "",
	}

	_, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err == nil {
		t.Fatal("expected error for empty webhook_url")
	}
	if !strings.Contains(err.Error(), "no webhook_url") {
		t.Errorf("error = %q, expected 'no webhook_url'", err.Error())
	}
}

func TestDeliverEvent_EmptyResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Reply != "" {
		t.Errorf("Reply = %q, want empty", result.Reply)
	}
}

func TestDeliverEvent_SignatureVerification(t *testing.T) {
	var gotTimestamp, gotSignature string
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTimestamp = r.Header.Get("X-Timestamp")
		gotSignature = r.Header.Get("X-Signature")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	secret := "verify-me"
	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: secret, AppWebhookURL: srv.URL,
	}
	_, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	expected := computeSignature(secret, gotTimestamp, gotBody)
	if gotSignature != "sha256="+expected {
		t.Errorf("signature mismatch: got %q, want sha256=%s", gotSignature, expected)
	}
}

func TestDeliverEvent_CreateLogError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	mock := &mockLogDB{createLogErr: errFake}
	d := newTestDispatcher(mock, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}

	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("expected delivery to succeed despite log error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}
}

func TestDeliverEvent_TraceIDHeader(t *testing.T) {
	var gotTraceID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = r.Header.Get("X-Trace-Id")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	d := newTestDispatcher(&mockLogDB{}, srv.Client())
	inst := &store.AppInstallation{
		ID: "inst-1", AppID: "app-1", BotID: "bot-1",
		AppWebhookSecret: "secret", AppWebhookURL: srv.URL,
	}
	d.DeliverEvent(inst, NewEvent("message.text", nil))

	if !strings.HasPrefix(gotTraceID, "tr_") {
		t.Errorf("X-Trace-Id = %q, expected tr_ prefix", gotTraceID)
	}
}
