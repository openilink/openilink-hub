package bot

import (
	"encoding/json"
	"testing"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/builtin"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/memstore"
)

func TestResolveMediaURLs(t *testing.T) {
	baseURL := "https://hub.example.com"
	botDBID := "bot-123"

	items := []relay.MessageItem{
		{Type: "text", Text: "hello"},
		{
			Type:     "file",
			FileName: "doc.pdf",
			Media: &relay.Media{
				URL:       "https://wechat-cdn.example.com/encrypted-file",
				EQP:       "eqp-file-param",
				AESKey:    "abc123",
				FileSize:  1024,
				MediaType: "file",
			},
		},
		{
			Type: "image",
			Media: &relay.Media{
				URL:       "https://wechat-cdn.example.com/encrypted-image",
				EQP:       "eqp-image-param",
				AESKey:    "def456",
				MediaType: "image",
			},
		},
	}

	result := resolveMediaURLs(items, baseURL, botDBID)

	if result[0].Media != nil {
		t.Error("text item should have no media")
	}

	want := "https://hub.example.com/api/v1/channels/media?aes=abc123&bot=bot-123&ct=application%2Foctet-stream&eqp=eqp-file-param"
	if result[1].Media.URL != want {
		t.Errorf("file URL = %q, want %q", result[1].Media.URL, want)
	}
	if result[1].Media.FileSize != 1024 {
		t.Error("file size should be preserved")
	}
	if result[1].Media.EQP != "" {
		t.Errorf("file EQP should be cleared, got %q", result[1].Media.EQP)
	}
	if result[1].Media.AESKey != "" {
		t.Errorf("file AESKey should be cleared, got %q", result[1].Media.AESKey)
	}

	wantImg := "https://hub.example.com/api/v1/channels/media?aes=def456&bot=bot-123&ct=image%2Fjpeg&eqp=eqp-image-param"
	if result[2].Media.URL != wantImg {
		t.Errorf("image URL = %q, want %q", result[2].Media.URL, wantImg)
	}
	if result[2].Media.EQP != "" {
		t.Errorf("image EQP should be cleared, got %q", result[2].Media.EQP)
	}
	if result[2].Media.AESKey != "" {
		t.Errorf("image AESKey should be cleared, got %q", result[2].Media.AESKey)
	}

	// Original not mutated
	if items[1].Media.URL != "https://wechat-cdn.example.com/encrypted-file" {
		t.Error("original items should not be mutated")
	}
}

func TestResolveMediaURLs_NoMedia(t *testing.T) {
	items := []relay.MessageItem{
		{Type: "text", Text: "hello"},
	}
	result := resolveMediaURLs(items, "https://hub.example.com", "bot-123")
	if len(result) != 1 || result[0].Text != "hello" {
		t.Error("text-only items should pass through unchanged")
	}
}

func TestResolveMediaURLs_RefMsg(t *testing.T) {
	baseURL := "https://hub.example.com"
	botDBID := "bot-123"

	items := []relay.MessageItem{
		{
			Type: "text",
			Text: "quoting an image",
			RefMsg: &relay.RefMsg{
				Title: "original sender",
				Item: relay.MessageItem{
					Type: "image",
					Media: &relay.Media{
						URL:       "https://wechat-cdn.example.com/ref-image",
						EQP:       "eqp-ref-param",
						AESKey:    "refkey",
						MediaType: "image",
					},
				},
			},
		},
	}

	result := resolveMediaURLs(items, baseURL, botDBID)

	ref := result[0].RefMsg
	if ref == nil {
		t.Fatal("RefMsg should be present")
	}
	if ref.Item.Media == nil {
		t.Fatal("RefMsg item media should be present")
	}
	if ref.Item.Media.EQP != "" {
		t.Errorf("RefMsg EQP should be cleared, got %q", ref.Item.Media.EQP)
	}
	if ref.Item.Media.AESKey != "" {
		t.Errorf("RefMsg AESKey should be cleared, got %q", ref.Item.Media.AESKey)
	}
	wantURL := "https://hub.example.com/api/v1/channels/media?aes=refkey&bot=bot-123&ct=image%2Fjpeg&eqp=eqp-ref-param"
	if ref.Item.Media.URL != wantURL {
		t.Errorf("RefMsg media URL = %q, want %q", ref.Item.Media.URL, wantURL)
	}

	// Original not mutated
	if items[0].RefMsg.Item.Media.EQP != "eqp-ref-param" {
		t.Error("original RefMsg should not be mutated")
	}
}

func TestResolveMediaURLs_AlreadyStorageURL(t *testing.T) {
	items := []relay.MessageItem{
		{
			Type: "image",
			Media: &relay.Media{
				URL:       "https://storage.example.com/bot-123/img.jpg",
				EQP:       "",
				AESKey:    "",
				MediaType: "image",
			},
		},
	}
	result := resolveMediaURLs(items, "https://hub.example.com", "bot-123")
	if result[0].Media.URL != "https://storage.example.com/bot-123/img.jpg" {
		t.Error("items without EQP should keep original URL")
	}
}

// --- helpers for app_dispatch unit tests ---

// noopTraceStore satisfies store.TraceStore with no-ops so spans don't
// need a real database during these unit tests.
type noopTraceStore struct{}

func (noopTraceStore) InsertSpan(traceID, spanID, parentSpanID, name, kind, statusCode, statusMessage string,
	startTime, endTime int64, attrsJSON, eventsJSON []byte, botID string) error {
	return nil
}
func (noopTraceStore) AppendSpan(traceID, botID, name, kind, statusCode, statusMessage string, attrs map[string]any) error {
	return nil
}
func (noopTraceStore) ListRootSpans(botID string, limit int) ([]store.TraceSpan, error) {
	return nil, nil
}
func (noopTraceStore) ListSpansByTrace(traceID string) ([]store.TraceSpan, error) { return nil, nil }

// fakeBuiltinHandler records whether HandleEvent was invoked.
type fakeBuiltinHandler struct{ called bool }

func (h *fakeBuiltinHandler) HandleEvent(_ *store.AppInstallation, _ *appdelivery.Event) error {
	h.called = true
	return nil
}

// newTestManager builds a minimal Manager for app_dispatch tests.
func newTestManager(ms *memstore.Store, hub *appdelivery.WSHub) *Manager {
	disp := appdelivery.NewDispatcher(ms)
	return &Manager{
		instances: make(map[string]*Instance),
		store:     ms,
		appDisp:   disp,
		appWSHub:  hub,
	}
}

func newTestTracer(botID string) (*store.Tracer, *store.SpanBuilder) {
	tracer := store.NewTracer(noopTraceStore{}, botID)
	root := tracer.Start("test", store.SpanKindInternal, nil)
	return tracer, root
}

func textMessage(sender, content string) (provider.InboundMessage, parsedMessage) {
	msg := provider.InboundMessage{
		ExternalID: "msg-test",
		Sender:     sender,
		Items:      []provider.MessageItem{{Type: "text", Text: content}},
	}
	p := parsedMessage{msgType: "text", content: content}
	return msg, p
}

// --- tests ---

// TestDeliverToApps_WSAndBuiltinBothFire verifies the fix for issue #208:
// when a builtin app (bridge) has an active WebSocket connection, events must
// be delivered to BOTH the WS client AND the builtin handler independently.
//
// Before the fix, the builtin handler ran first and did continue, so the WS
// branch was never reached and the connected client received nothing.
func TestDeliverToApps_WSAndBuiltinBothFire(t *testing.T) {
	const (
		botID   = "bot-ws-priority"
		appID   = "app-ws-priority"
		instID  = "inst-ws-priority"
		appSlug = "fake-bridge-ws"
	)

	// Register a fake builtin so the old short-circuit code path exists.
	fakeHandler := &fakeBuiltinHandler{}
	builtin.Register(builtin.AppManifest{
		Slug:   appSlug,
		Events: []string{"message"},
		Scopes: []string{"message:read"},
	}, fakeHandler)

	ms := memstore.New()
	ms.AddApp(&store.App{
		ID:       appID,
		Slug:     appSlug,
		Registry: "builtin",
		Events:   json.RawMessage(`["message"]`),
		Scopes:   json.RawMessage(`["message:read","message:write"]`),
		Tools:    json.RawMessage(`[]`),
		Status:   "active",
	})
	ms.AddInstallation(&store.AppInstallation{
		ID:          instID,
		AppID:       appID,
		BotID:       botID,
		AppSlug:     appSlug,
		AppRegistry: "builtin",
		AppToken:    "tok-ws-priority",
		Scopes:      json.RawMessage(`["message:read","message:write"]`),
		Enabled:     true,
	})

	// Register a WS connection for this installation.
	hub := appdelivery.NewWSHub()
	sendCh := make(chan []byte, 4)
	hub.Register(instID, &appdelivery.WSConn{
		InstID: instID,
		BotID:  botID,
		Send:   sendCh,
	})

	m := newTestManager(ms, hub)
	tracer, root := newTestTracer(botID)
	msg, p := textMessage("user-1", "hello bridge")

	m.deliverToApps(&Instance{DBID: botID}, msg, p, tracer, root)

	// The event must arrive on the WS channel.
	select {
	case data := <-sendCh:
		var env map[string]any
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatalf("unmarshal ws payload: %v", err)
		}
		if env["type"] != "event" {
			t.Errorf("type = %v, want 'event'", env["type"])
		}
		if env["installation_id"] != instID {
			t.Errorf("installation_id = %v, want %q", env["installation_id"], instID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out: event was not delivered to the WebSocket")
	}

	// The builtin handler must ALSO have been called — channels are independent.
	if !fakeHandler.called {
		t.Error("builtin handler was not called — all delivery channels should fire independently")
	}
}

// TestDeliverToApps_BuiltinHandlerWhenNoWS verifies that the builtin handler
// is still invoked when there is no active WebSocket connection.
func TestDeliverToApps_BuiltinHandlerWhenNoWS(t *testing.T) {
	const (
		botID   = "bot-no-ws"
		appID   = "app-no-ws"
		instID  = "inst-no-ws"
		appSlug = "fake-bridge-noWs"
	)

	fakeHandler := &fakeBuiltinHandler{}
	builtin.Register(builtin.AppManifest{
		Slug:   appSlug,
		Events: []string{"message"},
		Scopes: []string{"message:read"},
	}, fakeHandler)

	ms := memstore.New()
	ms.AddApp(&store.App{
		ID:       appID,
		Slug:     appSlug,
		Registry: "builtin",
		Events:   json.RawMessage(`["message"]`),
		Scopes:   json.RawMessage(`["message:read","message:write"]`),
		Tools:    json.RawMessage(`[]`),
		Status:   "active",
	})
	ms.AddInstallation(&store.AppInstallation{
		ID:          instID,
		AppID:       appID,
		BotID:       botID,
		AppSlug:     appSlug,
		AppRegistry: "builtin",
		AppToken:    "tok-no-ws",
		Scopes:      json.RawMessage(`["message:read","message:write"]`),
		Enabled:     true,
	})

	// Empty hub — no active WS connection.
	hub := appdelivery.NewWSHub()

	m := newTestManager(ms, hub)
	tracer, root := newTestTracer(botID)
	msg, p := textMessage("user-1", "hi no ws")

	m.deliverToApps(&Instance{DBID: botID}, msg, p, tracer, root)

	if !fakeHandler.called {
		t.Error("builtin handler was not called when no WS connection was active")
	}
}
