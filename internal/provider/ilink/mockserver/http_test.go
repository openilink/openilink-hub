package mockserver

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
)

func TestHTTPGetUpdates(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetToken("test-token")
	srv.Engine().SetStatus("connected")

	// Inject a message before polling.
	srv.Engine().InjectInbound(InboundRequest{Sender: "user1", Text: "hello"})

	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetUpdates(ctx, "buf-0")
	if err != nil {
		t.Fatalf("GetUpdates error: %v", err)
	}
	if resp.Ret != 0 {
		t.Fatalf("GetUpdates ret = %d, want 0", resp.Ret)
	}
	if len(resp.Msgs) != 1 {
		t.Fatalf("got %d msgs, want 1", len(resp.Msgs))
	}
	msg := resp.Msgs[0]
	if msg.FromUserID != "user1" {
		t.Errorf("FromUserID = %q, want %q", msg.FromUserID, "user1")
	}
	if len(msg.ItemList) == 0 || msg.ItemList[0].TextItem == nil || msg.ItemList[0].TextItem.Text != "hello" {
		t.Errorf("message text mismatch, got items: %+v", msg.ItemList)
	}
}

func TestHTTPGetUpdatesSessionExpired(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	srv.Engine().SetStatus("session_expired")

	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resp, err := client.GetUpdates(ctx, "buf-0")
	if err != nil {
		t.Fatalf("GetUpdates error: %v", err)
	}
	if resp.ErrCode != -14 {
		t.Errorf("ErrCode = %d, want -14", resp.ErrCode)
	}
}

func TestHTTPSendMessage(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))

	ctx := context.Background()
	_, err := client.SendText(ctx, "recipient1", "hi there", "ctx-abc")
	if err != nil {
		t.Fatalf("SendText error: %v", err)
	}

	sent := srv.Engine().SentMessages()
	if len(sent) != 1 {
		t.Fatalf("sent count = %d, want 1", len(sent))
	}
	if sent[0].Recipient != "recipient1" {
		t.Errorf("recipient = %q, want %q", sent[0].Recipient, "recipient1")
	}
	if sent[0].Text != "hi there" {
		t.Errorf("text = %q, want %q", sent[0].Text, "hi there")
	}
}

func TestHTTPGetConfig(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))

	ctx := context.Background()
	resp, err := client.GetConfig(ctx, "user1", "ctx-token")
	if err != nil {
		t.Fatalf("GetConfig error: %v", err)
	}
	if resp.Ret != 0 {
		t.Errorf("ret = %d, want 0", resp.Ret)
	}
	if resp.TypingTicket == "" {
		t.Error("typing_ticket is empty")
	}
}

func TestHTTPSendTyping(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))

	ctx := context.Background()
	err := client.SendTyping(ctx, "user1", "ticket-abc", ilink.Typing)
	if err != nil {
		t.Fatalf("SendTyping error: %v", err)
	}

	// Verify typing state in engine.
	srv.Engine().mu.Lock()
	isTyping := srv.Engine().typing["user1"]
	srv.Engine().mu.Unlock()
	if !isTyping {
		t.Error("typing state not set for user1")
	}
}

func TestHTTPUploadDownloadRoundtrip(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	client := ilink.NewClient("test-token",
		ilink.WithBaseURL(url),
		ilink.WithCDNBaseURL(url+"/c2c"),
	)

	ctx := context.Background()
	plaintext := []byte("test file content for upload roundtrip")

	result, err := client.UploadFile(ctx, plaintext, "user1", ilink.MediaFile)
	if err != nil {
		t.Fatalf("UploadFile error: %v", err)
	}

	if result.FileKey == "" {
		t.Error("FileKey is empty")
	}
	if result.DownloadEncryptedQueryParam == "" {
		t.Error("DownloadEncryptedQueryParam is empty")
	}

	// Download and decrypt via SDK.
	// UploadResult.AESKey is hex-encoded; DownloadFile expects base64(hex), matching CDNMedia.aes_key format.
	aesKeyBase64 := base64.StdEncoding.EncodeToString([]byte(result.AESKey))
	downloaded, err := client.DownloadFile(ctx, result.DownloadEncryptedQueryParam, aesKeyBase64)
	if err != nil {
		t.Fatalf("DownloadFile error: %v", err)
	}

	if !bytes.Equal(downloaded, plaintext) {
		t.Errorf("downloaded data mismatch: got %d bytes, want %d bytes", len(downloaded), len(plaintext))
	}
}

func TestHTTPQRFlow(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	client := ilink.NewClient("", ilink.WithBaseURL(url))
	ctx := context.Background()

	// 1. Fetch QR code.
	qr, err := client.FetchQRCode(ctx)
	if err != nil {
		t.Fatalf("FetchQRCode error: %v", err)
	}
	if qr.QRCode == "" {
		t.Fatal("QRCode is empty")
	}

	// 2. Simulate scan via control endpoint.
	scanResp, err := http.Post(url+"/mock/qr/scan", "application/json", nil)
	if err != nil {
		t.Fatalf("mock scan error: %v", err)
	}
	scanResp.Body.Close()

	// 3. Poll — should get "scaned" (the SDK spelling).
	status, err := client.PollQRStatus(ctx, qr.QRCode)
	if err != nil {
		t.Fatalf("PollQRStatus error: %v", err)
	}
	if status.Status != "scaned" {
		t.Errorf("status = %q, want %q", status.Status, "scaned")
	}

	// 4. Confirm via control endpoint.
	credsJSON, _ := json.Marshal(Credentials{
		BotID:       "bot-123",
		BotToken:    "token-xyz",
		BaseURL:     "https://test.api/",
		ILinkUserID: "user-456",
	})
	confirmResp, err := http.Post(url+"/mock/qr/confirm", "application/json", bytes.NewReader(credsJSON))
	if err != nil {
		t.Fatalf("mock confirm error: %v", err)
	}
	confirmResp.Body.Close()

	// 5. Poll — should get "confirmed" with credentials.
	status, err = client.PollQRStatus(ctx, qr.QRCode)
	if err != nil {
		t.Fatalf("PollQRStatus error: %v", err)
	}
	if status.Status != "confirmed" {
		t.Errorf("status = %q, want %q", status.Status, "confirmed")
	}
	if status.BotToken != "token-xyz" {
		t.Errorf("BotToken = %q, want %q", status.BotToken, "token-xyz")
	}
	if status.ILinkBotID != "bot-123" {
		t.Errorf("ILinkBotID = %q, want %q", status.ILinkBotID, "bot-123")
	}
	if status.ILinkUserID != "user-456" {
		t.Errorf("ILinkUserID = %q, want %q", status.ILinkUserID, "user-456")
	}
}

func TestHTTPControlInboundAndSent(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	// Inject via control endpoint.
	body, _ := json.Marshal(InboundRequest{Sender: "ext-user", Text: "from control"})
	resp, err := http.Post(url+"/mock/inbound", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /mock/inbound error: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// GET /mock/sent should be empty initially.
	sentResp, err := http.Get(url + "/mock/sent")
	if err != nil {
		t.Fatalf("GET /mock/sent error: %v", err)
	}
	defer sentResp.Body.Close()
	var sentMsgs []SentMessage
	json.NewDecoder(sentResp.Body).Decode(&sentMsgs)
	if len(sentMsgs) != 0 {
		t.Errorf("sent count = %d, want 0", len(sentMsgs))
	}
}

func TestHTTPControlReset(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	// Send a message first.
	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))
	client.SendText(context.Background(), "user1", "hello", "ctx-1")

	if len(srv.Engine().SentMessages()) != 1 {
		t.Fatal("expected 1 sent message before reset")
	}

	// Reset via control endpoint.
	resp, err := http.Post(url+"/mock/reset", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /mock/reset error: %v", err)
	}
	resp.Body.Close()

	if len(srv.Engine().SentMessages()) != 0 {
		t.Error("expected 0 sent messages after reset")
	}
}

func TestHTTPControlSessionExpire(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	// Expire session via control endpoint.
	resp, err := http.Post(url+"/mock/session/expire", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /mock/session/expire error: %v", err)
	}
	resp.Body.Close()

	// GetUpdates should now return errcode -14.
	client := ilink.NewClient("test-token", ilink.WithBaseURL(url))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	updResp, err := client.GetUpdates(ctx, "buf-0")
	if err != nil {
		t.Fatalf("GetUpdates error: %v", err)
	}
	if updResp.ErrCode != -14 {
		t.Errorf("ErrCode = %d, want -14", updResp.ErrCode)
	}
}

func TestHTTPControlListMedia(t *testing.T) {
	srv := NewHTTPServer()
	url := srv.Start()
	defer srv.Close()

	// Upload a file to populate media.
	aesKey := make([]byte, 16)
	rand.Read(aesKey)
	plaintext := []byte("media test data!")
	ciphertext, _ := ilink.EncryptAESECB(plaintext, aesKey)

	uploadReq := &ilink.GetUploadURLReq{
		FileKey:  "fk-001",
		AESKey:   hex.EncodeToString(aesKey),
		FileSize: int64(len(ciphertext)),
		RawSize:  int64(len(plaintext)),
	}
	client := ilink.NewClient("test-token",
		ilink.WithBaseURL(url),
		ilink.WithCDNBaseURL(url+"/c2c"),
	)
	ctx := context.Background()

	uploadResp, err := client.GetUploadURL(ctx, uploadReq)
	if err != nil {
		t.Fatalf("GetUploadURL error: %v", err)
	}

	// Upload ciphertext to CDN.
	cdnURL := ilink.BuildCDNUploadURL(url+"/c2c", uploadResp.UploadParam, "fk-001")
	req, _ := http.NewRequest(http.MethodPost, cdnURL, bytes.NewReader(ciphertext))
	req.Header.Set("Content-Type", "application/octet-stream")
	cdnResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("CDN upload error: %v", err)
	}
	io.Copy(io.Discard, cdnResp.Body)
	cdnResp.Body.Close()

	// List media via control endpoint.
	mediaResp, err := http.Get(url + "/mock/media")
	if err != nil {
		t.Fatalf("GET /mock/media error: %v", err)
	}
	defer mediaResp.Body.Close()

	var mediaList []MediaInfo
	json.NewDecoder(mediaResp.Body).Decode(&mediaList)
	if len(mediaList) != 1 {
		t.Fatalf("media count = %d, want 1", len(mediaList))
	}
}
