package mockserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
)

var syncCounter atomic.Int64

// randomHex generates n random bytes and returns their hex encoding.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GetUpdates blocks on the inbound channel waiting for messages.
// It drains all available messages after the first one arrives.
func (e *Engine) GetUpdates(ctx context.Context, buf string) (*GetUpdatesResult, error) {
	e.mu.Lock()
	if e.status == "session_expired" {
		e.mu.Unlock()
		return nil, errors.New("session expired")
	}
	latency := e.opts.latency
	inbound := e.inbound
	e.mu.Unlock()

	// Wait for the first message or context cancellation.
	var msgs []*ilink.WeixinMessage
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-inbound:
		if !ok {
			return nil, errors.New("session expired")
		}
		msgs = append(msgs, msg)
	}

	// Drain any additional messages available without blocking.
	for {
		select {
		case msg, ok := <-inbound:
			if !ok {
				goto done
			}
			msgs = append(msgs, msg)
		default:
			goto done
		}
	}
done:

	if latency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-e.clock.After(latency):
		}
	}

	seq := syncCounter.Add(1)
	return &GetUpdatesResult{
		Messages: msgs,
		SyncBuf:  fmt.Sprintf("sync-%d", seq),
	}, nil
}

// SendText sends a text message and returns the generated client ID.
func (e *Engine) SendText(recipient, text, contextToken string) (string, error) {
	clientID := fmt.Sprintf("sdk-%d-%s", e.clock.Now().UnixMilli(), randomHex(4))

	e.mu.Lock()
	e.sent = append(e.sent, SentMessage{
		Recipient:    recipient,
		Text:         text,
		ContextToken: contextToken,
		ClientID:     clientID,
		Timestamp:    e.clock.Now().UnixMilli(),
	})
	e.mu.Unlock()

	// Notify sentCh non-blocking.
	select {
	case e.sentCh <- struct{}{}:
	default:
	}

	return clientID, nil
}

// Push sends a text message without a context token.
func (e *Engine) Push(recipient, text string) (string, error) {
	return e.SendText(recipient, text, "")
}

// SendMessage sends a full WeixinMessage, extracting text and media from items.
func (e *Engine) SendMessage(msg *ilink.WeixinMessage) error {
	var text string
	for _, item := range msg.ItemList {
		if item.TextItem != nil {
			text = item.TextItem.Text
			break
		}
	}

	clientID := fmt.Sprintf("sdk-%d-%s", e.clock.Now().UnixMilli(), randomHex(4))

	itemsJSON, _ := json.Marshal(msg.ItemList)

	sm := SentMessage{
		Recipient:    msg.ToUserID,
		Text:         text,
		ContextToken: msg.ContextToken,
		ClientID:     clientID,
		Items:        itemsJSON,
		Timestamp:    e.clock.Now().UnixMilli(),
	}

	// Check for media items and try to decrypt from e.media.
	for _, item := range msg.ItemList {
		var media *ilink.CDNMedia
		switch {
		case item.VoiceItem != nil && item.VoiceItem.Media != nil:
			media = item.VoiceItem.Media
		case item.ImageItem != nil && item.ImageItem.Media != nil:
			media = item.ImageItem.Media
		case item.FileItem != nil && item.FileItem.Media != nil:
			media = item.FileItem.Media
		case item.VideoItem != nil && item.VideoItem.Media != nil:
			media = item.VideoItem.Media
		}
		if media != nil {
			e.mu.Lock()
			entry, ok := e.media[media.EncryptQueryParam]
			e.mu.Unlock()
			if ok {
				key, err := parseAESKey(media.AESKey)
				if err == nil {
					plaintext, err := decryptMedia(entry.ciphertext, key)
					if err == nil {
						sm.MediaData = plaintext
					}
				}
			}
		}
	}

	e.mu.Lock()
	e.sent = append(e.sent, sm)
	e.mu.Unlock()

	select {
	case e.sentCh <- struct{}{}:
	default:
	}

	return nil
}

// SendMediaFile encrypts and stores media data, then records a sent message.
func (e *Engine) SendMediaFile(recipient, contextToken string, data []byte, fileName, text string) error {
	keyRaw, keyHex := generateAESKey()
	ciphertext, err := encryptMedia(data, keyRaw)
	if err != nil {
		return fmt.Errorf("encrypt media: %w", err)
	}

	eqp := generateEQP()

	e.mu.Lock()
	e.media[eqp] = &mediaEntry{
		ciphertext: ciphertext,
		aesKeyHex:  keyHex,
		aesKeyRaw:  keyRaw,
		fileName:   fileName,
		rawSize:    len(data),
		uploadedAt: e.clock.Now(),
	}
	e.mu.Unlock()

	clientID := fmt.Sprintf("sdk-%d-%s", e.clock.Now().UnixMilli(), randomHex(4))

	e.mu.Lock()
	e.sent = append(e.sent, SentMessage{
		Recipient:    recipient,
		Text:         text,
		ContextToken: contextToken,
		ClientID:     clientID,
		MediaData:    data,
		FileName:     fileName,
		Timestamp:    e.clock.Now().UnixMilli(),
	})
	e.mu.Unlock()

	select {
	case e.sentCh <- struct{}{}:
	default:
	}

	return nil
}

// uploadParamData is the JSON structure encoded in upload_param.
type uploadParamData struct {
	FileKey string `json:"filekey"`
	EQP     string `json:"eqp"`
	AESKey  string `json:"aeskey"`
}

// GetUploadURL stores the AES key and returns upload parameters.
func (e *Engine) GetUploadURL(req *ilink.GetUploadURLReq) (*ilink.GetUploadURLResp, error) {
	eqp := generateEQP()

	paramJSON, _ := json.Marshal(uploadParamData{
		FileKey: req.FileKey,
		EQP:     eqp,
		AESKey:  req.AESKey,
	})
	uploadParam := base64.StdEncoding.EncodeToString(paramJSON)

	return &ilink.GetUploadURLResp{
		Ret:         0,
		UploadParam: uploadParam,
	}, nil
}

// UploadToCDN stores ciphertext and returns a download EQP.
func (e *Engine) UploadToCDN(uploadParam, filekey string, ciphertext []byte) (string, error) {
	paramJSON, err := base64.StdEncoding.DecodeString(uploadParam)
	if err != nil {
		return "", fmt.Errorf("decode upload_param: %w", err)
	}

	var param uploadParamData
	if err := json.Unmarshal(paramJSON, &param); err != nil {
		return "", fmt.Errorf("unmarshal upload_param: %w", err)
	}

	eqp := param.EQP

	e.mu.Lock()
	e.media[eqp] = &mediaEntry{
		ciphertext: ciphertext,
		aesKeyHex:  param.AESKey,
		rawSize:    len(ciphertext),
		uploadedAt: e.clock.Now(),
	}
	e.mu.Unlock()

	return eqp, nil
}

// DownloadFile retrieves and decrypts a media file by EQP.
func (e *Engine) DownloadFile(eqp, aesKeyBase64 string) ([]byte, error) {
	e.mu.Lock()
	entry, ok := e.media[eqp]
	e.mu.Unlock()

	if !ok {
		return nil, fmt.Errorf("media not found: %s", eqp)
	}

	key, err := parseAESKey(aesKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("parse AES key: %w", err)
	}

	plaintext, err := decryptMedia(entry.ciphertext, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt media: %w", err)
	}

	return plaintext, nil
}

// DownloadVoice retrieves and decrypts a voice file. The sampleRate parameter
// is ignored in the mock (SILK decoding is handled client-side).
func (e *Engine) DownloadVoice(eqp, aesKeyBase64 string, sampleRate int) ([]byte, error) {
	return e.DownloadFile(eqp, aesKeyBase64)
}

// SendTyping sets the typing indicator for a recipient.
func (e *Engine) SendTyping(recipient, ticket string, typing bool) error {
	e.mu.Lock()
	e.typing[recipient] = typing
	e.mu.Unlock()
	return nil
}

// GetConfig returns bot config for a given recipient.
func (e *Engine) GetConfig(recipient, contextToken string) (*ConfigResult, error) {
	return &ConfigResult{
		TypingTicket: "mock-ticket-" + randomHex(4),
	}, nil
}

// FetchQRCode creates a new QR login session.
func (e *Engine) FetchQRCode() (*QRResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	qrCode := "mock-qr-" + randomHex(8)
	session := &qrSession{
		qrCode: qrCode,
		state:  "wait",
		ch:     make(chan struct{}, 1),
	}

	e.qr = session

	// Start expiry timer.
	qrTTL := 30 * time.Second
	if e.opts.sessionTTL > 0 {
		qrTTL = e.opts.sessionTTL
	}
	timer := e.clock.NewTimer(qrTTL)
	session.timer = timer
	go func() {
		<-timer.C
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.qr == session && (session.state == "wait" || session.state == "scanned") {
			session.state = "expired"
			select {
			case session.ch <- struct{}{}:
			default:
			}
		}
	}()

	if e.opts.autoConfirmQR {
		session.state = "confirmed"
		session.creds = Credentials{
			BotID:       "mock-bot-id",
			BotToken:    "mock-bot-token",
			BaseURL:     "https://mock.ilink/api",
			ILinkUserID: "mock-user-id",
		}
		select {
		case session.ch <- struct{}{}:
		default:
		}
	}

	return &QRResult{
		QRCode:    qrCode,
		QRContent: "https://mock.ilink/qr/" + qrCode,
	}, nil
}

// PollQRStatus polls the QR code scan/login status.
func (e *Engine) PollQRStatus(ctx context.Context, qrCode string) (*QRStatusResult, error) {
	e.mu.Lock()
	session := e.qr
	e.mu.Unlock()

	if session == nil {
		return nil, errors.New("no QR session")
	}

	e.mu.Lock()
	state := session.state
	e.mu.Unlock()

	// If already terminal, return immediately.
	if state == "confirmed" || state == "expired" {
		if state == "confirmed" {
			creds := session.creds
			return &QRStatusResult{Status: state, Creds: &creds}, nil
		}
		return &QRStatusResult{Status: state}, nil
	}

	// Block until state changes or ctx cancelled.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-session.ch:
	}

	e.mu.Lock()
	state = session.state
	creds := session.creds
	e.mu.Unlock()

	result := &QRStatusResult{Status: state}
	if state == "confirmed" {
		result.Creds = &creds
	}
	return result, nil
}
