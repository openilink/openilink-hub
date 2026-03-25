package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

// receivedEvent tracks what a mock app server receives.
type receivedEvent struct {
	Headers  http.Header
	Body     []byte
	Envelope eventEnvelope
}

// mockAppServer simulates an external App that receives events from the Hub.
type mockAppServer struct {
	mu       sync.Mutex
	events   []receivedEvent
	secret   string
	replyFn  func(env eventEnvelope) any
	server   *httptest.Server
}

func newMockAppServer(secret string, replyFn func(eventEnvelope) any) *mockAppServer {
	m := &mockAppServer{
		secret:  secret,
		replyFn: replyFn,
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.handler))
	return m
}

func (m *mockAppServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	// Handle url_verification challenge.
	var probe struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if json.Unmarshal(body, &probe) == nil && probe.Type == "url_verification" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"challenge": probe.Challenge})
		return
	}

	var env eventEnvelope
	json.Unmarshal(body, &env)

	re := receivedEvent{
		Headers:  r.Header.Clone(),
		Body:     body,
		Envelope: env,
	}
	m.mu.Lock()
	m.events = append(m.events, re)
	m.mu.Unlock()

	// Verify signature if secret is set.
	if m.secret != "" {
		timestamp := r.Header.Get("X-Timestamp")
		sigHeader := r.Header.Get("X-Signature")
		expected := computeSignature(m.secret, timestamp, body)
		if sigHeader != "sha256="+expected {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"invalid signature"}`))
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)

	if m.replyFn != nil {
		reply := m.replyFn(env)
		if reply != nil {
			json.NewEncoder(w).Encode(reply)
			return
		}
	}
	w.Write([]byte(`{"ok":true}`))
}

func (m *mockAppServer) close() {
	m.server.Close()
}

func (m *mockAppServer) getEvents() []receivedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]receivedEvent, len(m.events))
	copy(cp, m.events)
	return cp
}

// --- Integration tests ---

func TestMockApp_URLVerification(t *testing.T) {
	m := newMockAppServer("", nil)
	defer m.close()

	challenge := "test-challenge-123"
	payload, _ := json.Marshal(map[string]string{
		"type":      "url_verification",
		"challenge": challenge,
	})

	resp, err := http.Post(m.server.URL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		t.Fatalf("POST error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	if result["challenge"] != challenge {
		t.Errorf("challenge = %q, want %q", result["challenge"], challenge)
	}
}

func TestMockApp_EventDeliveryWithSignature(t *testing.T) {
	secret := "integration-test-secret"
	m := newMockAppServer(secret, nil)
	defer m.close()

	mock := &mockLogDB{}
	d := newTestDispatcher(mock, m.server.Client())

	inst := &store.AppInstallation{
		ID: "inst-int-1", AppID: "app-int-1", BotID: "bot-int-1",
		AppSigningSecret: secret, AppRequestURL: m.server.URL,
	}
	event := NewEvent("message.text", map[string]string{"text": "hello world"})

	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		t.Fatalf("DeliverEvent error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", result.StatusCode)
	}

	// Verify mock received the event.
	events := m.getEvents()
	if len(events) != 1 {
		t.Fatalf("mock received %d events, want 1", len(events))
	}

	re := events[0]
	if re.Envelope.Type != "event" {
		t.Errorf("envelope type = %q, want %q", re.Envelope.Type, "event")
	}
	if re.Envelope.InstallationID != "inst-int-1" {
		t.Errorf("installation_id = %q", re.Envelope.InstallationID)
	}
	if re.Envelope.Event.Type != "message.text" {
		t.Errorf("event type = %q", re.Envelope.Event.Type)
	}

	// Verify signature was correctly sent and validated by mock.
	sigHeader := re.Headers.Get("X-Signature")
	ts := re.Headers.Get("X-Timestamp")
	expectedSig := computeSignature(secret, ts, re.Body)
	if sigHeader != "sha256="+expectedSig {
		t.Error("signature verification failed")
	}

	// Verify event log was created.
	if mock.createLogCalled.Load() != 1 {
		t.Errorf("CreateEventLog called %d times", mock.createLogCalled.Load())
	}
	if mock.updateDelivered.Load() != 1 {
		t.Errorf("UpdateEventLogDelivered called %d times", mock.updateDelivered.Load())
	}
}

func TestMockApp_CommandDelivery(t *testing.T) {
	secret := "cmd-secret"
	m := newMockAppServer(secret, func(env eventEnvelope) any {
		if env.Event != nil && env.Event.Type == "command" {
			return map[string]string{"reply": "command received"}
		}
		return nil
	})
	defer m.close()

	d := newTestDispatcher(&mockLogDB{}, m.server.Client())
	inst := &store.AppInstallation{
		ID: "inst-cmd-1", AppID: "app-cmd-1", BotID: "bot-cmd-1",
		AppSigningSecret: secret, AppRequestURL: m.server.URL,
	}
	event := NewEvent("command", map[string]any{
		"command": "deploy",
		"text":    "production",
	})

	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		t.Fatalf("DeliverEvent error: %v", err)
	}

	// Verify envelope type.
	events := m.getEvents()
	if len(events) != 1 {
		t.Fatalf("mock received %d events, want 1", len(events))
	}
	if events[0].Envelope.Type != "event" {
		t.Errorf("envelope type = %q, want %q", events[0].Envelope.Type, "event")
	}

	// Verify sync reply.
	if result.Reply != "command received" {
		t.Errorf("Reply = %q, want %q", result.Reply, "command received")
	}
}

func TestMockApp_SyncReplyInResult(t *testing.T) {
	m := newMockAppServer("", func(env eventEnvelope) any {
		return map[string]string{
			"reply":      "Hello from the app!",
			"reply_type": "markdown",
		}
	})
	defer m.close()

	d := newTestDispatcher(&mockLogDB{}, m.server.Client())
	inst := &store.AppInstallation{
		ID: "inst-r-1", AppID: "app-r-1", BotID: "bot-r-1",
		AppSigningSecret: "secret", AppRequestURL: m.server.URL,
	}

	result, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Reply != "Hello from the app!" {
		t.Errorf("Reply = %q", result.Reply)
	}
	if result.ReplyType != "markdown" {
		t.Errorf("ReplyType = %q", result.ReplyType)
	}
}

func TestMockApp_InvalidSignatureRejected(t *testing.T) {
	secret := "real-secret"
	m := newMockAppServer(secret, nil)
	defer m.close()

	// Use a DIFFERENT secret for the installation, causing signature mismatch.
	d := newTestDispatcher(&mockLogDB{}, m.server.Client())
	inst := &store.AppInstallation{
		ID: "inst-bad-1", AppID: "app-bad-1", BotID: "bot-bad-1",
		AppSigningSecret: "wrong-secret", AppRequestURL: m.server.URL,
	}

	_, err := d.DeliverEvent(inst, NewEvent("message.text", nil))
	// The mock returns 401 for bad signature.
	if err == nil {
		t.Fatal("expected error for bad signature")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error = %q, expected 401", err.Error())
	}
}

func TestMockApp_MultipleEventsTracked(t *testing.T) {
	m := newMockAppServer("", nil)
	defer m.close()

	d := newTestDispatcher(&mockLogDB{}, m.server.Client())
	inst := &store.AppInstallation{
		ID: "inst-m-1", AppID: "app-m-1", BotID: "bot-m-1",
		AppSigningSecret: "secret", AppRequestURL: m.server.URL,
	}

	for i := 0; i < 5; i++ {
		_, err := d.DeliverEvent(inst, NewEvent("message.text", map[string]int{"n": i}))
		if err != nil {
			t.Fatalf("delivery %d error: %v", i, err)
		}
	}

	events := m.getEvents()
	if len(events) != 5 {
		t.Errorf("mock received %d events, want 5", len(events))
	}
}

func TestMockApp_FullFlowWithMatchAndDeliver(t *testing.T) {
	secret := "full-flow-secret"
	m := newMockAppServer(secret, func(env eventEnvelope) any {
		if env.Event != nil && env.Event.Type == "command" {
			return map[string]string{"reply": "command handled"}
		}
		return map[string]string{"reply": "event ack"}
	})
	defer m.close()

	cmds, _ := json.Marshal([]store.AppTool{{Name: "run_deploy", Command: "deploy"}})
	events, _ := json.Marshal([]string{"message"})

	store := &mockAppStore{
		installations: []store.AppInstallation{
			{
				ID: "inst-ff-1", AppID: "app-ff-1", BotID: "bot-ff-1",
				Enabled: true, AppRequestURL: m.server.URL,
				AppSigningSecret: secret,
			},
		},
		apps: map[string]*store.App{
			"app-ff-1": {ID: "app-ff-1", Tools: cmds, Events: events},
		},
	}

	d := &Dispatcher{
		Client: m.server.Client(),
		dbLog:  &mockLogDB{},
		appDB:  store,
	}

	// Step 1: Match and deliver a command.
	matched, cmd, args, err := d.MatchCommand("bot-ff-1", "/deploy production")
	if err != nil {
		t.Fatalf("MatchCommand error: %v", err)
	}
	if cmd != "deploy" {
		t.Errorf("cmd = %q", cmd)
	}
	if args != "production" {
		t.Errorf("args = %q", args)
	}
	if len(matched) != 1 {
		t.Fatalf("matched %d, want 1", len(matched))
	}

	cmdEvent := NewEvent("command", map[string]any{
		"command": cmd,
		"text":    args,
	})
	result, err := d.DeliverEvent(&matched[0], cmdEvent)
	if err != nil {
		t.Fatalf("command delivery error: %v", err)
	}
	if result.Reply != "command handled" {
		t.Errorf("Reply = %q", result.Reply)
	}

	// Step 2: Match and deliver a message event.
	eventMatched, err := d.MatchEvent("bot-ff-1", "message.text")
	if err != nil {
		t.Fatalf("MatchEvent error: %v", err)
	}
	if len(eventMatched) != 1 {
		t.Fatalf("event matched %d, want 1", len(eventMatched))
	}

	msgEvent := NewEvent("message.text", map[string]string{"text": "hello"})
	result, err = d.DeliverEvent(&eventMatched[0], msgEvent)
	if err != nil {
		t.Fatalf("event delivery error: %v", err)
	}
	if result.Reply != "event ack" {
		t.Errorf("Reply = %q", result.Reply)
	}

	// Verify mock received both events.
	allEvents := m.getEvents()
	if len(allEvents) != 2 {
		t.Errorf("mock received %d events, want 2", len(allEvents))
	}

	// Both should have envelope type "event".
	if allEvents[0].Envelope.Type != "event" {
		t.Errorf("first event type = %q", allEvents[0].Envelope.Type)
	}
	if allEvents[1].Envelope.Type != "event" {
		t.Errorf("second event type = %q", allEvents[1].Envelope.Type)
	}

	// Verify signatures on both.
	for i, re := range allEvents {
		ts := re.Headers.Get("X-Timestamp")
		sigHeader := re.Headers.Get("X-Signature")

		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts))
		mac.Write([]byte(":"))
		mac.Write(re.Body)
		expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

		if sigHeader != expected {
			t.Errorf("event[%d] signature mismatch", i)
		}
	}
}
