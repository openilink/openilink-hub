package ilink

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"bytes"
	"encoding/base64"
	"strings"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/youthlin/silk"
)

func init() {
	provider.Register("ilink", func() provider.Provider {
		return &Provider{}
	})
}

// Credentials stored as JSONB in bots.credentials.
type Credentials struct {
	BotID       string `json:"bot_id"`
	BotToken    string `json:"bot_token"`
	BaseURL     string `json:"base_url,omitempty"`
	ILinkUserID string `json:"ilink_user_id,omitempty"`
}

type syncState struct {
	SyncBuf string `json:"sync_buf"`
}

type Provider struct {
	client *ilink.Client
	creds  Credentials
	cancel context.CancelFunc
	status atomic.Value
	mu     sync.Mutex
}

func (p *Provider) Name() string { return "ilink" }

func (p *Provider) Status() string {
	v := p.status.Load()
	if v == nil {
		return "disconnected"
	}
	return v.(string)
}

func (p *Provider) Start(ctx context.Context, opts provider.StartOptions) error {
	var creds Credentials
	if err := json.Unmarshal(opts.Credentials, &creds); err != nil {
		return err
	}
	p.creds = creds

	clientOpts := []ilink.Option{
		ilink.WithSILKDecoder(decodeSILK),
	}
	if creds.BaseURL != "" {
		clientOpts = append(clientOpts, ilink.WithBaseURL(creds.BaseURL))
	}
	p.client = ilink.NewClient(creds.BotToken, clientOpts...)

	var ss syncState
	if opts.SyncState != nil {
		json.Unmarshal(opts.SyncState, &ss)
	}

	ctx, p.cancel = context.WithCancel(ctx)
	p.status.Store("connected")
	if opts.OnStatus != nil {
		opts.OnStatus("connected")
	}

	go func() {
		// Cache last raw response body for attaching to inbound messages
		var lastRawBody []byte

		err := p.client.Monitor(ctx, func(msg ilink.WeixinMessage) {
			if opts.OnMessage != nil {
				im := convertInbound(msg)
				// Attach raw HTTP response body (contains all messages from this poll)
				if lastRawBody != nil {
					im.Raw = json.RawMessage(lastRawBody)
				}
				opts.OnMessage(im)
			}
		}, &ilink.MonitorOptions{
			InitialBuf: ss.SyncBuf,
			OnBufUpdate: func(buf string) {
				if opts.OnSyncUpdate != nil {
					data, _ := json.Marshal(syncState{SyncBuf: buf})
					opts.OnSyncUpdate(data)
				}
			},
			OnError: func(err error) {
				slog.Warn("ilink monitor error", "err", err)
			},
			OnSessionExpired: func() {
				slog.Error("ilink session expired")
				p.status.Store("session_expired")
				if opts.OnStatus != nil {
					opts.OnStatus("session_expired")
				}
			},
			OnResponse: func(resp *ilink.GetUpdatesResp) {
				if raw := resp.RawResponse(); raw != nil {
					lastRawBody = raw.Body
				}
			},
		})

		// Don't overwrite session_expired — that's a terminal state
		if p.Status() != "session_expired" {
			var newStatus string
			if err != nil && err != context.Canceled {
				slog.Error("ilink monitor stopped", "err", err)
				newStatus = "error"
			} else {
				newStatus = "disconnected"
			}
			p.status.Store(newStatus)
			if opts.OnStatus != nil {
				opts.OnStatus(newStatus)
			}
		}
	}()

	return nil
}

func (p *Provider) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
}

func (p *Provider) Send(ctx context.Context, msg provider.OutboundMessage) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	recipient := msg.Recipient
	if recipient == "" {
		recipient = p.creds.ILinkUserID
	}

	slog.Info("send", "recipient", recipient, "has_data", len(msg.Data) > 0,
		"file", msg.FileName, "text_len", len(msg.Text), "context_token", msg.ContextToken)

	// Media send
	if len(msg.Data) > 0 && msg.FileName != "" {
		// Voice: encode WAV/PCM to SILK before sending
		if isVoiceFile(msg.FileName) {
			clientID, err := p.sendVoice(ctx, recipient, msg.ContextToken, msg.Data)
			if err != nil {
				slog.Error("send voice failed", "err", err)
			}
			return clientID, err
		}
		err := p.client.SendMediaFile(ctx, recipient, msg.ContextToken, msg.Data, msg.FileName, msg.Text)
		if err != nil {
			slog.Error("send media failed", "err", err)
			return "", err
		}
		return "", nil
	}

	// Text send
	if msg.ContextToken != "" {
		clientID, err := p.client.SendText(ctx, recipient, msg.Text, msg.ContextToken)
		if err != nil {
			slog.Error("send text failed", "recipient", recipient, "err", err)
		}
		return clientID, err
	}
	clientID, err := p.client.Push(ctx, recipient, msg.Text)
	if err != nil {
		slog.Error("push text failed", "recipient", recipient, "err", err)
	}
	return clientID, err
}

func (p *Provider) SendTyping(ctx context.Context, recipient, ticket string, typing bool) error {
	status := ilink.Typing
	if !typing {
		status = ilink.CancelTyping
	}
	if recipient == "" {
		recipient = p.creds.ILinkUserID
	}
	return p.client.SendTyping(ctx, recipient, ticket, status)
}

func (p *Provider) GetConfig(ctx context.Context, recipient, contextToken string) (*provider.BotConfig, error) {
	if recipient == "" {
		recipient = p.creds.ILinkUserID
	}
	resp, err := p.client.GetConfig(ctx, recipient, contextToken)
	if err != nil {
		return nil, err
	}
	return &provider.BotConfig{
		TypingTicket: resp.TypingTicket,
	}, nil
}

func (p *Provider) DownloadMedia(ctx context.Context, media *provider.Media) ([]byte, error) {
	if media == nil {
		return nil, fmt.Errorf("media is nil")
	}
	return p.client.DownloadMedia(ctx, toCDNMedia(media))
}

func (p *Provider) DownloadVoice(ctx context.Context, media *provider.Media, sampleRate int) ([]byte, error) {
	if media == nil {
		return nil, fmt.Errorf("media is nil")
	}
	return p.client.DownloadVoice(ctx, &ilink.VoiceItem{
		Media:      toCDNMedia(media),
		SampleRate: sampleRate,
	})
}

func toCDNMedia(m *provider.Media) *ilink.CDNMedia {
	if m == nil {
		return nil
	}
	return &ilink.CDNMedia{
		EncryptQueryParam: m.EncryptQueryParam,
		AESKey:            m.AESKey,
		FullURL:           m.URL,
	}
}

func decodeSILK(data []byte, sampleRate int) ([]byte, error) {
	return silk.Decode(bytes.NewReader(data), silk.WithSampleRate(sampleRate))
}

func isVoiceFile(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".wav") ||
		strings.HasSuffix(lower, ".mp3") ||
		strings.HasSuffix(lower, ".ogg") ||
		strings.HasSuffix(lower, ".silk") ||
		strings.HasSuffix(lower, ".pcm")
}

// wavInfo holds parsed WAV header information.
type wavInfo struct {
	SampleRate    int
	Channels      int
	BitsPerSample int
	PCMData       []byte
}

// parseWAV parses a WAV file and extracts PCM data and format info.
func parseWAV(data []byte) (*wavInfo, error) {
	if len(data) < 44 || string(data[:4]) != "RIFF" {
		return &wavInfo{SampleRate: 24000, Channels: 1, BitsPerSample: 16, PCMData: data}, nil
	}

	info := &wavInfo{}
	// fmt chunk info (standard positions in WAV header)
	info.Channels = int(data[22]) | int(data[23])<<8
	info.SampleRate = int(data[24]) | int(data[25])<<8 | int(data[26])<<16 | int(data[27])<<24
	info.BitsPerSample = int(data[34]) | int(data[35])<<8

	if info.SampleRate <= 0 {
		info.SampleRate = 24000
	}
	if info.Channels <= 0 {
		info.Channels = 1
	}
	if info.BitsPerSample <= 0 {
		info.BitsPerSample = 16
	}

	// Find "data" chunk
	for i := 12; i+8 < len(data); {
		chunkID := string(data[i : i+4])
		chunkSize := int(data[i+4]) | int(data[i+5])<<8 | int(data[i+6])<<16 | int(data[i+7])<<24
		if chunkID == "data" {
			start := i + 8
			end := start + chunkSize
			if end > len(data) {
				end = len(data)
			}
			info.PCMData = data[start:end]
			return info, nil
		}
		i += 8 + chunkSize
		if chunkSize%2 != 0 {
			i++
		}
	}
	info.PCMData = data[44:]
	return info, nil
}

// stereoToMono converts 16-bit stereo PCM to mono by averaging L+R.
func stereoToMono(pcm []byte) []byte {
	mono := make([]byte, len(pcm)/2)
	for i := 0; i+3 < len(pcm); i += 4 {
		l := int16(pcm[i]) | int16(pcm[i+1])<<8
		r := int16(pcm[i+2]) | int16(pcm[i+3])<<8
		m := int16((int32(l) + int32(r)) / 2)
		j := i / 2
		mono[j] = byte(m)
		mono[j+1] = byte(m >> 8)
	}
	return mono
}

// sendVoice encodes audio to SILK, uploads to CDN, and sends as voice message.
func (p *Provider) sendVoice(ctx context.Context, recipient, contextToken string, data []byte) (string, error) {
	wav, err := parseWAV(data)
	if err != nil {
		return "", fmt.Errorf("parse wav: %w", err)
	}

	pcm := wav.PCMData
	slog.Info("voice encode", "pcm_bytes", len(pcm), "sample_rate", wav.SampleRate,
		"channels", wav.Channels, "bits", wav.BitsPerSample)

	// Convert stereo to mono (SILK requires mono)
	if wav.Channels == 2 && wav.BitsPerSample == 16 {
		pcm = stereoToMono(pcm)
		slog.Info("voice stereo→mono", "mono_bytes", len(pcm))
	}

	// Encode PCM → SILK
	silkData, err := silk.Encode(bytes.NewReader(pcm), silk.SampleRate(wav.SampleRate), silk.Stx(true))
	if err != nil {
		return "", fmt.Errorf("silk encode (rate=%d, pcm=%d bytes): %w", wav.SampleRate, len(pcm), err)
	}
	slog.Info("voice silk encoded", "silk_bytes", len(silkData), "header", fmt.Sprintf("%x", silkData[:min(10, len(silkData))]))
	if err != nil {
		return "", fmt.Errorf("silk encode: %w", err)
	}

	// Upload as voice
	uploaded, err := p.client.UploadFile(ctx, silkData, recipient, ilink.MediaVoice)
	if err != nil {
		return "", fmt.Errorf("upload voice: %w", err)
	}

	// Calculate play time in milliseconds from PCM data
	// PCM is 16-bit (2 bytes per sample), mono
	playTime := len(pcm) * 1000 / (2 * wav.SampleRate)
	if playTime <= 0 {
		playTime = 1000
	}

	// Build and send voice message
	clientID := fmt.Sprintf("voice-%d", time.Now().UnixMilli())
	msg := &ilink.SendMessageReq{
		Msg: &ilink.WeixinMessage{
			ToUserID:     recipient,
			ClientID:     clientID,
			MessageType:  ilink.MsgTypeBot,
			MessageState: ilink.StateFinish,
			ContextToken: contextToken,
			ItemList: []ilink.MessageItem{{
				Type: ilink.ItemVoice,
				VoiceItem: &ilink.VoiceItem{
					Media: &ilink.CDNMedia{
						EncryptQueryParam: uploaded.DownloadEncryptedQueryParam,
						AESKey:            base64.StdEncoding.EncodeToString([]byte(uploaded.AESKey)),
						EncryptType:       ilink.EncryptAES128ECB,
					},
					EncodeType:    ilink.VoiceFormatSILK,
					SampleRate:    wav.SampleRate,
					BitsPerSample: 16,
					PlayTime:      playTime,
				},
			}},
		},
	}
	if err := p.client.SendMessage(ctx, msg); err != nil {
		return "", err
	}
	return clientID, nil
}

func convertInbound(msg ilink.WeixinMessage) provider.InboundMessage {
	var items []provider.MessageItem
	for _, item := range msg.ItemList {
		mi := convertItem(item)
		if mi != nil {
			items = append(items, *mi)
		}
	}

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
		// Raw is set by Monitor callback from OnResponse
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
					mi.Media.ThumbURL = item.ImageItem.ThumbMedia.FullURL
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
					mi.Media.ThumbURL = item.VideoItem.ThumbMedia.FullURL
				}
			}
		}

	default:
		return nil
	}

	// Convert referenced/quoted message
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
		URL:               m.FullURL,
		MediaType:         mediaType,
	}
}
