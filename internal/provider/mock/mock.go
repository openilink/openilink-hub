package mock

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/openilink/openilink-hub/internal/provider"
)

func init() {
	provider.Register("mock", func() provider.Provider {
		return New()
	})
}

// Provider is a mock provider for testing.
type Provider struct {
	mu             sync.Mutex
	status         string
	onMsg          func(provider.InboundMessage)
	onStatus       func(string)
	sent           []provider.OutboundMessage
	mediaDownloads int
	voiceDownloads int
}

func New() *Provider {
	return &Provider{status: "disconnected"}
}

func (p *Provider) Name() string { return "mock" }
func (p *Provider) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

func (p *Provider) Start(_ context.Context, opts provider.StartOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = "connected"
	p.onMsg = opts.OnMessage
	p.onStatus = opts.OnStatus
	return nil
}

func (p *Provider) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = "disconnected"
}

func (p *Provider) Send(_ context.Context, msg provider.OutboundMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, msg)
	return "mock-client-id", nil
}

func (p *Provider) SendTyping(_ context.Context, _, _ string, _ bool) error { return nil }

func (p *Provider) DownloadMedia(_ context.Context, _, _ string) ([]byte, error) {
	p.mu.Lock()
	p.mediaDownloads++
	p.mu.Unlock()
	return []byte("mock-media-data"), nil
}

func (p *Provider) DownloadVoice(_ context.Context, _, _ string, _ int) ([]byte, error) {
	p.mu.Lock()
	p.voiceDownloads++
	p.mu.Unlock()
	// Return a minimal WAV header for testing
	return []byte("RIFF\x00\x00\x00\x00WAVEfmt mock-wav-data"), nil
}

func (p *Provider) GetConfig(_ context.Context, _, _ string) (*provider.BotConfig, error) {
	return &provider.BotConfig{TypingTicket: "mock-ticket"}, nil
}

// SimulateInbound injects a fake inbound message for testing.
// Automatically sets Raw to a JSON representation if not already set.
func (p *Provider) SimulateInbound(msg provider.InboundMessage) {
	if msg.Raw == nil {
		raw, _ := json.Marshal(map[string]any{
			"message_id":     msg.ExternalID,
			"from_user_id":   msg.Sender,
			"to_user_id":     msg.Recipient,
			"create_time_ms": msg.Timestamp,
			"item_list":      msg.Items,
			"_mock":          true,
		})
		msg.Raw = raw
	}
	p.mu.Lock()
	cb := p.onMsg
	p.mu.Unlock()
	if cb != nil {
		cb(msg)
	}
}

// SentMessages returns all outbound messages sent through this provider.
func (p *Provider) SentMessages() []provider.OutboundMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]provider.OutboundMessage, len(p.sent))
	copy(out, p.sent)
	return out
}

func (p *Provider) DownloadMediaCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mediaDownloads
}

func (p *Provider) DownloadVoiceCalls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.voiceDownloads
}

// Credentials returns mock credentials JSON.
func Credentials() json.RawMessage {
	data, _ := json.Marshal(map[string]string{"mock": "true"})
	return data
}
