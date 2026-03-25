package mockserver

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestInjectInbound(t *testing.T) {
	e, _ := newTestEngine()

	e.InjectInbound(InboundRequest{
		Sender: "user1",
		Text:   "test message",
	})

	ctx := context.Background()
	result, err := e.GetUpdates(ctx, "")
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(result.Messages))
	}
	msg := result.Messages[0]
	if msg.FromUserID != "user1" {
		t.Errorf("FromUserID = %q, want %q", msg.FromUserID, "user1")
	}
	if msg.ToUserID != "bot" {
		t.Errorf("ToUserID = %q, want %q (default)", msg.ToUserID, "bot")
	}
	if msg.MessageID != 1 {
		t.Errorf("MessageID = %d, want 1", msg.MessageID)
	}
	if len(msg.ItemList) != 1 {
		t.Fatalf("ItemList len = %d, want 1", len(msg.ItemList))
	}
	if msg.ItemList[0].TextItem == nil || msg.ItemList[0].TextItem.Text != "test message" {
		t.Error("text item mismatch")
	}
	if msg.ContextToken == "" {
		t.Error("ContextToken should be auto-generated")
	}
}

func TestInjectInboundWithItems(t *testing.T) {
	e, _ := newTestEngine()

	e.InjectInbound(InboundRequest{
		Sender: "user1",
		Items: []ItemRequest{
			{Type: "text", Text: "item text"},
			{Type: "file", FileName: "doc.pdf", Data: []byte("pdf content")},
		},
	})

	ctx := context.Background()
	result, err := e.GetUpdates(ctx, "")
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	msg := result.Messages[0]
	if len(msg.ItemList) != 2 {
		t.Fatalf("ItemList len = %d, want 2", len(msg.ItemList))
	}
	if msg.ItemList[0].TextItem == nil {
		t.Error("expected text item at index 0")
	}
	if msg.ItemList[1].FileItem == nil {
		t.Error("expected file item at index 1")
	}
	if msg.ItemList[1].FileItem.Media == nil {
		t.Error("expected media reference on file item")
	}
}

func TestInjectInboundWithRecipient(t *testing.T) {
	e, _ := newTestEngine()

	e.InjectInbound(InboundRequest{
		Sender:    "user1",
		Recipient: "custom-bot",
		Text:      "hi",
	})

	result, _ := e.GetUpdates(context.Background(), "")
	if result.Messages[0].ToUserID != "custom-bot" {
		t.Errorf("ToUserID = %q, want %q", result.Messages[0].ToUserID, "custom-bot")
	}
}

func TestWaitForSent(t *testing.T) {
	e, _ := newTestEngine()

	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = e.SendText("user1", "delayed message", "")
	}()

	msgs := e.WaitForSent(t, 2*time.Second)
	if len(msgs) != 1 {
		t.Fatalf("sent = %d, want 1", len(msgs))
	}
	if msgs[0].Text != "delayed message" {
		t.Errorf("text = %q, want %q", msgs[0].Text, "delayed message")
	}
}

func TestAssertHelpers(t *testing.T) {
	e, _ := newTestEngine()

	_, _ = e.SendText("u1", "hello world", "")
	_, _ = e.SendText("u2", "goodbye world", "")

	e.AssertSentCount(t, 2)
	e.AssertSentContains(t, "hello")
	e.AssertSentContains(t, "goodbye")
}

func TestAssertSentCountFail(t *testing.T) {
	e, _ := newTestEngine()

	// Use a mock testing.TB to verify failure without failing the real test.
	ft := &fakeTB{}
	e.AssertSentCount(ft, 1)
	if !ft.errored {
		t.Error("AssertSentCount should have errored for count mismatch")
	}
}

func TestAssertSentContainsFail(t *testing.T) {
	e, _ := newTestEngine()
	_, _ = e.SendText("u1", "hello", "")

	ft := &fakeTB{}
	e.AssertSentContains(ft, "nonexistent")
	if !ft.errored {
		t.Error("AssertSentContains should have errored for missing substring")
	}
}

func TestExpireSession(t *testing.T) {
	e, _ := newTestEngine()

	e.InjectInbound(InboundRequest{Sender: "user1", Text: "before expire"})

	// Drain the message.
	_, _ = e.GetUpdates(context.Background(), "")

	e.ExpireSession()

	_, err := e.GetUpdates(context.Background(), "")
	if err == nil {
		t.Fatal("expected error after ExpireSession")
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %v, want session expired", err)
	}
}

func TestReset(t *testing.T) {
	e, _ := newTestEngine()

	// Add some state.
	_, _ = e.SendText("u1", "msg", "")
	e.InjectInbound(InboundRequest{Sender: "u1", Text: "in"})
	_ = e.SendMediaFile("u1", "", []byte("data"), "f.txt", "")
	e.SetToken("tok-123")
	e.SetStatus("connected")
	_ = e.SendTyping("u1", "", true)

	// Reset.
	e.Reset()

	if len(e.SentMessages()) != 0 {
		t.Error("sent should be empty after reset")
	}
	if len(e.ListMedia()) != 0 {
		t.Error("media should be empty after reset")
	}
	e.mu.Lock()
	token := e.token
	status := e.status
	seq := e.msgSeq
	typingLen := len(e.typing)
	e.mu.Unlock()

	if token != "" {
		t.Error("token should be empty after reset")
	}
	if status != "disconnected" {
		t.Errorf("status = %q, want 'disconnected'", status)
	}
	if seq != 0 {
		t.Errorf("msgSeq = %d, want 0", seq)
	}
	if typingLen != 0 {
		t.Error("typing should be empty after reset")
	}
}

func TestSetTokenAndStatus(t *testing.T) {
	e, _ := newTestEngine()

	e.SetToken("my-token")
	e.SetStatus("connected")

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.token != "my-token" {
		t.Errorf("token = %q, want %q", e.token, "my-token")
	}
	if e.status != "connected" {
		t.Errorf("status = %q, want %q", e.status, "connected")
	}
}

func TestMultipleInjectDrain(t *testing.T) {
	e, _ := newTestEngine()

	// Inject multiple messages.
	for i := 0; i < 5; i++ {
		e.InjectInbound(InboundRequest{Sender: "user1", Text: "msg"})
	}

	// Small delay to let channel buffer fill.
	time.Sleep(5 * time.Millisecond)

	result, err := e.GetUpdates(context.Background(), "")
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(result.Messages) != 5 {
		t.Errorf("got %d messages, want 5", len(result.Messages))
	}
}

// fakeTB is a minimal testing.TB implementation for testing assertion helpers.
type fakeTB struct {
	testing.TB
	errored bool
}

func (f *fakeTB) Helper() {}

func (f *fakeTB) Errorf(format string, args ...interface{}) {
	f.errored = true
}

func (f *fakeTB) Fatalf(format string, args ...interface{}) {
	f.errored = true
}
