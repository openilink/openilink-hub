package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

// ==================== Test 5: @handle message routing ====================

func TestMentionRouting_ParseAndMatchHandle(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{
				ID: "i1", AppID: "a1", BotID: "b1",
				Handle: "echo-work", Enabled: true,
				AppWebhookURL: "http://work.example.com",
			},
			{
				ID: "i2", AppID: "a1", BotID: "b1",
				Handle: "echo-family", Enabled: true,
				AppWebhookURL: "http://family.example.com",
			},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1"},
		},
	}

	d := newMatchDispatcher(store)

	// ParseMention("@echo-work hello") returns correct handle/text
	handle, command, text := ParseMention("@echo-work hello")
	if handle != "echo-work" {
		t.Errorf("handle = %q, want %q", handle, "echo-work")
	}
	if command != "" {
		t.Errorf("command = %q, want empty", command)
	}
	if text != "hello" {
		t.Errorf("text = %q, want %q", text, "hello")
	}

	// MatchHandle("b1", "echo-work") returns installation i1
	inst, err := d.MatchHandle("b1", "echo-work")
	if err != nil {
		t.Fatalf("MatchHandle error: %v", err)
	}
	if inst == nil || inst.ID != "i1" {
		t.Errorf("expected i1, got %v", inst)
	}

	// MatchHandle("b1", "echo-family") returns installation i2
	inst, err = d.MatchHandle("b1", "echo-family")
	if err != nil {
		t.Fatalf("MatchHandle error: %v", err)
	}
	if inst == nil || inst.ID != "i2" {
		t.Errorf("expected i2, got %v", inst)
	}

	// MatchHandle("b1", "nonexistent") returns nil
	inst, _ = d.MatchHandle("b1", "nonexistent")
	if inst != nil {
		t.Errorf("expected nil for nonexistent handle, got %v", inst)
	}
}

// ==================== Test 6: @handle /command routing ====================

func TestMentionRouting_HandleWithCommand(t *testing.T) {
	// Test ParseMention
	handle, command, text := ParseMention("@echo-work /echo hello")
	if handle != "echo-work" {
		t.Errorf("handle = %q, want %q", handle, "echo-work")
	}
	if command != "/echo" {
		t.Errorf("command = %q, want %q", command, "/echo")
	}
	if text != "hello" {
		t.Errorf("text = %q, want %q", text, "hello")
	}

	// Create mock HTTP server that records received command events
	var mu sync.Mutex
	var receivedEnvelope *eventEnvelope
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		// Handle url_verification
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
		mu.Lock()
		receivedEnvelope = &env
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reply": "echo: hello"})
	}))
	defer srv.Close()

	// Create dispatcher with mock store and deliver event
	secret := "echo-secret"
	inst := &store.AppInstallation{
		ID:            "inst-echo-1",
		AppID:         "app-echo-1",
		BotID:         "bot-echo-1",
		Handle:        "echo-work",
		AppWebhookSecret: secret,
		AppWebhookURL:    srv.URL,
		Enabled:       true,
	}

	d := newTestDispatcher(&mockLogDB{}, srv.Client())

	// Deliver a command event (simulating what tryDeliverMention does)
	event := NewEvent("command", map[string]any{
		"command": command,
		"text":    text,
		"handle":  handle,
	})

	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		t.Fatalf("DeliverEvent error: %v", err)
	}
	if result.Reply != "echo: hello" {
		t.Errorf("reply = %q, want %q", result.Reply, "echo: hello")
	}

	// Verify mock received the command event
	mu.Lock()
	env := receivedEnvelope
	mu.Unlock()

	if env == nil {
		t.Fatal("mock did not receive event")
	}
	if env.Type != "event" {
		t.Errorf("envelope type = %q, want %q", env.Type, "event")
	}
	if env.InstallationID != "inst-echo-1" {
		t.Errorf("installation_id = %q, want %q", env.InstallationID, "inst-echo-1")
	}

	// Verify event data contains command and handle
	if env.Event == nil {
		t.Fatal("event is nil")
	}
	if env.Event.Type != "command" {
		t.Errorf("event.type = %q, want %q", env.Event.Type, "command")
	}

	data, ok := env.Event.Data.(map[string]any)
	if !ok {
		// Data was JSON-unmarshaled, might need re-marshal/unmarshal
		raw, _ := json.Marshal(env.Event.Data)
		json.Unmarshal(raw, &data)
	}
	if data != nil {
		if cmd, _ := data["command"].(string); cmd != "/echo" {
			t.Errorf("event data command = %q, want %q", cmd, "/echo")
		}
		if h, _ := data["handle"].(string); h != "echo-work" {
			t.Errorf("event data handle = %q, want %q", h, "echo-work")
		}
	}
}

// ==================== Test 7: @handle without text ====================

func TestMentionRouting_HandleWithoutText(t *testing.T) {
	handle, command, text := ParseMention("@echo-work")
	if handle != "echo-work" {
		t.Errorf("handle = %q, want %q", handle, "echo-work")
	}
	if command != "" {
		t.Errorf("command = %q, want empty", command)
	}
	if text != "" {
		t.Errorf("text = %q, want empty", text)
	}
}

// ==================== Test 8: Multiple installs same app different handles ====================

func TestMentionRouting_MultipleInstallsSameAppDifferentHandles(t *testing.T) {
	store := &mockAppStore{
		installations: []store.AppInstallation{
			{
				ID: "i1", AppID: "a1", BotID: "b1",
				Handle: "github-work", Enabled: true,
				AppWebhookURL: "http://work.example.com",
			},
			{
				ID: "i2", AppID: "a1", BotID: "b1",
				Handle: "github-personal", Enabled: true,
				AppWebhookURL: "http://personal.example.com",
			},
		},
		apps: map[string]*store.App{
			"a1": {ID: "a1"},
		},
	}

	d := newMatchDispatcher(store)

	// Route to first handle
	inst1, err := d.MatchHandle("b1", "github-work")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if inst1 == nil {
		t.Fatal("expected match for github-work")
	}
	if inst1.ID != "i1" {
		t.Errorf("expected i1, got %q", inst1.ID)
	}
	if inst1.AppWebhookURL != "http://work.example.com" {
		t.Errorf("request_url = %q, want %q", inst1.AppWebhookURL, "http://work.example.com")
	}

	// Route to second handle
	inst2, err := d.MatchHandle("b1", "github-personal")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if inst2 == nil {
		t.Fatal("expected match for github-personal")
	}
	if inst2.ID != "i2" {
		t.Errorf("expected i2, got %q", inst2.ID)
	}
	if inst2.AppWebhookURL != "http://personal.example.com" {
		t.Errorf("request_url = %q, want %q", inst2.AppWebhookURL, "http://personal.example.com")
	}

	// Verify they are different installations of the same app
	if inst1.AppID != inst2.AppID {
		t.Errorf("both should have same AppID, got %q and %q", inst1.AppID, inst2.AppID)
	}
	if inst1.ID == inst2.ID {
		t.Error("should be different installations")
	}
}

// ==================== Additional: ParseMention edge cases for routing ====================

func TestMentionRouting_ParseMentionEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		handle  string
		command string
		text    string
	}{
		{
			name:    "handle with multiple words",
			input:   "@mybot hello world foo",
			handle:  "mybot",
			command: "",
			text:    "hello world foo",
		},
		{
			name:    "handle with command and multi-word text",
			input:   "@mybot /deploy us-east-1 production",
			handle:  "mybot",
			command: "/deploy",
			text:    "us-east-1 production",
		},
		{
			name:    "handle with only command, no args",
			input:   "@mybot /status",
			handle:  "mybot",
			command: "/status",
			text:    "",
		},
		{
			name:    "leading/trailing whitespace",
			input:   "  @mybot  /cmd  arg  ",
			handle:  "mybot",
			command: "/cmd",
			text:    "arg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handle, command, text := ParseMention(tt.input)
			if handle != tt.handle {
				t.Errorf("handle = %q, want %q", handle, tt.handle)
			}
			if command != tt.command {
				t.Errorf("command = %q, want %q", command, tt.command)
			}
			if text != tt.text {
				t.Errorf("text = %q, want %q", text, tt.text)
			}
		})
	}
}

// ==================== Full mention routing with event delivery ====================

func TestMentionRouting_FullMentionToEventDelivery(t *testing.T) {
	secret := "mention-secret"

	// Mock App that handles mention events
	m := newMockAppServer(secret, func(env eventEnvelope) any {
		if env.Type == "event" {
			return map[string]string{"reply": "got your mention"}
		}
		return nil
	})
	defer m.close()

	store := &mockAppStore{
		installations: []store.AppInstallation{
			{
				ID: "inst-m-1", AppID: "app-m-1", BotID: "bot-m-1",
				Handle: "echo-work", Enabled: true,
				AppWebhookURL:    m.server.URL,
				AppWebhookSecret: secret,
			},
		},
		apps: map[string]*store.App{
			"app-m-1": {ID: "app-m-1"},
		},
	}

	d := &Dispatcher{
		Client: m.server.Client(),
		dbLog:  &mockLogDB{},
		appDB:  store,
	}

	// Parse the mention
	handle, command, text := ParseMention("@echo-work hello there")
	if handle != "echo-work" || command != "" || text != "hello there" {
		t.Fatalf("parse unexpected: handle=%q cmd=%q text=%q", handle, command, text)
	}

	// Match the handle
	inst, err := d.MatchHandle("bot-m-1", handle)
	if err != nil {
		t.Fatalf("MatchHandle error: %v", err)
	}
	if inst == nil {
		t.Fatal("expected installation match")
	}

	// Deliver the event
	event := NewEvent("message.text", map[string]any{
		"content": text,
		"handle":  handle,
	})
	result, err := d.DeliverEvent(inst, event)
	if err != nil {
		t.Fatalf("DeliverEvent error: %v", err)
	}
	if result.Reply != "got your mention" {
		t.Errorf("reply = %q, want %q", result.Reply, "got your mention")
	}

	// Verify mock received it
	events := m.getEvents()
	if len(events) != 1 {
		t.Fatalf("mock received %d events, want 1", len(events))
	}
	if events[0].Envelope.InstallationID != "inst-m-1" {
		t.Errorf("installation_id = %q", events[0].Envelope.InstallationID)
	}
}
