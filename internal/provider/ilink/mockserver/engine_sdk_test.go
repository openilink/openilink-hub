package mockserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
)

func newTestEngine(opts ...Option) (*Engine, *FakeClock) {
	fc := NewFakeClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	allOpts := append([]Option{WithClock(fc)}, opts...)
	return NewEngine(allOpts...), fc
}

func TestGetUpdatesBlocking(t *testing.T) {
	e, _ := newTestEngine()

	e.InjectInbound(InboundRequest{Sender: "user1", Text: "hello"})

	ctx := context.Background()
	result, err := e.GetUpdates(ctx, "")
	if err != nil {
		t.Fatalf("GetUpdates: %v", err)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(result.Messages))
	}
	if result.Messages[0].ItemList[0].TextItem.Text != "hello" {
		t.Errorf("text = %q, want %q", result.Messages[0].ItemList[0].TextItem.Text, "hello")
	}
	if result.SyncBuf == "" {
		t.Error("SyncBuf should not be empty")
	}
}

func TestGetUpdatesCancel(t *testing.T) {
	e, _ := newTestEngine()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.GetUpdates(ctx, "")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestGetUpdatesSessionExpired(t *testing.T) {
	e, _ := newTestEngine()
	e.ExpireSession()

	ctx := context.Background()
	_, err := e.GetUpdates(ctx, "")
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	if !strings.Contains(err.Error(), "session expired") {
		t.Errorf("error = %v, want session expired", err)
	}
}

func TestSendText(t *testing.T) {
	e, _ := newTestEngine()

	clientID, err := e.SendText("user1", "hi there", "ctx-123")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if clientID == "" {
		t.Fatal("clientID should not be empty")
	}
	if !strings.HasPrefix(clientID, "sdk-") {
		t.Errorf("clientID = %q, want prefix 'sdk-'", clientID)
	}

	msgs := e.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent count = %d, want 1", len(msgs))
	}
	if msgs[0].Text != "hi there" {
		t.Errorf("text = %q, want %q", msgs[0].Text, "hi there")
	}
	if msgs[0].Recipient != "user1" {
		t.Errorf("recipient = %q, want %q", msgs[0].Recipient, "user1")
	}
	if msgs[0].ContextToken != "ctx-123" {
		t.Errorf("contextToken = %q, want %q", msgs[0].ContextToken, "ctx-123")
	}
}

func TestPush(t *testing.T) {
	e, _ := newTestEngine()

	clientID, err := e.Push("user1", "push msg")
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if clientID == "" {
		t.Fatal("clientID should not be empty")
	}
	msgs := e.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %d, want 1", len(msgs))
	}
	if msgs[0].ContextToken != "" {
		t.Errorf("contextToken = %q, want empty", msgs[0].ContextToken)
	}
}

func TestSendMessage(t *testing.T) {
	e, _ := newTestEngine()

	msg := &ilink.WeixinMessage{
		ToUserID:     "user1",
		ContextToken: "ctx-abc",
		ItemList: []ilink.MessageItem{
			{Type: ilink.ItemText, TextItem: &ilink.TextItem{Text: "from SendMessage"}},
		},
	}
	if err := e.SendMessage(msg); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	msgs := e.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %d, want 1", len(msgs))
	}
	if msgs[0].Text != "from SendMessage" {
		t.Errorf("text = %q, want %q", msgs[0].Text, "from SendMessage")
	}
}

func TestUploadDownloadRoundtrip(t *testing.T) {
	e, _ := newTestEngine()

	plaintext := []byte("this is a test file content for upload/download roundtrip")

	// Generate key and encrypt.
	keyRaw, keyHex := generateAESKey()
	ciphertext, err := encryptMedia(plaintext, keyRaw)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	// GetUploadURL
	resp, err := e.GetUploadURL(&ilink.GetUploadURLReq{
		FileKey: "test-file-key",
		AESKey:  keyHex,
	})
	if err != nil {
		t.Fatalf("GetUploadURL: %v", err)
	}
	if resp.Ret != 0 {
		t.Fatalf("ret = %d, want 0", resp.Ret)
	}
	if resp.UploadParam == "" {
		t.Fatal("upload_param should not be empty")
	}

	// UploadToCDN
	eqp, err := e.UploadToCDN(resp.UploadParam, "test-file-key", ciphertext)
	if err != nil {
		t.Fatalf("UploadToCDN: %v", err)
	}
	if eqp == "" {
		t.Fatal("eqp should not be empty")
	}

	// DownloadFile
	aesKeyB64 := base64.StdEncoding.EncodeToString(keyRaw)
	got, err := e.DownloadFile(eqp, aesKeyB64)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("downloaded data mismatch: got %d bytes, want %d bytes", len(got), len(plaintext))
	}

	// DownloadVoice should work the same way.
	got2, err := e.DownloadVoice(eqp, aesKeyB64, 16000)
	if err != nil {
		t.Fatalf("DownloadVoice: %v", err)
	}
	if !bytes.Equal(got2, plaintext) {
		t.Fatal("DownloadVoice data mismatch")
	}
}

func TestDownloadWrongKey(t *testing.T) {
	e, _ := newTestEngine()

	plaintext := []byte("secret data for wrong key test")
	keyRaw, keyHex := generateAESKey()
	ciphertext, _ := encryptMedia(plaintext, keyRaw)

	resp, _ := e.GetUploadURL(&ilink.GetUploadURLReq{
		FileKey: "fk",
		AESKey:  keyHex,
	})
	eqp, _ := e.UploadToCDN(resp.UploadParam, "fk", ciphertext)

	// Download with a different key.
	wrongKey, _ := generateAESKey()
	wrongKeyB64 := base64.StdEncoding.EncodeToString(wrongKey)
	got, err := e.DownloadFile(eqp, wrongKeyB64)
	if err == nil && bytes.Equal(got, plaintext) {
		t.Fatal("expected error or different data with wrong key")
	}
}

func TestDownloadNotFound(t *testing.T) {
	e, _ := newTestEngine()

	_, err := e.DownloadFile("nonexistent-eqp", base64.StdEncoding.EncodeToString(make([]byte, 16)))
	if err == nil {
		t.Fatal("expected error for nonexistent EQP")
	}
}

func TestSendTyping(t *testing.T) {
	e, _ := newTestEngine()

	if err := e.SendTyping("user1", "ticket", true); err != nil {
		t.Fatalf("SendTyping: %v", err)
	}
	e.mu.Lock()
	typing := e.typing["user1"]
	e.mu.Unlock()
	if !typing {
		t.Error("expected typing=true for user1")
	}

	_ = e.SendTyping("user1", "ticket", false)
	e.mu.Lock()
	typing = e.typing["user1"]
	e.mu.Unlock()
	if typing {
		t.Error("expected typing=false for user1")
	}
}

func TestGetConfig(t *testing.T) {
	e, _ := newTestEngine()

	cfg, err := e.GetConfig("user1", "ctx")
	if err != nil {
		t.Fatalf("GetConfig: %v", err)
	}
	if !strings.HasPrefix(cfg.TypingTicket, "mock-ticket-") {
		t.Errorf("TypingTicket = %q, want prefix 'mock-ticket-'", cfg.TypingTicket)
	}
}

func TestQRStateMachine(t *testing.T) {
	e, _ := newTestEngine()

	// FetchQR
	qr, err := e.FetchQRCode()
	if err != nil {
		t.Fatalf("FetchQRCode: %v", err)
	}
	if !strings.HasPrefix(qr.QRCode, "mock-qr-") {
		t.Errorf("QRCode = %q, want prefix 'mock-qr-'", qr.QRCode)
	}

	// Poll — should block, so poll in goroutine.
	type pollResult struct {
		result *QRStatusResult
		err    error
	}
	ch := make(chan pollResult, 1)
	go func() {
		r, err := e.PollQRStatus(context.Background(), qr.QRCode)
		ch <- pollResult{r, err}
	}()

	// Give goroutine time to start polling.
	time.Sleep(10 * time.Millisecond)

	// ScanQR
	e.ScanQR()

	pr := <-ch
	if pr.err != nil {
		t.Fatalf("PollQRStatus after scan: %v", pr.err)
	}
	if pr.result.Status != "scanned" {
		t.Errorf("status = %q, want 'scanned'", pr.result.Status)
	}

	// Poll again — should block until confirm.
	go func() {
		r, err := e.PollQRStatus(context.Background(), qr.QRCode)
		ch <- pollResult{r, err}
	}()

	time.Sleep(10 * time.Millisecond)

	creds := Credentials{
		BotID:       "bot-123",
		BotToken:    "token-abc",
		BaseURL:     "https://example.com",
		ILinkUserID: "uid-456",
	}
	e.ConfirmQR(creds)

	pr = <-ch
	if pr.err != nil {
		t.Fatalf("PollQRStatus after confirm: %v", pr.err)
	}
	if pr.result.Status != "confirmed" {
		t.Errorf("status = %q, want 'confirmed'", pr.result.Status)
	}
	if pr.result.Creds == nil {
		t.Fatal("creds should not be nil")
	}
	if pr.result.Creds.BotID != "bot-123" {
		t.Errorf("BotID = %q, want %q", pr.result.Creds.BotID, "bot-123")
	}

	// Polling again should return immediately (terminal state).
	result, err := e.PollQRStatus(context.Background(), qr.QRCode)
	if err != nil {
		t.Fatalf("PollQRStatus terminal: %v", err)
	}
	if result.Status != "confirmed" {
		t.Errorf("terminal status = %q, want 'confirmed'", result.Status)
	}
}

func TestQRExpiry(t *testing.T) {
	e, fc := newTestEngine()

	_, err := e.FetchQRCode()
	if err != nil {
		t.Fatalf("FetchQRCode: %v", err)
	}

	// Advance past 30s expiry.
	fc.Advance(31 * time.Second)

	// Give goroutine time to process timer.
	time.Sleep(10 * time.Millisecond)

	result, err := e.PollQRStatus(context.Background(), "")
	if err != nil {
		t.Fatalf("PollQRStatus: %v", err)
	}
	if result.Status != "expired" {
		t.Errorf("status = %q, want 'expired'", result.Status)
	}
}

func TestQRAutoConfirm(t *testing.T) {
	e, _ := newTestEngine(WithAutoConfirmQR())

	qr, err := e.FetchQRCode()
	if err != nil {
		t.Fatalf("FetchQRCode: %v", err)
	}

	result, err := e.PollQRStatus(context.Background(), qr.QRCode)
	if err != nil {
		t.Fatalf("PollQRStatus: %v", err)
	}
	if result.Status != "confirmed" {
		t.Errorf("status = %q, want 'confirmed'", result.Status)
	}
	if result.Creds == nil {
		t.Fatal("creds should not be nil for auto-confirm")
	}
	if result.Creds.BotToken != "mock-bot-token" {
		t.Errorf("BotToken = %q, want 'mock-bot-token'", result.Creds.BotToken)
	}
}

func TestSendMediaFile(t *testing.T) {
	e, _ := newTestEngine()

	data := []byte("image data here")
	err := e.SendMediaFile("user1", "ctx-1", data, "photo.jpg", "a photo")
	if err != nil {
		t.Fatalf("SendMediaFile: %v", err)
	}

	msgs := e.SentMessages()
	if len(msgs) != 1 {
		t.Fatalf("sent = %d, want 1", len(msgs))
	}
	if msgs[0].FileName != "photo.jpg" {
		t.Errorf("fileName = %q, want %q", msgs[0].FileName, "photo.jpg")
	}
	if !bytes.Equal(msgs[0].MediaData, data) {
		t.Error("MediaData mismatch")
	}

	media := e.ListMedia()
	if len(media) != 1 {
		t.Fatalf("media count = %d, want 1", len(media))
	}
}

func TestPollQRNoSession(t *testing.T) {
	e, _ := newTestEngine()
	_, err := e.PollQRStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for no QR session")
	}
}
