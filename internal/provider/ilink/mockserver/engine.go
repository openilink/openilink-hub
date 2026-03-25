package mockserver

import (
	"sync"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
)

// mediaEntry stores an uploaded media file in the engine.
type mediaEntry struct {
	ciphertext []byte
	aesKeyHex  string
	aesKeyRaw  []byte
	mediaType  int
	fileName   string
	rawSize    int
	uploadedAt time.Time
}

// qrSession tracks QR login state.
type qrSession struct {
	qrCode string
	state  string
	creds  Credentials
	ch     chan struct{}
	timer  *ClockTimer
}

// engineOptions holds configuration for the Engine.
type engineOptions struct {
	clock         Clock
	autoConfirmQR bool
	latency       time.Duration
	messageLoss   float64
	sessionTTL    time.Duration
}

// Option configures the Engine.
type Option func(*engineOptions)

// WithClock sets a custom clock (e.g. FakeClock for tests).
func WithClock(c Clock) Option { return func(o *engineOptions) { o.clock = c } }

// WithAutoConfirmQR makes QR login auto-confirm without waiting.
func WithAutoConfirmQR() Option { return func(o *engineOptions) { o.autoConfirmQR = true } }

// WithLatency adds artificial latency to engine operations.
func WithLatency(d time.Duration) Option { return func(o *engineOptions) { o.latency = d } }

// WithMessageLoss sets a probability (0-1) that a message is silently dropped.
func WithMessageLoss(rate float64) Option { return func(o *engineOptions) { o.messageLoss = rate } }

// WithSessionTTL sets the TTL for QR sessions.
func WithSessionTTL(d time.Duration) Option { return func(o *engineOptions) { o.sessionTTL = d } }

// Engine is the in-memory mock iLink server. It simulates QR login, message
// polling, sending, media upload/download, and typing indicators.
type Engine struct {
	mu      sync.Mutex
	clock   Clock
	opts    engineOptions
	token   string
	status  string
	inbound chan *ilink.WeixinMessage
	sent    []SentMessage
	sentCh  chan struct{}
	syncBuf string
	msgSeq  int64
	media   map[string]*mediaEntry
	qr      *qrSession
	typing  map[string]bool
}

// NewEngine creates a new mock engine with the given options.
func NewEngine(opts ...Option) *Engine {
	o := engineOptions{clock: realClock{}}
	for _, opt := range opts {
		opt(&o)
	}
	return &Engine{
		clock:   o.clock,
		opts:    o,
		status:  "disconnected",
		inbound: make(chan *ilink.WeixinMessage, 100),
		sentCh:  make(chan struct{}, 1),
		media:   make(map[string]*mediaEntry),
		typing:  make(map[string]bool),
	}
}
