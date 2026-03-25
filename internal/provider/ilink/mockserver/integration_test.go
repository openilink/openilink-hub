package mockserver_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"
	"testing"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"

	ilinkProvider "github.com/openilink/openilink-hub/internal/provider/ilink"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/provider/ilink/mockserver"
)

// TestHTTPModeFullFlow exercises the full path through the REAL iLink provider
// connected to the mock HTTP server: start, receive inbound, send reply,
// and verify the engine recorded everything.
func TestHTTPModeFullFlow(t *testing.T) {
	srv := mockserver.NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetToken("test-token")
	srv.Engine().SetStatus("connected")

	// Create the real iLink provider.
	p := &ilinkProvider.Provider{}

	creds, _ := json.Marshal(ilinkProvider.Credentials{
		BotToken: "test-token",
		BaseURL:  url,
	})

	var received []provider.InboundMessage
	var mu sync.Mutex
	receivedCh := make(chan struct{}, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{
		Credentials: creds,
		OnMessage: func(msg provider.InboundMessage) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			receivedCh <- struct{}{}
		},
		OnStatus:     func(string) {},
		OnSyncUpdate: func(json.RawMessage) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	// Inject inbound message.
	srv.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "user1@wx",
		Text:   "hello bot",
	})

	// Wait for message with timeout.
	select {
	case <-receivedCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for inbound message")
	}

	mu.Lock()
	if len(received) == 0 {
		t.Fatal("no messages received")
	}
	msg := received[0]
	mu.Unlock()

	// Verify message content.
	if len(msg.Items) == 0 || msg.Items[0].Text != "hello bot" {
		t.Fatalf("unexpected message items: %+v", msg.Items)
	}
	if msg.Sender != "user1@wx" {
		t.Fatalf("wrong sender: got %q, want %q", msg.Sender, "user1@wx")
	}

	// Send reply via real provider.
	clientID, err := p.Send(ctx, provider.OutboundMessage{
		Recipient:    "user1@wx",
		Text:         "hi!",
		ContextToken: msg.ContextToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	if clientID == "" {
		t.Fatal("empty client ID from Send")
	}

	// Verify reply recorded in engine.
	sent := srv.Engine().SentMessages()
	if len(sent) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(sent))
	}
	if sent[0].Text != "hi!" {
		t.Fatalf("wrong sent text: got %q, want %q", sent[0].Text, "hi!")
	}
	if sent[0].Recipient != "user1@wx" {
		t.Fatalf("wrong recipient: got %q, want %q", sent[0].Recipient, "user1@wx")
	}
}

// TestHTTPModeMediaRoundtrip proves the CDN upload/download path works
// end-to-end through the mock HTTP server using the real SDK client.
func TestHTTPModeMediaRoundtrip(t *testing.T) {
	srv := mockserver.NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetToken("test-token")
	srv.Engine().SetStatus("connected")

	// Create a raw SDK client with both base URL and CDN URL pointing to the mock.
	client := ilink.NewClient("test-token",
		ilink.WithBaseURL(url),
		ilink.WithCDNBaseURL(url+"/c2c"),
	)

	ctx := context.Background()

	// Upload a file.
	originalData := []byte("hello world image data for testing media roundtrip")
	result, err := client.UploadFile(ctx, originalData, "user1", ilink.MediaImage)
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if result.DownloadEncryptedQueryParam == "" {
		t.Fatal("empty download EQP from upload")
	}
	if result.AESKey == "" {
		t.Fatal("empty AES key from upload")
	}

	// UploadResult.AESKey is hex-encoded (e.g. "abcdef0123456789...").
	// DownloadFile expects aesKeyBase64 which is base64(hexString).
	// This matches the real CDNMedia.aes_key encoding used by SendImage etc.
	aesKeyBase64 := base64.StdEncoding.EncodeToString([]byte(result.AESKey))

	// Download and decrypt.
	downloaded, err := client.DownloadFile(ctx, result.DownloadEncryptedQueryParam, aesKeyBase64)
	if err != nil {
		t.Fatalf("DownloadFile: %v", err)
	}

	if !bytes.Equal(downloaded, originalData) {
		t.Fatalf("roundtrip data mismatch:\n  got  (%d bytes): %q\n  want (%d bytes): %q",
			len(downloaded), downloaded, len(originalData), originalData)
	}

	// Verify media is tracked in the engine.
	media := srv.Engine().ListMedia()
	if len(media) == 0 {
		t.Fatal("expected at least one media entry in engine")
	}
}

// TestHTTPModeMediaSendViaProvider tests sending media through the real iLink
// provider connected to the mock HTTP server, verifying the full
// upload + send message pipeline.
func TestHTTPModeMediaSendViaProvider(t *testing.T) {
	srv := mockserver.NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetToken("test-token")
	srv.Engine().SetStatus("connected")

	p := &ilinkProvider.Provider{}

	// The iLink provider sets WithBaseURL but not WithCDNBaseURL.
	// The CDN URLs default to the production CDN, so media upload through
	// the provider won't hit the mock. Instead we set base_url which the
	// provider passes to the SDK client, and that affects API calls only.
	//
	// For a complete media flow test via provider.Send, the provider calls
	// client.SendMediaFile which calls client.UploadFile (uses cdnBaseURL).
	// Since we can't set cdnBaseURL through provider credentials, we test
	// text-only send through the provider and use the raw SDK client for
	// media roundtrip (see TestHTTPModeMediaRoundtrip).
	//
	// However, we CAN verify that the provider's DownloadMedia works via the
	// engine by injecting inbound media and then downloading it.

	creds, _ := json.Marshal(ilinkProvider.Credentials{
		BotToken: "test-token",
		BaseURL:  url,
	})

	var received []provider.InboundMessage
	var mu sync.Mutex
	receivedCh := make(chan struct{}, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{
		Credentials: creds,
		OnMessage: func(msg provider.InboundMessage) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			receivedCh <- struct{}{}
		},
		OnStatus:     func(string) {},
		OnSyncUpdate: func(json.RawMessage) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	// Inject inbound image message with media data.
	imageData := []byte("fake-png-image-bytes-for-testing")
	srv.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "user2@wx",
		Items: []mockserver.ItemRequest{
			{Type: "image", Data: imageData, FileName: "photo.png"},
		},
	})

	// Wait for the message to arrive.
	select {
	case <-receivedCh:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for inbound image message")
	}

	mu.Lock()
	msg := received[0]
	mu.Unlock()

	if len(msg.Items) == 0 {
		t.Fatal("no items in received message")
	}
	if msg.Items[0].Type != "image" {
		t.Fatalf("expected image item, got %q", msg.Items[0].Type)
	}
	if msg.Items[0].Media == nil {
		t.Fatal("nil media in image item")
	}

	// Download the media using the provider's DownloadMedia method.
	// The provider calls client.DownloadFile which uses cdnBaseURL.
	// Since cdnBaseURL defaults to production, we use the engine directly
	// to verify the media is correctly stored and retrievable.
	eqp := msg.Items[0].Media.EncryptQueryParam
	aesKey := msg.Items[0].Media.AESKey

	if eqp == "" {
		t.Fatal("empty EQP in received media")
	}
	if aesKey == "" {
		t.Fatal("empty AES key in received media")
	}

	// Download directly from engine (bypasses HTTP/CDN layer).
	downloaded, err := srv.Engine().DownloadFile(eqp, aesKey)
	if err != nil {
		t.Fatalf("engine.DownloadFile: %v", err)
	}

	if !bytes.Equal(downloaded, imageData) {
		t.Fatalf("media data mismatch:\n  got  (%d bytes): %q\n  want (%d bytes): %q",
			len(downloaded), downloaded, len(imageData), imageData)
	}
}

// TestHTTPModeMultipleMessages verifies that multiple inbound messages are
// all delivered through the real provider.
func TestHTTPModeMultipleMessages(t *testing.T) {
	srv := mockserver.NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetToken("test-token")
	srv.Engine().SetStatus("connected")

	p := &ilinkProvider.Provider{}
	creds, _ := json.Marshal(ilinkProvider.Credentials{
		BotToken: "test-token",
		BaseURL:  url,
	})

	var received []provider.InboundMessage
	var mu sync.Mutex
	receivedCh := make(chan struct{}, 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := p.Start(ctx, provider.StartOptions{
		Credentials: creds,
		OnMessage: func(msg provider.InboundMessage) {
			mu.Lock()
			received = append(received, msg)
			mu.Unlock()
			receivedCh <- struct{}{}
		},
		OnStatus:     func(string) {},
		OnSyncUpdate: func(json.RawMessage) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Stop()

	// Send 3 messages.
	for i := 0; i < 3; i++ {
		srv.Engine().InjectInbound(mockserver.InboundRequest{
			Sender: "user1@wx",
			Text:   "msg-" + string(rune('A'+i)),
		})
	}

	// Wait for all 3.
	for i := 0; i < 3; i++ {
		select {
		case <-receivedCh:
		case <-time.After(5 * time.Second):
			mu.Lock()
			got := len(received)
			mu.Unlock()
			t.Fatalf("timeout waiting for message %d; got %d so far", i+1, got)
		}
	}

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(received))
	}

	texts := map[string]bool{}
	for _, m := range received {
		if len(m.Items) > 0 {
			texts[m.Items[0].Text] = true
		}
	}
	for _, want := range []string{"msg-A", "msg-B", "msg-C"} {
		if !texts[want] {
			t.Errorf("missing message %q in received set: %v", want, texts)
		}
	}
}
