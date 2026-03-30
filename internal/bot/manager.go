package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/push"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/storage"
	"github.com/openilink/openilink-hub/internal/store"
)

const maxConcurrentDownloads = 5

// Manager manages all active bot instances.
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	store     store.Store
	hub       *relay.Hub
	aiSink    *sink.AI            // AI sink (bot-level)
	storage   storage.Store       // optional, for media files
	baseURL   string              // Hub origin for proxy URLs
	dlSem     chan struct{}        // semaphore for concurrent media downloads
	appDisp   *appdelivery.Dispatcher // app event delivery
	appWSHub  *appdelivery.WSHub      // app WebSocket connections
	pushHub   *push.Hub               // browser push WebSocket
}

func NewManager(s store.Store, hub *relay.Hub, aiSink *sink.AI, st storage.Store, baseURL string) *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
		store:     s,
		hub:       hub,
		aiSink:    aiSink,
		storage:   st,
		baseURL:   baseURL,
		dlSem:     make(chan struct{}, maxConcurrentDownloads),
		appDisp:   appdelivery.NewDispatcher(s),
	}
}

// SetPushHub sets the browser push WebSocket hub.
func (m *Manager) SetPushHub(hub *push.Hub) {
	m.pushHub = hub
}

// SetAppWSHub sets the WebSocket hub for app installations.
// Called after both Manager and WSHub are created in main.go.
func (m *Manager) SetAppWSHub(hub *appdelivery.WSHub) {
	m.appWSHub = hub
}

func (m *Manager) StartAll(ctx context.Context) {
	bots, err := m.store.GetAllBots()
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

func (m *Manager) StartBot(ctx context.Context, bot *store.Bot) error {
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
	inst.UserID = bot.UserID
	inst.AIEnabled = bot.AIEnabled
	inst.AIModel = bot.AIModel

	err := p.Start(ctx, provider.StartOptions{
		Credentials: bot.Credentials,
		SyncState:   bot.SyncState,
		OnMessage: func(msg provider.InboundMessage) {
			m.onInbound(inst, msg)
		},
		OnStatus: func(status string) {
			if err := m.store.UpdateBotStatus(bot.ID, status); err != nil {
				slog.Error("update bot status failed", "bot", bot.ID, "status", status, "err", err)
			}
			m.onStatusChange(inst, status)
		},
		OnSyncUpdate: func(state json.RawMessage) {
			if err := m.store.UpdateBotSyncState(bot.ID, state); err != nil {
				slog.Error("update sync state failed", "bot", bot.ID, "err", err)
			}
		},
	})
	if err != nil {
		return err
	}

	m.instances[bot.ID] = inst
	slog.Info("bot started", "bot", bot.ID, "provider", bot.Provider)

	// Recover any messages that were stored but not fully processed (e.g. crash).
	go m.recoverUnprocessed(inst)

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

// SetBotAIModel updates the in-memory AIModel for a running bot instance.
// It satisfies the sink.BotModelSyncer interface without creating an import cycle.
func (m *Manager) SetBotAIModel(botDBID, model string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if inst, ok := m.instances[botDBID]; ok {
		inst.AIModel = model
	}
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
	bots, err := m.store.GetBotsNeedingReminder()
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
		text := fmt.Sprintf("[系统提醒] 您的 Bot 已超过 %d 小时未收到消息，距离会话过期还有约 %d 小时。请回复 OK 以保持会话活跃。", hours, remaining)

		token := m.store.GetLatestContextToken(bot.ID)
		_, err := inst.Send(context.Background(), provider.OutboundMessage{
			Text:         text,
			ContextToken: token,
		})
		if err != nil {
			slog.Error("reminder send failed", "bot", bot.ID, "err", err)
			continue
		}

		if err := m.store.MarkBotReminded(bot.ID); err != nil {
			slog.Error("mark reminded failed", "bot", bot.ID, "err", err)
		}
		slog.Info("reminder sent", "bot", bot.ID, "hours", hours)
	}
}

// RetryMediaDownload retries downloading media for a failed message.
func (m *Manager) RetryMediaDownload(msgID int64) error {
	msg, err := m.store.GetMessage(msgID)
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
	m.store.UpdateMediaStatusByID(msgID, "downloading", nil)

	slog.Info("media retry start", "msgID", msgID)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("media retry panic", "msgID", msgID, "err", r)
				m.store.UpdateMediaStatusByID(msgID, "failed", nil)
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
		keysJSON, _ := json.Marshal(keys)
		m.store.UpdateMediaStatusByID(msgID, status, keysJSON)
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
//  1. Store — save/upsert message to DB immediately (+ start async media download)
//  2. Route — match channels by handle/filter
//  3. Deliver — fan out to matched channels' sinks
//
// Duplicate messages (same message_id seen again) only upsert the DB row and
// push the update via WebSocket — they do NOT re-trigger sink/app delivery.
func (m *Manager) onInbound(inst *Instance, msg provider.InboundMessage) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("onInbound panic", "bot", inst.DBID, "msg", msg.ExternalID, "err", r)
		}
	}()

	parsed := m.parseMessage(msg)

	// Phase 1: Store/upsert message (independent of channels)
	result := m.storeMessage(inst, msg, parsed)

	if !result.Inserted {
		// Duplicate message_id: DB row was updated (not inserted). Broadcast
		// the state change to WebSocket clients but skip full delivery.
		m.broadcastStateUpdate(inst, msg, parsed, result.ID)
		return
	}
	msgID := result.ID

	// Create OTel-style tracer for this message
	tracer := store.NewTracer(m.store, inst.DBID)
	rootSpan := tracer.Start("process_message", store.SpanKindInternal, map[string]any{
		"message.sender":  msg.Sender,
		"message.content": parsed.content,
		"message.type":    parsed.msgType,
		"message.id":      msg.ExternalID,
	})

	// Ensure trace is flushed and browser clients notified even on panic.
	defer func() {
		rootSpan.End()
		tracer.Flush()
		m.notifyPush(inst, tracer.TraceID())
	}()

	storeSpan := tracer.StartChild(rootSpan, "store", store.SpanKindInternal, map[string]any{
		"message.db_id": msgID,
	})
	storeSpan.End()

	// Phase 1b: Async media download (independent of channels)
	if parsed.hasMedia && msgID > 0 {
		go m.downloadMedia(inst, msg, msgID)
		rootSpan.AddEvent("media_download_started", nil)
	}

	// Show typing indicator while delivering.
	typingDone := m.startTyping(inst, msg)

	// Phase 3: AI + Apps concurrently
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.deliverToAI(inst, msg, parsed, msgID, tracer, rootSpan)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.deliverToApps(inst, msg, parsed, tracer, rootSpan)
	}()
	wg.Wait()

	// Stop typing indicator
	typingDone()

	// Phase 5: Mark as fully processed
	if err := m.store.MarkProcessed(msgID); err != nil {
		slog.Error("mark processed failed", "bot", inst.DBID, "msg", msgID, "err", err)
	}
}

// notifyPush sends trace_completed and message_new events to browser clients.
func (m *Manager) notifyPush(inst *Instance, traceID string) {
	if m.pushHub == nil || inst.UserID == "" {
		return
	}
	m.pushHub.Notify(inst.UserID, inst.DBID, push.NewEnvelope(push.EventTraceCompleted, push.BotEvent{
		BotID:   inst.DBID,
		TraceID: traceID,
	}))
	m.pushHub.Notify(inst.UserID, inst.DBID, push.NewEnvelope(push.EventMessageNew, push.BotEvent{
		BotID: inst.DBID,
	}))
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

	items := make([]relay.MessageItem, len(msg.Items))
	for i, item := range msg.Items {
		items[i] = convertRelayItem(item)
	}

	return parsedMessage{
		msgType: msgType, content: content, hasMedia: hasMedia, relayItems: items,
	}
}

// buildDBMessage creates a store.Message from provider message, mirroring WeChat structure.
func (m *Manager) buildDBMessage(botDBID string, channelID *string, msg provider.InboundMessage, p parsedMessage) *store.Message {
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

	return &store.Message{
		BotID:        botDBID,
		ChannelID:    channelID,
		Direction:    "inbound",
		MessageID:    messageID,
		FromUserID:   msg.Sender,
		ToUserID:     msg.Recipient,
		CreateTimeMs: &msg.Timestamp,
		SessionID:    msg.SessionID,
		GroupID:       msg.GroupID,
		MessageState: msg.MessageState,
		ItemList:     itemList,
		ContextToken: msg.ContextToken,
		MediaStatus:  mediaStatus,
		Raw:          raw,
	}
}

// storeMessage saves/upserts the message to DB without any channel association.
// Returns the SaveResult indicating whether it was a new insert or a duplicate update.
func (m *Manager) storeMessage(inst *Instance, msg provider.InboundMessage, p parsedMessage) store.SaveResult {
	dbMsg := m.buildDBMessage(inst.DBID, nil, msg, p)
	result, _ := m.store.SaveMessage(dbMsg)
	if result.Inserted {
		if err := m.store.IncrBotMsgCount(inst.DBID); err != nil {
			slog.Error("incr msg count failed", "bot", inst.DBID, "err", err)
		}
	}
	return result
}

// broadcastStateUpdate pushes a message state update to all connected WebSocket
// clients for the bot (via relay hub broadcast).
func (m *Manager) broadcastStateUpdate(inst *Instance, msg provider.InboundMessage, p parsedMessage, msgID int64) {
	env := relay.NewEnvelope("message", relay.MessageData{
		SeqID: msgID, ExternalID: msg.ExternalID,
		Sender: msg.Sender, Recipient: msg.Recipient, GroupID: msg.GroupID,
		Timestamp: msg.Timestamp, MessageState: msg.MessageState,
		Items: p.relayItems, ContextToken: msg.ContextToken, SessionID: msg.SessionID,
	})
	m.hub.Broadcast(inst.DBID, env)
}

// startTyping turns on the typing indicator for the message sender and returns
// a function that cancels it. The indicator auto-expires after 25s as a safety net.
// If the typing ticket cannot be obtained (e.g. no context token), the returned
// function is a no-op.
func (m *Manager) startTyping(inst *Instance, msg provider.InboundMessage) func() {
	if msg.ContextToken == "" {
		return func() {}
	}

	ctx := context.Background()
	bcfg, err := inst.Provider.GetConfig(ctx, msg.Sender, msg.ContextToken)
	if err != nil || bcfg == nil || bcfg.TypingTicket == "" {
		return func() {}
	}

	ticket := bcfg.TypingTicket
	inst.Provider.SendTyping(ctx, msg.Sender, ticket, true)

	// Safety: auto-cancel after 25s in case delivery hangs.
	timer := time.AfterFunc(25*time.Second, func() {
		inst.Provider.SendTyping(context.Background(), msg.Sender, ticket, false)
	})

	return func() {
		timer.Stop()
		inst.Provider.SendTyping(context.Background(), msg.Sender, ticket, false)
	}
}

// recoverUnprocessed re-delivers inbound messages that were stored but never
// marked as processed (e.g. hub crashed mid-delivery). Called once per bot on startup.
func (m *Manager) recoverUnprocessed(inst *Instance) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("recoverUnprocessed panic", "bot", inst.DBID, "err", r)
		}
	}()

	msgs, err := m.store.GetUnprocessedMessages(inst.DBID, 100)
	if err != nil {
		slog.Error("load unprocessed messages failed", "bot", inst.DBID, "err", err)
		return
	}
	if len(msgs) == 0 {
		return
	}
	slog.Info("recovering unprocessed messages", "bot", inst.DBID, "count", len(msgs))

	for i := range msgs {
		msg := rebuildInbound(&msgs[i])
		parsed := m.parseMessage(msg)

		// Retry media download if it was still pending
		if msgs[i].MediaStatus == "downloading" && parsed.hasMedia {
			go m.downloadMedia(inst, msg, msgs[i].ID)
		}

		tracer := store.NewTracer(m.store, inst.DBID)
		rootSpan := tracer.Start("recover_message", store.SpanKindInternal, map[string]any{
			"message.db_id": msgs[i].ID,
			"message.id":    msg.ExternalID,
		})
		var rwg sync.WaitGroup
		rwg.Add(1)
		go func() { defer rwg.Done(); m.deliverToAI(inst, msg, parsed, msgs[i].ID, tracer, rootSpan) }()
		rwg.Add(1)
		go func() { defer rwg.Done(); m.deliverToApps(inst, msg, parsed, tracer, rootSpan) }()
		rwg.Wait()
		rootSpan.End()
		tracer.Flush()
		m.notifyPush(inst, tracer.TraceID())

		if err := m.store.MarkProcessed(msgs[i].ID); err != nil {
			slog.Error("mark recovered msg processed failed", "id", msgs[i].ID, "err", err)
		}
	}
	slog.Info("recovery complete", "bot", inst.DBID, "count", len(msgs))
}

// rebuildInbound reconstructs a provider.InboundMessage from a stored store.Message.
func rebuildInbound(dbm *store.Message) provider.InboundMessage {
	externalID := ""
	if dbm.MessageID != nil {
		externalID = fmt.Sprintf("%d", *dbm.MessageID)
	}
	var ts int64
	if dbm.CreateTimeMs != nil {
		ts = *dbm.CreateTimeMs
	}

	// Reconstruct provider items from stored item_list JSON
	var items []provider.MessageItem
	if len(dbm.ItemList) > 0 {
		json.Unmarshal(dbm.ItemList, &items)
	}

	return provider.InboundMessage{
		ExternalID:   externalID,
		Sender:       dbm.FromUserID,
		Recipient:    dbm.ToUserID,
		GroupID:      dbm.GroupID,
		Timestamp:    ts,
		MessageState: dbm.MessageState,
		Items:        items,
		ContextToken: dbm.ContextToken,
		SessionID:    dbm.SessionID,
	}
}

// downloadMedia downloads media files async and updates stored messages.
// Uses semaphore to limit concurrent downloads.
func (m *Manager) downloadMedia(inst *Instance, msg provider.InboundMessage, msgID int64) {
	// Acquire download slot
	m.dlSem <- struct{}{}
	defer func() { <-m.dlSem }()

	defer func() {
		if r := recover(); r != nil {
			slog.Error("media download panic", "err", r, "bot", inst.DBID)
			m.store.UpdateMediaStatus(inst.DBID, "failed", nil)
		}
	}()

	slog.Info("media download start", "bot", inst.DBID, "msg", msg.ExternalID)
	silkKeys := m.processMedia(inst, &msg)

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
		if item.Media.ThumbURL != "" {
			keys[idx+"_thumb"] = item.Media.ThumbURL
		}
		if sk, ok := silkKeys[i]; ok {
			keys[idx+"_silk"] = sk
		}
	}

	keysJSON, _ := json.Marshal(keys)
	if err := m.store.UpdateMediaStatus(inst.DBID, status, keysJSON); err != nil {
		slog.Error("media status update failed", "bot", inst.DBID, "err", err)
	}
	slog.Info("media download done", "bot", inst.DBID, "msg", msg.ExternalID, "status", status)
}


// deliverToAI runs the AI sink at bot level, independent of channel matching.
func (m *Manager) deliverToAI(inst *Instance, msg provider.InboundMessage, p parsedMessage, msgID int64, tracer *store.Tracer, rootSpan *store.SpanBuilder) {
	if m.aiSink == nil || !inst.AIEnabled {
		return
	}
	d := sink.Delivery{
		BotDBID:   inst.DBID,
		Provider:  inst.Provider,
		Message:   msg,
		SeqID:     msgID,
		MsgType:   p.msgType,
		Content:   p.content,
		AIEnabled: true,
		AIModel:   inst.AIModel,
		Tracer:    tracer,
		RootSpan:  rootSpan,
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error("ai sink panic", "bot", inst.DBID, "err", r)
		}
	}()
	m.aiSink.Handle(d)
}

// processMedia handles media items:
//   - With storage (S3/FS): download → store → set URL to storage
//   - Without storage: set URL to Hub CDN proxy endpoint
func (m *Manager) processMedia(inst *Instance, msg *provider.InboundMessage) map[int]string {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	silkKeys := map[int]string{} // index → raw SILK storage key
	for i := range msg.Items {
		item := &msg.Items[i]
		if item.Media == nil || item.Media.EncryptQueryParam == "" {
			continue
		}

		if m.storage != nil {
			var data []byte
			var err error

			if item.Type == "voice" {
				// Store raw SILK before decoding to WAV
				rawSilk, rawErr := inst.Provider.DownloadMedia(ctx, item.Media.EncryptQueryParam, item.Media.AESKey)
				if rawErr == nil && len(rawSilk) > 0 {
					now := time.Now()
					silkKey := fmt.Sprintf("%s/%d/%02d/%02d/%s_%d_raw.silk",
						inst.DBID, now.Year(), now.Month(), now.Day(),
						msg.ExternalID, i)
					if _, err := m.storage.Put(ctx, silkKey, "audio/silk", rawSilk); err == nil {
						silkKeys[i] = silkKey
					}
				}
				data, err = m.downloadVoiceWithFallback(ctx, inst, item)
			} else {
				data, err = inst.Provider.DownloadMedia(ctx, item.Media.EncryptQueryParam, item.Media.AESKey)
			}
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
			url, err := m.storage.Put(ctx, key, ct, data)
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
					if thumbURL, err := m.storage.Put(ctx, thumbKey, "image/jpeg", thumbData); err == nil {
						item.Media.ThumbURL = thumbURL
					}
				}
			}
		} else {
			// Fallback: proxy URL via Hub (no storage configured)
			item.Media.URL = fmt.Sprintf("%s/api/v1/channels/media?bot=%s&eqp=%s&aes=%s&ct=%s",
				m.baseURL, inst.DBID, item.Media.EncryptQueryParam, item.Media.AESKey,
				mediaContentType(item.Type))
		}
	}
	return silkKeys
}

// downloadVoiceWithFallback tries SILK decode at 24kHz, then with item's SampleRate, then raw file.
func (m *Manager) downloadVoiceWithFallback(ctx context.Context, inst *Instance, item *provider.MessageItem) ([]byte, error) {
	eqp := item.Media.EncryptQueryParam
	aes := item.Media.AESKey

	// Try 1: SILK decode at 24kHz (most common)
	data, err := inst.Provider.DownloadVoice(ctx, eqp, aes, 24000)
	if err == nil {
		slog.Info("voice decoded", "rate", 24000)
		return data, nil
	}
	slog.Warn("voice decode 24kHz failed, trying fallback", "err", err)

	// Try 2: SILK decode at 16kHz
	data, err = inst.Provider.DownloadVoice(ctx, eqp, aes, 16000)
	if err == nil {
		slog.Info("voice decoded", "rate", 16000)
		return data, nil
	}
	slog.Warn("voice decode 16kHz failed, storing raw", "err", err)

	// Try 3: store raw file (SILK or whatever format)
	data, err = inst.Provider.DownloadMedia(ctx, eqp, aes)
	if err != nil {
		return nil, err
	}
	// Change extension to .silk since we couldn't decode
	item.Type = "file"
	slog.Info("voice stored as raw file")
	return data, nil
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


func convertRelayItem(item provider.MessageItem) relay.MessageItem {
	ri := relay.MessageItem{
		Type:     item.Type,
		Text:     item.Text,
		FileName: item.FileName,
	}
	if item.Media != nil {
		ri.Media = &relay.Media{
			URL:         item.Media.URL,
			AESKey:      item.Media.AESKey,
			FileSize:    item.Media.FileSize,
			MediaType:   item.Media.MediaType,
			PlayTime:    item.Media.PlayTime,
			PlayLength:  item.Media.PlayLength,
			ThumbWidth:  item.Media.ThumbWidth,
			ThumbHeight: item.Media.ThumbHeight,
		}
	}
	if item.RefMsg != nil {
		refItem := convertRelayItem(item.RefMsg.Item)
		ri.RefMsg = &relay.RefMsg{
			Title: item.RefMsg.Title,
			Item:  refItem,
		}
	}
	return ri
}
