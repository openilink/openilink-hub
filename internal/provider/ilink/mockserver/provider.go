package mockserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openilink/openilink-hub/internal/provider"
	ilink "github.com/openilink/openilink-sdk-go"
)

func init() {
	provider.Register("mock", func() provider.Provider {
		return NewProvider()
	})
}

// Provider wraps Engine and implements provider.Provider and provider.Binder.
type Provider struct {
	engine *Engine
	cancel context.CancelFunc
	opts   provider.StartOptions
}

// Compile-time interface checks.
var (
	_ provider.Provider = (*Provider)(nil)
	_ provider.Binder   = (*Provider)(nil)
)

// NewProvider creates a new mock Provider with the given engine options.
func NewProvider(opts ...Option) *Provider {
	return &Provider{engine: NewEngine(opts...)}
}

// Engine returns the underlying mock engine for test control.
func (p *Provider) Engine() *Engine { return p.engine }

// Name returns the provider name.
func (p *Provider) Name() string { return "mock" }

// Status returns the current connection status.
func (p *Provider) Status() string {
	p.engine.mu.Lock()
	defer p.engine.mu.Unlock()
	return p.engine.status
}

// Start connects the provider and begins polling for inbound messages.
func (p *Provider) Start(ctx context.Context, opts provider.StartOptions) error {
	p.opts = opts
	var creds Credentials
	if opts.Credentials != nil {
		json.Unmarshal(opts.Credentials, &creds)
	}
	if creds.BotToken == "" {
		creds.BotToken = "mock-token"
	}
	p.engine.SetToken(creds.BotToken)
	p.engine.SetStatus("connected")
	if opts.OnStatus != nil {
		opts.OnStatus("connected")
	}

	ctx, p.cancel = context.WithCancel(ctx)
	go p.pollLoop(ctx)
	return nil
}

func (p *Provider) pollLoop(ctx context.Context) {
	buf := ""
	for {
		result, err := p.engine.GetUpdates(ctx, buf)
		if err != nil {
			p.engine.mu.Lock()
			expired := p.engine.status == "session_expired"
			p.engine.mu.Unlock()
			if expired && p.opts.OnStatus != nil {
				p.opts.OnStatus("session_expired")
			}
			return
		}
		if result.SyncBuf != "" {
			buf = result.SyncBuf
			if p.opts.OnSyncUpdate != nil {
				data, _ := json.Marshal(map[string]string{"sync_buf": buf})
				p.opts.OnSyncUpdate(data)
			}
		}
		for _, msg := range result.Messages {
			if p.opts.OnMessage != nil {
				p.opts.OnMessage(convertWeixinToInbound(msg))
			}
		}
	}
}

// Stop disconnects the provider.
func (p *Provider) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.engine.SetStatus("disconnected")
}

// Send sends an outbound message via the engine.
func (p *Provider) Send(ctx context.Context, msg provider.OutboundMessage) (string, error) {
	if len(msg.Data) > 0 && msg.FileName != "" {
		return "", p.engine.SendMediaFile(msg.Recipient, msg.ContextToken, msg.Data, msg.FileName, msg.Text)
	}
	if msg.ContextToken != "" {
		return p.engine.SendText(msg.Recipient, msg.Text, msg.ContextToken)
	}
	return p.engine.Push(msg.Recipient, msg.Text)
}

// SendTyping sets the typing indicator.
func (p *Provider) SendTyping(ctx context.Context, recipient, ticket string, typing bool) error {
	return p.engine.SendTyping(recipient, ticket, typing)
}

// GetConfig returns bot configuration.
func (p *Provider) GetConfig(ctx context.Context, recipient, contextToken string) (*provider.BotConfig, error) {
	result, err := p.engine.GetConfig(recipient, contextToken)
	if err != nil {
		return nil, err
	}
	return &provider.BotConfig{TypingTicket: result.TypingTicket}, nil
}

// DownloadMedia retrieves and decrypts a media file.
func (p *Provider) DownloadMedia(ctx context.Context, eqp, aesKey string) ([]byte, error) {
	return p.engine.DownloadFile(eqp, aesKey)
}

// DownloadVoice retrieves and decrypts a voice file.
func (p *Provider) DownloadVoice(ctx context.Context, eqp, aesKey string, sampleRate int) ([]byte, error) {
	return p.engine.DownloadVoice(eqp, aesKey, sampleRate)
}

// StartBind implements provider.Binder.
func (p *Provider) StartBind(ctx context.Context) (*provider.BindSession, error) {
	qr, err := p.engine.FetchQRCode()
	if err != nil {
		return nil, err
	}
	return &provider.BindSession{
		SessionID: "mock-session-" + randomHex(8),
		QRURL:     qr.QRContent,
		PollStatus: func(ctx context.Context) (*provider.BindPollResult, error) {
			result, err := p.engine.PollQRStatus(ctx, qr.QRCode)
			if err != nil {
				return nil, err
			}
			pr := &provider.BindPollResult{Status: result.Status}
			if result.Status == "confirmed" && result.Creds != nil {
				data, _ := json.Marshal(result.Creds)
				pr.Credentials = data
			}
			return pr, nil
		},
	}, nil
}

// MockCredentials returns a json.RawMessage with default mock credentials.
func MockCredentials() json.RawMessage {
	data, _ := json.Marshal(Credentials{
		BotID: "mock-bot-id", BotToken: "mock-token", ILinkUserID: "mock-user",
	})
	return data
}

// convertWeixinToInbound converts an ilink.WeixinMessage to provider.InboundMessage.
func convertWeixinToInbound(msg *ilink.WeixinMessage) provider.InboundMessage {
	var items []provider.MessageItem
	for _, item := range msg.ItemList {
		mi := convertItem(item)
		if mi != nil {
			items = append(items, *mi)
		}
	}

	// Serialize the WeixinMessage as Raw so tests can verify raw storage.
	raw, _ := json.Marshal(msg)

	return provider.InboundMessage{
		ExternalID:   fmt.Sprintf("%d", msg.MessageID),
		Sender:       msg.FromUserID,
		Recipient:    msg.ToUserID,
		GroupID:      msg.GroupID,
		Timestamp:    msg.CreateTimeMs,
		MessageState: int(msg.MessageState),
		Items:        items,
		ContextToken: msg.ContextToken,
		SessionID:    msg.SessionID,
		Raw:          raw,
	}
}

func convertItem(item ilink.MessageItem) *provider.MessageItem {
	mi := &provider.MessageItem{}

	switch item.Type {
	case ilink.ItemText:
		if item.TextItem == nil {
			return nil
		}
		mi.Type = "text"
		mi.Text = item.TextItem.Text

	case ilink.ItemImage:
		mi.Type = "image"
		if item.ImageItem != nil {
			mi.Media = convertCDNMedia(item.ImageItem.Media, "image")
			if mi.Media != nil {
				if item.ImageItem.URL != "" {
					mi.Media.URL = item.ImageItem.URL
				}
				mi.Media.ThumbWidth = item.ImageItem.ThumbWidth
				mi.Media.ThumbHeight = item.ImageItem.ThumbHeight
				if item.ImageItem.ThumbMedia != nil {
					mi.Media.ThumbEQP = item.ImageItem.ThumbMedia.EncryptQueryParam
					mi.Media.ThumbAESKey = item.ImageItem.ThumbMedia.AESKey
				}
			}
		}

	case ilink.ItemVoice:
		mi.Type = "voice"
		if item.VoiceItem != nil {
			mi.Text = item.VoiceItem.Text
			mi.Media = convertCDNMedia(item.VoiceItem.Media, "voice")
			if mi.Media != nil {
				mi.Media.PlayTime = item.VoiceItem.PlayTime
			}
		}

	case ilink.ItemFile:
		mi.Type = "file"
		if item.FileItem != nil {
			mi.FileName = item.FileItem.FileName
			mi.Media = convertCDNMedia(item.FileItem.Media, "file")
		}

	case ilink.ItemVideo:
		mi.Type = "video"
		if item.VideoItem != nil {
			mi.Media = convertCDNMedia(item.VideoItem.Media, "video")
			if mi.Media != nil {
				mi.Media.FileSize = item.VideoItem.VideoSize
				mi.Media.PlayLength = item.VideoItem.PlayLength
				mi.Media.ThumbWidth = item.VideoItem.ThumbWidth
				mi.Media.ThumbHeight = item.VideoItem.ThumbHeight
				if item.VideoItem.ThumbMedia != nil {
					mi.Media.ThumbEQP = item.VideoItem.ThumbMedia.EncryptQueryParam
					mi.Media.ThumbAESKey = item.VideoItem.ThumbMedia.AESKey
				}
			}
		}

	default:
		return nil
	}

	// Convert referenced/quoted message.
	if item.RefMsg != nil && item.RefMsg.MessageItem != nil {
		refItem := convertItem(*item.RefMsg.MessageItem)
		if refItem != nil {
			mi.RefMsg = &provider.RefMsg{
				Title: item.RefMsg.Title,
				Item:  *refItem,
			}
		}
	}

	return mi
}

func convertCDNMedia(m *ilink.CDNMedia, mediaType string) *provider.Media {
	if m == nil {
		return nil
	}
	return &provider.Media{
		EncryptQueryParam: m.EncryptQueryParam,
		AESKey:            m.AESKey,
		MediaType:         mediaType,
	}
}
