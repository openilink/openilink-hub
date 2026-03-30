package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/openilink/openilink-hub/internal/provider"
)

// mockProvider implements provider.Provider with in-memory state.
// Send() records outbound messages; InjectMessage() triggers OnMessage.
type mockProvider struct {
	mu        sync.Mutex
	status    string
	sent      []provider.OutboundMessage
	onMessage func(provider.InboundMessage)
}

var _ provider.Provider = (*mockProvider)(nil)

func newMockProvider() *mockProvider {
	return &mockProvider{status: "connected"}
}

func (p *mockProvider) Name() string { return "mock" }

func (p *mockProvider) Start(_ context.Context, opts provider.StartOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onMessage = opts.OnMessage
	if opts.OnStatus != nil {
		opts.OnStatus("connected")
	}
	return nil
}

func (p *mockProvider) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = "stopped"
}

func (p *mockProvider) Send(_ context.Context, msg provider.OutboundMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, msg)
	var b [6]byte
	rand.Read(b[:])
	return "mock_" + hex.EncodeToString(b[:]), nil
}

func (p *mockProvider) SendTyping(context.Context, string, string, bool) error { return nil }

func (p *mockProvider) GetConfig(context.Context, string, string) (*provider.BotConfig, error) {
	return &provider.BotConfig{}, nil
}

func (p *mockProvider) DownloadMedia(context.Context, *provider.Media) ([]byte, error) {
	return nil, nil
}

func (p *mockProvider) DownloadVoice(context.Context, *provider.Media, int) ([]byte, error) {
	return nil, nil
}

func (p *mockProvider) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.status
}

// InjectMessage simulates an inbound message arriving at this bot.
func (p *mockProvider) InjectMessage(msg provider.InboundMessage) {
	p.mu.Lock()
	cb := p.onMessage
	p.mu.Unlock()
	if cb != nil {
		cb(msg)
	}
}

// GetSent returns all outbound messages recorded by Send().
func (p *mockProvider) GetSent() []provider.OutboundMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]provider.OutboundMessage, len(p.sent))
	copy(out, p.sent)
	return out
}

// ClearSent clears the recorded outbound messages.
func (p *mockProvider) ClearSent() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = nil
}
