package mockserver

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/provider"
)

func TestProviderStartReceive(t *testing.T) {
	p := NewProvider()

	var (
		mu       sync.Mutex
		received []provider.InboundMessage
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{
		Credentials: MockCredentials(),
		OnMessage: func(msg provider.InboundMessage) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	if p.Status() != "connected" {
		t.Fatalf("status = %q, want connected", p.Status())
	}

	// Inject an inbound message.
	p.Engine().InjectInbound(InboundRequest{
		Sender: "user-1",
		Text:   "hello provider",
	})

	// Wait for the message to be received.
	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timed out waiting for inbound message")
		case <-time.After(10 * time.Millisecond):
		}
	}

	mu.Lock()
	msg := received[0]
	mu.Unlock()

	if msg.Sender != "user-1" {
		t.Errorf("sender = %q, want user-1", msg.Sender)
	}
	if len(msg.Items) != 1 || msg.Items[0].Type != "text" || msg.Items[0].Text != "hello provider" {
		t.Errorf("unexpected items: %+v", msg.Items)
	}
}

func TestProviderSend(t *testing.T) {
	p := NewProvider()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{Credentials: MockCredentials()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	id, err := p.Send(ctx, provider.OutboundMessage{
		Recipient:    "user-1",
		Text:         "hi there",
		ContextToken: "ctx-123",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if id == "" {
		t.Error("expected non-empty client ID")
	}

	sent := p.Engine().SentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	if sent[0].Text != "hi there" {
		t.Errorf("sent text = %q, want %q", sent[0].Text, "hi there")
	}
	if sent[0].ContextToken != "ctx-123" {
		t.Errorf("context token = %q, want ctx-123", sent[0].ContextToken)
	}
}

func TestProviderSendMedia(t *testing.T) {
	p := NewProvider()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{Credentials: MockCredentials()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	data := []byte("fake-image-data")
	_, err = p.Send(ctx, provider.OutboundMessage{
		Recipient: "user-1",
		Text:      "see attached",
		Data:      data,
		FileName:  "photo.png",
	})
	if err != nil {
		t.Fatalf("Send media: %v", err)
	}

	sent := p.Engine().SentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	if sent[0].FileName != "photo.png" {
		t.Errorf("file name = %q, want photo.png", sent[0].FileName)
	}
	if string(sent[0].MediaData) != string(data) {
		t.Error("media data mismatch")
	}

	// Verify media is stored in engine.
	media := p.Engine().ListMedia()
	if len(media) == 0 {
		t.Error("expected media to be stored in engine")
	}
}

func TestProviderStop(t *testing.T) {
	p := NewProvider()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{Credentials: MockCredentials()})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if p.Status() != "connected" {
		t.Fatalf("status = %q, want connected", p.Status())
	}

	p.Stop()

	if p.Status() != "disconnected" {
		t.Errorf("status after stop = %q, want disconnected", p.Status())
	}
}

func TestProviderBind(t *testing.T) {
	p := NewProvider()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := p.StartBind(ctx)
	if err != nil {
		t.Fatalf("StartBind: %v", err)
	}

	if session.QRURL == "" {
		t.Error("expected non-empty QR URL")
	}
	if session.SessionID == "" {
		t.Error("expected non-empty session ID")
	}

	// Simulate scanning.
	p.Engine().ScanQR()

	// Poll should see scanned.
	result, err := session.PollStatus(ctx)
	if err != nil {
		t.Fatalf("PollStatus after scan: %v", err)
	}
	if result.Status != "scanned" {
		t.Errorf("status = %q, want scanned", result.Status)
	}

	// Confirm QR.
	p.Engine().ConfirmQR(Credentials{
		BotID:       "test-bot",
		BotToken:    "test-token",
		ILinkUserID: "test-user",
	})

	// Poll should see confirmed with credentials.
	result, err = session.PollStatus(ctx)
	if err != nil {
		t.Fatalf("PollStatus after confirm: %v", err)
	}
	if result.Status != "confirmed" {
		t.Errorf("status = %q, want confirmed", result.Status)
	}
	if result.Credentials == nil {
		t.Fatal("expected credentials on confirmed")
	}

	var creds Credentials
	if err := json.Unmarshal(result.Credentials, &creds); err != nil {
		t.Fatalf("unmarshal credentials: %v", err)
	}
	if creds.BotToken != "test-token" {
		t.Errorf("bot token = %q, want test-token", creds.BotToken)
	}
}

func TestProviderBindAutoConfirm(t *testing.T) {
	p := NewProvider(WithAutoConfirmQR())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := p.StartBind(ctx)
	if err != nil {
		t.Fatalf("StartBind: %v", err)
	}

	// Should be immediately confirmed.
	result, err := session.PollStatus(ctx)
	if err != nil {
		t.Fatalf("PollStatus: %v", err)
	}
	if result.Status != "confirmed" {
		t.Errorf("status = %q, want confirmed", result.Status)
	}
	if result.Credentials == nil {
		t.Fatal("expected credentials on auto-confirm")
	}
}

func TestProviderSessionExpired(t *testing.T) {
	p := NewProvider()

	var (
		mu       sync.Mutex
		statuses []string
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{
		Credentials: MockCredentials(),
		OnStatus: func(status string) {
			mu.Lock()
			statuses = append(statuses, status)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer p.Stop()

	// Expire the session.
	p.Engine().ExpireSession()

	// Wait for session_expired status callback.
	deadline := time.After(2 * time.Second)
	for {
		mu.Lock()
		found := false
		for _, s := range statuses {
			if s == "session_expired" {
				found = true
				break
			}
		}
		mu.Unlock()
		if found {
			break
		}
		select {
		case <-deadline:
			mu.Lock()
			t.Fatalf("timed out waiting for session_expired; got statuses: %v", statuses)
			mu.Unlock()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestMockCredentials(t *testing.T) {
	raw := MockCredentials()
	if raw == nil {
		t.Fatal("MockCredentials returned nil")
	}

	var creds Credentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if creds.BotID != "mock-bot-id" {
		t.Errorf("bot_id = %q, want mock-bot-id", creds.BotID)
	}
	if creds.BotToken != "mock-token" {
		t.Errorf("bot_token = %q, want mock-token", creds.BotToken)
	}
	if creds.ILinkUserID != "mock-user" {
		t.Errorf("ilink_user_id = %q, want mock-user", creds.ILinkUserID)
	}
}
