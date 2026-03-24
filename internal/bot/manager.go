package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/storage"
)

var mentionRe = regexp.MustCompile(`@(\S+)`)

func parseMentions(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	var handles []string
	for _, m := range matches {
		handles = append(handles, m[1])
	}
	return handles
}

const maxConcurrentDownloads = 5

// Manager manages all active bot instances.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	db        *database.DB
	hub       *relay.Hub
	sinks     []sink.Sink
	store     *storage.Storage        // optional, for media files
	baseURL   string                  // Hub origin for proxy URLs
	dlSem     chan struct{}           // semaphore for concurrent media downloads
	appDisp   *appdelivery.Dispatcher // app event delivery
}

func NewManager(db *database.DB, hub *relay.Hub, sinks []sink.Sink, store *storage.Storage, baseURL string) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		db:        db,
		hub:       hub,
		sinks:     sinks,
		store:     store,
		baseURL:   baseURL,
		dlSem:     make(chan struct{}, maxConcurrentDownloads),
		appDisp:   appdelivery.NewDispatcher(db),
	}
}

func (m *Manager) StartAll(ctx context.Context) {
	bots, err := m.db.GetAllBots()
	if err != nil {
		slog.Error("failed to load bots", "err", err)
		return
	}
	for _, b := range bots {
		if len(b.Credentials) == 0 || string(b.Credentials) == "{}" {
			continue
		}
		// Don't auto-start bots with expired sessions — need manual re-bind
		if b.Status == "session_expired" {
			slog.Info("skip expired bot", "bot", b.ID)
			continue
		}
		if err := m.StartBot(ctx, &b); err != nil {
			slog.Error("failed to start bot", "bot", b.ID, "err", err)
		}
	}
	slog.Info("started all bots", "count", len(bots))

	// Start background reminder checker
	go m.reminderLoop(ctx)
}

func (m *Manager) StartBot(ctx context.Context, bot *database.Bot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.instances[bot.ID]; ok {
		old.Stop()
	}

	factory, ok := provider.Get(bot.Provider)
	if !ok {
		slog.Error("unknown provider", "provider", bot.Provider, "bot", bot.ID)
		return nil
	}

	p := factory()
	inst := NewInstance(bot.ID, p)

	err := p.Start(ctx, provider.StartOptions{
		Credentials: bot.Credentials,
		SyncState:   bot.SyncState,
		OnMessage: func(msg provider.InboundMessage) {
			m.onInbound(inst, msg)
		},
		OnStatus: func(status string) {
			if err := m.db.UpdateBotStatus(bot.ID, status); err != nil {
				slog.Error("update bot status failed", "bot", bot.ID, "status", status, "err", err)
			}
			m.onStatusChange(inst, status)
		},
		OnSyncUpdate: func(state json.RawMessage) {
			if err := m.db.UpdateBotSyncState(bot.ID, state); err != nil {
				slog.Error("update sync state failed", "bot", bot.ID, "err", err)
			}
		},
	})
	if err != nil {
		return err
	}

	m.instances[bot.ID] = inst
	slog.Info("bot started", "bot", bot.ID, "provider", bot.Provider)
	return nil
}

func (m *Manager) StopBot(botDBID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[botDBID]; ok {
		inst.Stop()
		delete(m.instances, botDBID)
	}
}

func (m *Manager) GetInstance(botDBID string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[botDBID]
	return inst, ok
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		inst.Stop()
	}
	m.instances = make(map[string]*Instance)
}

// reminderLoop periodically checks for bots that need inactivity reminders.
func (m *Manager) reminderLoop(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkReminders()
		}
	}
}

func (m *Manager) checkReminders() {
	bots, err := m.db.GetBotsNeedingReminder()
	if err != nil {
		slog.Error("reminder check failed", "err", err)
		return
	}
	for _, bot := range bots {
		inst, ok := m.GetInstance(bot.ID)
		if !ok {
			continue
		}

		hours := bot.ReminderHours
		remaining := 24 - hours
		text := fmt.Sprintf("[系统提醒] 您的 Bot 已超过 %d 小时未收到消息，距离会话过期还有约 %d 小时。请发送一条消息以保持会话活跃。", hours, remaining)

		token := m.db.GetLatestContextToken(bot.ID)
		_, err := inst.Send(context.Background(), provider.OutboundMessage{
			Text:         text,
			ContextToken: token,
		})
		if err != nil {
			slog.Error("reminder send failed", "bot", bot.ID, "err", err)
			continue
		}

		if err := m.db.MarkBotReminded(bot.ID); err != nil {
			slog.Error("mark reminded failed", "bot", bot.ID, "err", err)
		}
		slog.Info("reminder sent", "bot", bot.ID, "hours", hours)
	}
}

// RetryMediaDownload retries downloading media for a failed message.
func (m *Manager) RetryMediaDownload(msgID int64) error {
	msg, err := m.db.GetMessage(msgID)
	if err != nil {
		return err
	}

	// Extract media params from item_list
	var items []provider.MessageItem
	if err := json.Unmarshal(msg.ItemList, &items); err != nil || len(items) == 0 {
		return fmt.Errorf("no items in message")
	}
	var mediaItem *provider.MessageItem
	for i := range items {
		if items[i].Media != nil && items[i].Media.EncryptQueryParam != "" {
			mediaItem = &items[i]
			break
		}
	}
	if mediaItem == nil {
		return fmt.Errorf("no media item found")
	}

	inst, ok := m.GetInstance(msg.BotID)
	if !ok {
		return fmt.Errorf("bot not connected")
	}

	// Mark as downloading
	m.db.Exec("UPDATE messages SET media_status = 'downloading' WHERE id = $1", msgID)

	slog.Info("media retry start", "msgID", msgID)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("media retry panic", "msgID", msgID, "err", r)
				m.db.Exec("UPDATE messages SET media_status = 'failed' WHERE id = $1", msgID)
			}
		}()

		fakeMsg := provider.InboundMessage{
			ExternalID: fmt.Sprintf("retry-%d", msgID),
			Items:      []provider.MessageItem{*mediaItem},
		}
		m.processMedia(inst, &fakeMsg)

		item := fakeMsg.Items[0]
		status := "failed"
		keys := map[string]string{}
		if item.Media.StorageKey != "" {
			keys["0"] = item.Media.StorageKey
			status = "ready"
		} else if item.Media.URL != "" {
			keys["0"] = item.Media.URL
			status = "ready"
		}
		if item.Media.RawStorageKey != "" {
			keys["0_silk"] = item.Media.RawStorageKey
			status = "ready"
		} else if item.Media.RawURL != "" {
			keys["0_raw"] = item.Media.RawURL
			status = "ready"
		}
		if item.Media.ThumbURL != "" {
			keys["0_thumb"] = item.Media.ThumbURL
		}
		keysJSON, _ := json.Marshal(keys)
		itemListJSON, _ := json.Marshal(fakeMsg.Items)
		if err := m.db.UpdateMessageMedia(msgID, status, keysJSON, itemListJSON); err != nil {
			slog.Error("media retry update failed", "msgID", msgID, "err", err)
		}
		slog.Info("media retry done", "msgID", msgID, "status", status)
	}()
	return nil
}

func (m *Manager) onStatusChange(inst *Instance, status string) {
	env := relay.NewEnvelope("bot_status", relay.BotStatusData{
		BotID:  inst.DBID,
		Status: status,
	})
	m.hub.Broadcast(inst.DBID, env)
}

// onInbound processes an inbound message in three decoupled phases:
//  1. Store — save message to DB immediately (+ start async media download)
//  2. Route — match channels by handle/filter
//  3. Deliver — fan out to matched channels' sinks
func (m *Manager) onInbound(inst *Instance, msg provider.InboundMessage) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("onInbound panic", "bot", inst.DBID, "msg", msg.ExternalID, "err", r)
		}
	}()

	slog.Info("hub inbound message",
		"bot", inst.DBID,
		"external_id", msg.ExternalID,
		"sender", msg.Sender,
		"items", len(msg.Items),
		"context_token", msg.ContextToken != "")

	parsed := m.parseMessage(msg)

	// Phase 1: Store message (independent of channels)
	msgID := m.storeMessage(inst, msg, parsed)

	// Phase 2: Route to channels
	matched := m.matchChannels(inst.DBID, msg.Sender, parsed)

	// Phase 1b: Async media download (independent of channels)
	if parsed.hasMedia && msgID > 0 {
		go m.downloadMedia(inst, msg, matched, msgID)
	}

	// Phase 3: Deliver to sinks
	m.deliverToChannels(inst, msg, parsed, matched, msgID, "message", nil)

	// Phase 4: Deliver to Apps
	go m.deliverToApps(inst, msg, parsed)
}

// parsedMessage holds extracted info from an inbound message.
type parsedMessage struct {
	msgType    string
	content    string
	hasMedia   bool
	relayItems []relay.MessageItem
}

func (m *Manager) parseMessage(msg provider.InboundMessage) parsedMessage {
	msgType := "text"
	content := ""
	hasMedia := false
	for _, item := range msg.Items {
		switch item.Type {
		case "text":
			content = item.Text
		case "image", "voice", "file", "video":
			msgType = item.Type
			if content == "" {
				if item.Text != "" {
					content = item.Text
				} else if item.FileName != "" {
					content = item.FileName
				} else {
					content = "[" + item.Type + "]"
				}
			}
		}
		if item.Media != nil && item.Media.EncryptQueryParam != "" {
			hasMedia = true
		}
	}

	items := ConvertRelayItems(msg.Items, "")

	return parsedMessage{
		msgType: msgType, content: content, hasMedia: hasMedia, relayItems: items,
	}
}

// buildDBMessage creates a database.Message from provider message, mirroring WeChat structure.
func (m *Manager) buildDBMessage(botDBID string, channelID *string, msg provider.InboundMessage, p parsedMessage) *database.Message {
	var raw *json.RawMessage
	if msg.Raw != nil {
		r := json.RawMessage(msg.Raw)
		raw = &r
	}

	// Parse external ID as message_id
	var messageID *int64
	if id, err := strconv.ParseInt(msg.ExternalID, 10, 64); err == nil {
		messageID = &id
	}

	// item_list: store provider items as JSON
	itemList, _ := json.Marshal(msg.Items)

	mediaStatus := ""
	if p.hasMedia {
		mediaStatus = "downloading"
	}

	return &database.Message{
		BotID:        botDBID,
		ChannelID:    channelID,
		Direction:    "inbound",
		MessageID:    messageID,
		FromUserID:   msg.Sender,
		ToUserID:     msg.Recipient,
		CreateTimeMs: &msg.Timestamp,
		SessionID:    msg.SessionID,
		GroupID:      msg.GroupID,
		MessageState: msg.MessageState,
		ItemList:     itemList,
		ContextToken: msg.ContextToken,
		MediaStatus:  mediaStatus,
		Raw:          raw,
	}
}

// storeMessage saves the message to DB without any channel association.
func (m *Manager) storeMessage(inst *Instance, msg provider.InboundMessage, p parsedMessage) int64 {
	if err := m.db.IncrBotMsgCount(inst.DBID); err != nil {
		slog.Error("incr msg count failed", "bot", inst.DBID, "err", err)
	}
	dbMsg := m.buildDBMessage(inst.DBID, nil, msg, p)
	seqID, _ := m.db.SaveMessage(dbMsg)
	return seqID
}

// downloadMedia downloads media files async and updates stored messages.
// Uses semaphore to limit concurrent downloads.
func (m *Manager) downloadMedia(inst *Instance, msg provider.InboundMessage, matched []database.Channel, msgID int64) {
	// Acquire download slot
	m.dlSem <- struct{}{}
	defer func() { <-m.dlSem }()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("media download panic", "err", r, "bot", inst.DBID)
			m.db.UpdateMessageMediaStatus(msgID, "failed", nil)
		}
	}()

	slog.Info("media download start", "bot", inst.DBID, "msg", msg.ExternalID)
	m.processMedia(inst, &msg)

	// Collect storage keys
	keys := map[string]string{}
	status := "failed"
	for i, item := range msg.Items {
		if item.Media == nil {
			continue
		}
		idx := fmt.Sprintf("%d", i)
		if item.Media.StorageKey != "" {
			keys[idx] = item.Media.StorageKey
			status = "ready"
		} else if item.Media.URL != "" {
			keys[idx] = item.Media.URL
			status = "ready"
		}
		if item.Media.RawStorageKey != "" {
			keys[idx+"_silk"] = item.Media.RawStorageKey
			status = "ready"
		} else if item.Media.RawURL != "" {
			keys[idx+"_raw"] = item.Media.RawURL
			status = "ready"
		}
		if item.Media.ThumbURL != "" {
			keys[idx+"_thumb"] = item.Media.ThumbURL
		}
	}

	keysJSON, _ := json.Marshal(keys)
	itemListJSON, _ := json.Marshal(msg.Items)
	if err := m.db.UpdateMessageMedia(msgID, status, keysJSON, itemListJSON); err != nil {
		slog.Error("media status update failed", "bot", inst.DBID, "err", err)
	}
	slog.Info("media download done", "bot", inst.DBID, "msg", msg.ExternalID, "status", status)
	if status == "ready" && len(matched) > 0 {
		m.deliverToChannels(inst, msg, m.parseMessage(msg), matched, msgID, "message_media_ready", func(sk sink.Sink) bool {
			switch sk.Name() {
			case "ws", "webhook":
				return true
			default:
				return false
			}
		})
	}
}

// matchChannels finds channels that should receive this message.
func (m *Manager) matchChannels(botDBID, sender string, p parsedMessage) []database.Channel {
	channels, err := m.db.ListChannelsByBot(botDBID)
	if err != nil {
		slog.Error("load channels failed", "bot", botDBID, "err", err)
		return nil
	}

	var matched []database.Channel
	mentioned := parseMentions(p.content)
	mentionMatched := make(map[string]bool)

	for _, ch := range channels {
		if ch.Handle == "" {
			if matchFilter(ch.FilterRule, sender, p.content, p.msgType) {
				matched = append(matched, ch)
			}
		} else if len(mentioned) > 0 && !mentionMatched[strings.ToLower(ch.Handle)] {
			for _, m := range mentioned {
				if strings.EqualFold(m, ch.Handle) {
					matched = append(matched, ch)
					mentionMatched[strings.ToLower(ch.Handle)] = true
					break
				}
			}
		}
	}
	return matched
}

// deliverToChannels fans out to matched channels' sinks concurrently.
// Uses the global msgID as seqID — no per-channel message copies.
func (m *Manager) deliverToChannels(inst *Instance, msg provider.InboundMessage, p parsedMessage, matched []database.Channel, msgID int64, event string, useSink func(sink.Sink) bool) {
	if len(matched) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, ch := range matched {
		if err := m.db.UpdateChannelLastSeq(ch.ID, msgID); err != nil {
			slog.Error("update channel last_seq failed", "channel", ch.ID, "err", err)
		}

		// Strip @handle prefix from content for this channel's delivery
		deliveryContent := p.content
		if ch.Handle != "" {
			deliveryContent = strings.TrimSpace(strings.TrimPrefix(
				strings.TrimSpace(deliveryContent), "@"+ch.Handle))
		}

		env := relay.NewEnvelope(event, relay.MessageData{
			SeqID: msgID, ExternalID: msg.ExternalID,
			Sender: msg.Sender, Recipient: msg.Recipient, GroupID: msg.GroupID,
			Timestamp: msg.Timestamp, MessageState: msg.MessageState,
			Items: ConvertRelayItems(msg.Items, ch.APIKey), ContextToken: msg.ContextToken, SessionID: msg.SessionID,
		})

		d := sink.Delivery{
			Event:   event,
			BotDBID: inst.DBID, Provider: inst.Provider, Channel: ch,
			Message: msg, Envelope: env, SeqID: msgID,
			MsgType: p.msgType, Content: deliveryContent,
		}
		for _, s := range m.sinks {
			if useSink != nil && !useSink(s) {
				continue
			}
			wg.Add(1)
			go func(sk sink.Sink, delivery sink.Delivery) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("sink panic", "sink", sk.Name(), "bot", delivery.BotDBID,
							"channel", delivery.Channel.ID, "err", r)
					}
				}()
				sk.Handle(delivery)
			}(s, d)
		}
	}
	wg.Wait()
}

// matchFilter checks if a message passes the channel's filter rule.
// processMedia handles media items:
// - With MinIO: download → store → set URL to MinIO
// - Without MinIO: set URL to Hub proxy endpoint
func (m *Manager) processMedia(inst *Instance, msg *provider.InboundMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for i := range msg.Items {
		item := &msg.Items[i]
		if item.Media == nil || item.Media.EncryptQueryParam == "" {
			continue
		}

		if m.store != nil {
			if item.Type == "voice" {
				rawSilk, rawErr := inst.Provider.DownloadMedia(ctx, item.Media.EncryptQueryParam, item.Media.AESKey)
				if rawErr == nil && len(rawSilk) > 0 {
					now := time.Now()
					silkKey := fmt.Sprintf("%s/%d/%02d/%02d/%s_%d_raw.silk",
						inst.DBID, now.Year(), now.Month(), now.Day(),
						msg.ExternalID, i)
					rawURL, err := m.store.Put(ctx, silkKey, "audio/silk", rawSilk)
					if err == nil {
						item.Media.RawStorageKey = silkKey
						item.Media.RawURL = rawURL
						if item.Media.FileSize == 0 {
							item.Media.FileSize = int64(len(rawSilk))
						}
					}
				} else if rawErr != nil {
					slog.Warn("raw voice download failed", "bot", inst.DBID, "type", item.Type, "err", rawErr)
				}
				data, decoded, err := m.downloadVoiceWithFallback(ctx, inst, item)
				if err != nil {
					slog.Error("voice decode failed", "bot", inst.DBID, "type", item.Type, "err", err)
				} else if decoded {
					now := time.Now()
					key := fmt.Sprintf("%s/%d/%02d/%02d/%s_%d%s",
						inst.DBID, now.Year(), now.Month(), now.Day(),
						msg.ExternalID, i, mediaExt(item.Type))
					url, err := m.store.Put(ctx, key, mediaContentType(item.Type), data)
					if err != nil {
						slog.Error("media store failed", "bot", inst.DBID, "key", key, "err", err)
					} else {
						item.Media.URL = url
						item.Media.StorageKey = key
						item.Media.FileSize = int64(len(data))
					}
				}
				continue
			}

			data, err := inst.Provider.DownloadMedia(ctx, item.Media.EncryptQueryParam, item.Media.AESKey)
			if err != nil {
				slog.Error("media download failed", "bot", inst.DBID, "type", item.Type, "err", err)
				continue
			}
			ext := mediaExt(item.Type)
			ct := mediaContentType(item.Type)
			now := time.Now()
			key := fmt.Sprintf("%s/%d/%02d/%02d/%s_%d%s",
				inst.DBID, now.Year(), now.Month(), now.Day(),
				msg.ExternalID, i, ext)
			url, err := m.store.Put(ctx, key, ct, data)
			if err != nil {
				slog.Error("media store failed", "bot", inst.DBID, "key", key, "err", err)
				continue
			}
			item.Media.URL = url
			item.Media.StorageKey = key
			item.Media.FileSize = int64(len(data))

			// Video/image: also download thumbnail if available
			if (item.Type == "video" || item.Type == "image") && item.Media.ThumbEQP != "" {
				thumbData, err := inst.Provider.DownloadMedia(ctx, item.Media.ThumbEQP, item.Media.ThumbAESKey)
				if err == nil {
					thumbKey := fmt.Sprintf("%s/%d/%02d/%02d/%s_%d_thumb.jpg",
						inst.DBID, now.Year(), now.Month(), now.Day(),
						msg.ExternalID, i)
					if thumbURL, err := m.store.Put(ctx, thumbKey, "image/jpeg", thumbData); err == nil {
						item.Media.ThumbURL = thumbURL
					}
				}
			}
		} else {
			item.Media.URL = buildChannelMediaURL(m.baseURL, item.Type, item.Media, false)
			if item.Type == "voice" {
				item.Media.RawURL = buildChannelMediaURL(m.baseURL, item.Type, item.Media, true)
			}
		}
	}
}

// downloadVoiceWithFallback tries SILK decode at common sample rates and reports whether a WAV was produced.
func (m *Manager) downloadVoiceWithFallback(ctx context.Context, inst *Instance, item *provider.MessageItem) ([]byte, bool, error) {
	eqp := item.Media.EncryptQueryParam
	aes := item.Media.AESKey

	rates := []int{24000}
	if item.Media != nil && item.Media.SampleRate > 0 && item.Media.SampleRate != 24000 {
		rates = append(rates, item.Media.SampleRate)
	}
	has16k := false
	for _, rate := range rates {
		if rate == 16000 {
			has16k = true
			break
		}
	}
	if !has16k {
		rates = append(rates, 16000)
	}

	var lastErr error
	for _, rate := range rates {
		data, err := inst.Provider.DownloadVoice(ctx, eqp, aes, rate)
		if err == nil {
			slog.Info("voice decoded", "rate", rate)
			return data, true, nil
		}
		lastErr = err
		slog.Warn("voice decode failed", "rate", rate, "err", err)
	}

	return nil, false, lastErr
}

func mediaExt(itemType string) string {
	switch itemType {
	case "image":
		return ".jpg"
	case "voice":
		return ".wav"
	case "video":
		return ".mp4"
	default:
		return ""
	}
}

func mediaContentType(itemType string) string {
	switch itemType {
	case "image":
		return "image/jpeg"
	case "voice":
		return "audio/wav"
	case "video":
		return "video/mp4"
	case "file":
		return "application/octet-stream"
	default:
		return "application/octet-stream"
	}
}

func matchFilter(rule database.FilterRule, sender, text, msgType string) bool {
	if len(rule.UserIDs) > 0 {
		found := false
		for _, uid := range rule.UserIDs {
			if uid == sender {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.MessageTypes) > 0 {
		found := false
		for _, mt := range rule.MessageTypes {
			if mt == msgType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.Keywords) > 0 {
		found := false
		lower := strings.ToLower(text)
		for _, kw := range rule.Keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func ConvertRelayItems(items []provider.MessageItem, apiKey string) []relay.MessageItem {
	out := make([]relay.MessageItem, len(items))
	for i, item := range items {
		out[i] = ConvertRelayItem(item, apiKey)
	}
	return out
}

func ConvertRelayItem(item provider.MessageItem, apiKey string) relay.MessageItem {
	ri := relay.MessageItem{
		Type:     item.Type,
		Text:     item.Text,
		FileName: item.FileName,
	}
	if item.Media != nil {
		ri.Media = &relay.Media{
			URL:           withChannelMediaURL(item.Media.URL, apiKey),
			RawURL:        withChannelMediaURL(item.Media.RawURL, apiKey),
			AESKey:        item.Media.AESKey,
			FileSize:      item.Media.FileSize,
			MediaType:     item.Media.MediaType,
			PlayTime:      item.Media.PlayTime,
			SampleRate:    item.Media.SampleRate,
			BitsPerSample: item.Media.BitsPerSample,
			EncodeType:    item.Media.EncodeType,
			PlayLength:    item.Media.PlayLength,
			ThumbWidth:    item.Media.ThumbWidth,
			ThumbHeight:   item.Media.ThumbHeight,
		}
	}
	if item.RefMsg != nil {
		refItem := ConvertRelayItem(item.RefMsg.Item, apiKey)
		ri.RefMsg = &relay.RefMsg{
			Title: item.RefMsg.Title,
			Item:  refItem,
		}
	}
	return ri
}

func buildChannelMediaURL(baseURL, itemType string, media *provider.Media, raw bool) string {
	values := url.Values{}
	values.Set("eqp", media.EncryptQueryParam)
	values.Set("aes", media.AESKey)
	values.Set("mt", itemType)
	if raw {
		values.Set("raw", "1")
		values.Set("ct", "audio/silk")
	} else {
		values.Set("ct", mediaContentType(itemType))
	}
	if itemType == "voice" && media.SampleRate > 0 {
		values.Set("sr", strconv.Itoa(media.SampleRate))
	}
	return fmt.Sprintf("%s/api/v1/channels/media?%s", baseURL, values.Encode())
}

func withChannelMediaURL(rawURL, apiKey string) string {
	if rawURL == "" || apiKey == "" {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if !strings.Contains(u.Path, "/api/v1/media/") && !strings.HasSuffix(u.Path, "/api/v1/channels/media") {
		return rawURL
	}
	q := u.Query()
	if q.Get("key") == "" {
		q.Set("key", apiKey)
	}
	u.RawQuery = q.Encode()
	return u.String()
}
