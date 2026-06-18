package bot

import (
	"context"
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

// fakeProvider captures the last outbound message for assertions.
type fakeProvider struct {
	sentText string
	sentTo   string
}

func (f *fakeProvider) Name() string                                       { return "fake" }
func (f *fakeProvider) Start(context.Context, provider.StartOptions) error { return nil }
func (f *fakeProvider) Stop()                                              {}
func (f *fakeProvider) Send(_ context.Context, msg provider.OutboundMessage) (string, error) {
	f.sentText = msg.Text
	f.sentTo = msg.Recipient
	return "client-1", nil
}
func (f *fakeProvider) SendTyping(context.Context, string, string, bool) error { return nil }
func (f *fakeProvider) GetConfig(context.Context, string, string) (*provider.BotConfig, error) {
	return &provider.BotConfig{}, nil
}
func (f *fakeProvider) DownloadMedia(context.Context, *provider.Media) ([]byte, error) {
	return nil, nil
}
func (f *fakeProvider) DownloadVoice(context.Context, *provider.Media, int) ([]byte, error) {
	return nil, nil
}
func (f *fakeProvider) Status() string { return "connected" }

// TestChannelTag covers the channel-reply prefix (issue #248): prefer handle,
// fall back to app name, empty when neither is set, and never an "@" prefix.
func TestChannelTag(t *testing.T) {
	cases := []struct {
		name string
		inst *store.AppInstallation
		want string
	}{
		{"handle preferred over app name", &store.AppInstallation{Handle: "openclaw", AppName: "OpenClaw"}, "【openclaw】 "},
		{"fall back to app name", &store.AppInstallation{AppName: "OpenClaw"}, "【OpenClaw】 "},
		{"empty when no name", &store.AppInstallation{}, ""},
		{"nil installation", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := channelTag(tc.inst); got != tc.want {
				t.Errorf("channelTag = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSendAppResult_ChannelTagPrefix verifies a webhook app's text reply is
// prefixed with its channel tag before being sent to the user (issue #248).
func TestSendAppResult_ChannelTagPrefix(t *testing.T) {
	ms := memstore.New()
	fp := &fakeProvider{}
	m := newTestManager(ms, appdelivery.NewWSHub())
	inst := &Instance{DBID: "bot-tag", Provider: fp}
	installation := &store.AppInstallation{ID: "i1", Handle: "openclaw", AppName: "OpenClaw"}
	tracer, root := newTestTracer("bot-tag")

	m.sendAppResult(inst, installation, "user-9", &appdelivery.DeliveryResult{Reply: "hello there"}, tracer, root)

	if want := "【openclaw】 hello there"; fp.sentText != want {
		t.Errorf("sent text = %q, want %q", fp.sentText, want)
	}
	if fp.sentTo != "user-9" {
		t.Errorf("sent to = %q, want %q", fp.sentTo, "user-9")
	}
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

	// Register a fake builtin so the channel fires.
	fakeHandler := &fakeBuiltinHandler{}
	builtin.Register(builtin.AppManifest{
		Slug:   appSlug,
		Events: []string{"message"},
		Scopes: []string{"message:read"},
	}, fakeHandler)
	t.Cleanup(func() { builtin.Deregister(appSlug) })

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
	t.Cleanup(func() { builtin.Deregister(appSlug) })

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

// TestTryDeliverMention_UnicodeWhitespace locks the third parsing path: an
// @handle followed by WeChat's U+2005 separator must still route to the
// installation. This is the root cause behind issue #248 — @handle "didn't
// work" in WeChat (which inserts U+2005 after a mention) so users fell back to /.
func TestTryDeliverMention_UnicodeWhitespace(t *testing.T) {
	const (
		botID  = "bot-mention-u2005"
		appID  = "app-mention-u2005"
		instID = "inst-mention-u2005"
	)

	ms := memstore.New()
	ms.AddInstallation(&store.AppInstallation{
		ID:      instID,
		AppID:   appID,
		BotID:   botID,
		Handle:  "echo-work",
		AppSlug: "echo",
		Enabled: true,
	})

	hub := appdelivery.NewWSHub()
	sendCh := make(chan []byte, 4)
	hub.Register(instID, &appdelivery.WSConn{InstID: instID, BotID: botID, Send: sendCh})

	m := newTestManager(ms, hub)
	tracer, root := newTestTracer(botID)
	// "@echo-work" + U+2005 (FOUR-PER-EM SPACE) + "hello".
	content := "@echo-work hello"
	msg, p := textMessage("user-1", content)

	if !m.tryDeliverMention(&Instance{DBID: botID}, msg, p, content, tracer, root) {
		t.Fatal("tryDeliverMention returned false — @handle with U+2005 failed to route")
	}

	select {
	case data := <-sendCh:
		var env map[string]any
		if err := json.Unmarshal(data, &env); err != nil {
			t.Fatalf("unmarshal ws payload: %v", err)
		}
		ev, _ := env["event"].(map[string]any)
		if ev["type"] != "message.text" {
			t.Errorf("event type = %v, want message.text", ev["type"])
		}
		evData, _ := ev["data"].(map[string]any)
		if evData["content"] != "hello" {
			t.Errorf("content = %v, want %q", evData["content"], "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out: mention event was not delivered after U+2005 split")
	}
}
