package provider

import (
	"context"
	"encoding/json"
	"sync"
)

// Provider abstracts a messaging bot connection (iLink, future providers, etc.).
type Provider interface {
	Name() string
	Start(ctx context.Context, opts StartOptions) error
	Stop()
	Send(ctx context.Context, msg OutboundMessage) (string, error)
	SendTyping(ctx context.Context, recipient, ticket string, typing bool) error
	GetConfig(ctx context.Context, recipient, contextToken string) (*BotConfig, error)
	DownloadMedia(ctx context.Context, media *Media) ([]byte, error)
	DownloadVoice(ctx context.Context, media *Media, sampleRate int) ([]byte, error)
	Status() string
}

// BotConfig holds provider-specific configuration (e.g. typing ticket).
type BotConfig struct {
	TypingTicket string `json:"typing_ticket,omitempty"`
}

// Binder is an optional interface for providers that support QR/interactive binding.
type Binder interface {
	StartBind(ctx context.Context) (*BindSession, error)
}

type BindSession struct {
	SessionID string
	QRURL     string
	// PollStatus is called repeatedly; returns final credentials on success.
	PollStatus func(ctx context.Context) (*BindPollResult, error)
}

type BindPollResult struct {
	Status      string          // "wait", "scanned", "expired", "confirmed"
	QRURL       string          // new QR URL on refresh
	Credentials json.RawMessage // set on "confirmed"
}

type StartOptions struct {
	Credentials  json.RawMessage
	SyncState    json.RawMessage
	OnMessage    func(InboundMessage)
	OnStatus     func(status string)
	OnSyncUpdate func(state json.RawMessage)
}

type InboundMessage struct {
	ExternalID   string
	Sender       string
	Recipient    string
	GroupID      string // non-empty for group messages
	Timestamp    int64
	MessageState int // 0=new, 1=generating, 2=finish
	Items        []MessageItem
	ContextToken string
	SessionID    string
	Raw          json.RawMessage `json:"-"` // original provider message, not serialized in relay
}

type OutboundMessage struct {
	Recipient    string
	Text         string
	ContextToken string
	// Media (optional): if Data is set, sends as media file
	Data     []byte
	FileName string
}

type MessageItem struct {
	Type     string  `json:"type"` // "text", "image", "voice", "file", "video"
	Text     string  `json:"text,omitempty"`
	FileName string  `json:"file_name,omitempty"`
	Media    *Media  `json:"media,omitempty"`
	RefMsg   *RefMsg `json:"ref_msg,omitempty"`
}

// Media holds CDN media info for image/voice/file/video items.
type Media struct {
	URL               string `json:"url,omitempty"`
	StorageKey        string `json:"-"` // MinIO object key, not serialized
	EncryptQueryParam string `json:"encrypt_query_param,omitempty"`
	AESKey            string `json:"aes_key,omitempty"`
	FileSize          int64  `json:"file_size,omitempty"`
	MediaType         string `json:"media_type,omitempty"` // "image", "voice", "file", "video"
	// Voice-specific
	PlayTime int `json:"play_time,omitempty"` // seconds
	// Video/image thumbnail
	ThumbEQP    string `json:"thumb_eqp,omitempty"`
	ThumbAESKey string `json:"thumb_aes_key,omitempty"`
	ThumbURL    string `json:"thumb_url,omitempty"`
	// Video-specific
	PlayLength  int `json:"play_length,omitempty"` // seconds
	ThumbWidth  int `json:"thumb_width,omitempty"`
	ThumbHeight int `json:"thumb_height,omitempty"`
}

// RefMsg represents a quoted/referenced message.
type RefMsg struct {
	Title string      `json:"title,omitempty"`
	Item  MessageItem `json:"item"`
}

// --- Registry ---

type Factory func() Provider

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

func Register(name string, f Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = f
}

func Get(name string) (Factory, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	f, ok := registry[name]
	return f, ok
}
