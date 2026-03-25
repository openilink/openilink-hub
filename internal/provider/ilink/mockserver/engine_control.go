package mockserver

import (
	"encoding/base64"
	"math/rand"
	"strings"
	"testing"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
)

// InjectInbound simulates an incoming message from a user.
func (e *Engine) InjectInbound(req InboundRequest) {
	e.mu.Lock()

	e.msgSeq++
	msgID := e.msgSeq

	recipient := req.Recipient
	if recipient == "" {
		recipient = "bot"
	}

	contextToken := req.ContextToken
	if contextToken == "" {
		contextToken = "ctx-" + randomHex(8)
	}

	msg := &ilink.WeixinMessage{
		MessageID:    msgID,
		FromUserID:   req.Sender,
		ToUserID:     recipient,
		CreateTimeMs: e.clock.Now().UnixMilli(),
		ContextToken: contextToken,
		SessionID:    req.SessionID,
		GroupID:      req.GroupID,
		MessageState: ilink.MessageState(req.MessageState),
	}

	if len(req.Items) > 0 {
		for _, ir := range req.Items {
			item := e.convertItemRequest(ir)
			msg.ItemList = append(msg.ItemList, item)
		}
	} else if req.Text != "" {
		msg.ItemList = []ilink.MessageItem{
			{
				Type:     ilink.ItemText,
				TextItem: &ilink.TextItem{Text: req.Text},
			},
		}
	}

	// Check message loss.
	if e.opts.messageLoss > 0 && rand.Float64() < e.opts.messageLoss {
		e.mu.Unlock()
		return
	}

	inbound := e.inbound
	e.mu.Unlock()

	// Push to inbound channel (non-blocking if full, but channel is buffered).
	select {
	case inbound <- msg:
	default:
	}
}

// convertItemRequest converts an ItemRequest to an ilink.MessageItem.
// Must be called with e.mu held.
func (e *Engine) convertItemRequest(ir ItemRequest) ilink.MessageItem {
	item := ilink.MessageItem{}
	switch strings.ToLower(ir.Type) {
	case "text":
		item.Type = ilink.ItemText
		item.TextItem = &ilink.TextItem{Text: ir.Text}
	case "image":
		item.Type = ilink.ItemImage
		item.ImageItem = &ilink.ImageItem{}
		if len(ir.Data) > 0 {
			media := e.storeMediaLocked(ir.Data, ir.FileName, int(ilink.MediaImage))
			item.ImageItem.Media = media
		}
	case "voice":
		item.Type = ilink.ItemVoice
		item.VoiceItem = &ilink.VoiceItem{}
		if len(ir.Data) > 0 {
			media := e.storeMediaLocked(ir.Data, ir.FileName, int(ilink.MediaVoice))
			item.VoiceItem.Media = media
		}
	case "file":
		item.Type = ilink.ItemFile
		item.FileItem = &ilink.FileItem{FileName: ir.FileName}
		if len(ir.Data) > 0 {
			media := e.storeMediaLocked(ir.Data, ir.FileName, int(ilink.MediaFile))
			item.FileItem.Media = media
		}
	case "video":
		item.Type = ilink.ItemVideo
		item.VideoItem = &ilink.VideoItem{}
		if len(ir.Data) > 0 {
			media := e.storeMediaLocked(ir.Data, ir.FileName, int(ilink.MediaVideo))
			item.VideoItem.Media = media
		}
	}
	return item
}

// storeMediaLocked encrypts data and stores it in e.media, returning a CDNMedia reference.
// Must be called with e.mu held.
func (e *Engine) storeMediaLocked(data []byte, fileName string, mediaType int) *ilink.CDNMedia {
	keyRaw, keyHex := generateAESKey()
	ciphertext, err := encryptMedia(data, keyRaw)
	if err != nil {
		return nil
	}
	eqp := generateEQP()
	e.media[eqp] = &mediaEntry{
		ciphertext: ciphertext,
		aesKeyHex:  keyHex,
		aesKeyRaw:  keyRaw,
		mediaType:  mediaType,
		fileName:   fileName,
		rawSize:    len(data),
		uploadedAt: e.clock.Now(),
	}
	return &ilink.CDNMedia{
		EncryptQueryParam: eqp,
		AESKey:            base64.StdEncoding.EncodeToString(keyRaw),
		EncryptType:       ilink.EncryptAES128ECB,
	}
}

// SentMessages returns a copy of all sent messages.
func (e *Engine) SentMessages() []SentMessage {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make([]SentMessage, len(e.sent))
	copy(result, e.sent)
	return result
}

// WaitForSent waits until at least one new message has been sent, or timeout is reached.
func (e *Engine) WaitForSent(t testing.TB, timeout time.Duration) []SentMessage {
	t.Helper()
	e.mu.Lock()
	initial := len(e.sent)
	e.mu.Unlock()

	deadline := time.After(timeout)
	for {
		e.mu.Lock()
		current := len(e.sent)
		e.mu.Unlock()
		if current > initial {
			return e.SentMessages()
		}
		select {
		case <-deadline:
			t.Fatalf("WaitForSent: timed out after %v waiting for new messages", timeout)
			return nil
		case <-e.sentCh:
			// Check again in next loop iteration.
		}
	}
}

// AssertSentCount asserts that exactly n messages have been sent.
func (e *Engine) AssertSentCount(t testing.TB, n int) {
	t.Helper()
	e.mu.Lock()
	got := len(e.sent)
	e.mu.Unlock()
	if got != n {
		t.Errorf("sent count = %d, want %d", got, n)
	}
}

// AssertSentContains asserts that at least one sent message contains the substring.
func (e *Engine) AssertSentContains(t testing.TB, substring string) {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, m := range e.sent {
		if strings.Contains(m.Text, substring) {
			return
		}
	}
	t.Errorf("no sent message contains %q", substring)
}

// ExpireSession expires the current session, unblocking any GetUpdates calls.
func (e *Engine) ExpireSession() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = "session_expired"
	close(e.inbound)
	e.inbound = make(chan *ilink.WeixinMessage, 100)
}

// ScanQR simulates scanning the QR code.
func (e *Engine) ScanQR() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.qr != nil {
		e.qr.state = "scanned"
		select {
		case e.qr.ch <- struct{}{}:
		default:
		}
	}
}

// ConfirmQR simulates confirming the QR code login with credentials.
func (e *Engine) ConfirmQR(creds Credentials) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.qr != nil {
		e.qr.state = "confirmed"
		e.qr.creds = creds
		select {
		case e.qr.ch <- struct{}{}:
		default:
		}
	}
}

// ListMedia returns info about all stored media files.
func (e *Engine) ListMedia() []MediaInfo {
	e.mu.Lock()
	defer e.mu.Unlock()
	var result []MediaInfo
	for eqp, entry := range e.media {
		result = append(result, MediaInfo{
			EQP:       eqp,
			AESKey:    entry.aesKeyHex,
			MediaType: entry.mediaType,
			FileName:  entry.fileName,
			Size:      entry.rawSize,
		})
	}
	return result
}

// Reset clears all engine state and recreates channels.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.token = ""
	e.status = "disconnected"
	e.inbound = make(chan *ilink.WeixinMessage, 100)
	e.sent = nil
	e.sentCh = make(chan struct{}, 1)
	e.syncBuf = ""
	e.msgSeq = 0
	e.media = make(map[string]*mediaEntry)
	e.qr = nil
	e.typing = make(map[string]bool)
}

// SetToken sets the engine's authentication token.
func (e *Engine) SetToken(token string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.token = token
}

// SetStatus sets the engine's connection status.
func (e *Engine) SetStatus(status string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.status = status
}
