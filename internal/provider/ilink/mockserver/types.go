package mockserver

import (
	"encoding/json"

	ilink "github.com/openilink/openilink-sdk-go"
)

// InboundRequest represents a simulated incoming message from a user.
type InboundRequest struct {
	Sender       string        `json:"sender"`
	Recipient    string        `json:"recipient,omitempty"`
	Text         string        `json:"text,omitempty"`
	Items        []ItemRequest `json:"items,omitempty"`
	GroupID      string        `json:"group_id,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	MessageState int           `json:"message_state,omitempty"`
}

// ItemRequest describes a single item (text, file, etc.) in an InboundRequest.
type ItemRequest struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	FileName string `json:"file_name,omitempty"`
	Data     []byte `json:"data,omitempty"`
}

// SentMessage records a message sent by the bot via the mock engine.
type SentMessage struct {
	Recipient    string          `json:"recipient"`
	Text         string          `json:"text,omitempty"`
	ContextToken string          `json:"context_token,omitempty"`
	ClientID     string          `json:"client_id"`
	Items        json.RawMessage `json:"items,omitempty"`
	MediaData    []byte          `json:"-"`
	FileName     string          `json:"file_name,omitempty"`
	Timestamp    int64           `json:"timestamp"`
}

// MediaInfo describes an uploaded media file.
type MediaInfo struct {
	EQP       string `json:"eqp"`
	AESKey    string `json:"aes_key"`
	MediaType int    `json:"media_type"`
	FileName  string `json:"file_name,omitempty"`
	Size      int    `json:"size"`
}

// Credentials holds the bot login credentials returned after QR confirmation.
type Credentials struct {
	BotID       string `json:"bot_id"`
	BotToken    string `json:"bot_token"`
	BaseURL     string `json:"base_url,omitempty"`
	ILinkUserID string `json:"ilink_user_id,omitempty"`
}

// GetUpdatesResult is returned by Engine.GetUpdates.
type GetUpdatesResult struct {
	Messages []*ilink.WeixinMessage
	SyncBuf  string
}

// QRResult is returned by Engine.FetchQRCode.
type QRResult struct {
	QRCode    string
	QRContent string
}

// QRStatusResult is returned by Engine.PollQRStatus.
type QRStatusResult struct {
	Status string
	Creds  *Credentials // set when Status == "confirmed"
}

// ConfigResult is returned by Engine.GetConfig.
type ConfigResult struct {
	TypingTicket string
}

// UploadResult is returned by Engine.GetUploadURL.
type UploadResult struct {
	UploadParam string
}
