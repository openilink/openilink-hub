// Command appmock runs a mock Hub server for app developers to test
// their apps without needing a real Hub instance or WeChat bot.
//
// It reuses the real api.Server and bot.Manager with an in-memory Store
// and a mock Provider, ensuring the Bot API behaves identically to production.
//
// Additional /mock/* endpoints allow injecting test events and inspecting state.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/api"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/memstore"
)

func main() {
	listen := flag.String("listen", ":9801", "listen address")
	webhookURL := flag.String("webhook-url", "", "app webhook URL for event delivery")
	appToken := flag.String("app-token", "", "custom app_token (auto-generated if empty)")
	appSlug := flag.String("app-slug", "test-app", "app slug / handle")
	flag.Parse()

	// Generate app token if not provided
	if *appToken == "" {
		var b [16]byte
		rand.Read(b[:])
		*appToken = "mock_" + hex.EncodeToString(b[:])
	}

	// IDs
	const (
		botID  = "mock-bot"
		appID  = "mock-app"
		instID = "mock-inst"
		userID = "mock-user"
	)

	// Create in-memory store with pre-configured data
	ms := memstore.New()
	ms.AddBot(&store.Bot{
		ID:       botID,
		UserID:   userID,
		Name:     "Mock Bot",
		Provider: "mock",
		Status:   "connected",
	})
	ms.AddApp(&store.App{
		ID:            appID,
		OwnerID:       userID,
		Name:          "Test App",
		Slug:          *appSlug,
		Description:   "Mock app for development",
		Events:        json.RawMessage(`["message"]`),
		Scopes:        json.RawMessage(`["message:read","message:write","contact:read","bot:read","tools:write"]`),
		Tools:         json.RawMessage(`[]`),
		WebhookURL:    *webhookURL,
		WebhookSecret: "mock-webhook-secret",
		Status:        "active",
	})
	ms.AddInstallation(&store.AppInstallation{
		ID:               instID,
		AppID:            appID,
		BotID:            botID,
		AppToken:         *appToken,
		Handle:           *appSlug,
		Scopes:           json.RawMessage(`["message:read","message:write","contact:read","bot:read","tools:write"]`),
		Tools:            json.RawMessage(`[]`),
		Enabled:          true,
		AppName:          "Test App",
		AppSlug:          *appSlug,
		AppWebhookURL:    *webhookURL,
		AppWebhookSecret: "mock-webhook-secret",
	})

	// Add some mock contacts
	ms.AddContact(store.RecentContact{UserID: "user_alice", LastMsgAt: time.Now().Unix(), MsgCount: 5})
	ms.AddContact(store.RecentContact{UserID: "user_bob", LastMsgAt: time.Now().Unix(), MsgCount: 3})

	// Register mock provider
	mockProv := newMockProvider()
	provider.Register("mock", func() provider.Provider { return mockProv })

	// Create server components
	cfg := &config.Config{
		ListenAddr: *listen,
		RPOrigin:   "http://localhost" + *listen,
		RPID:       "localhost",
		RPName:     "Mock Hub",
		Secret:     "mock-secret",
	}

	hub := relay.NewHub(nil)
	mgr := bot.NewManager(ms, hub, nil, nil, cfg.RPOrigin)
	appWSHub := api.NewAppWSHub()
	mgr.SetAppWSHub(appWSHub)

	srv := &api.Server{
		Store:        ms,
		SessionStore: auth.NewSessionStore(),
		Config:       cfg,
		BotManager:   mgr,
		Hub:          hub,
		AppWSHub:     appWSHub,
	}

	// Start the mock bot
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := mgr.StartBot(ctx, &store.Bot{
		ID:          botID,
		Provider:    "mock",
		Credentials: json.RawMessage(`{}`),
	}); err != nil {
		slog.Error("failed to start mock bot", "err", err)
		os.Exit(1)
	}

	// Build handler: mock control endpoints + real Hub routes
	mux := http.NewServeMux()

	// Control endpoints
	ctrl := &controlHandler{
		store:    ms,
		provider: mockProv,
		appWSHub: appWSHub,
		botID:    botID,
		appID:    appID,
		instID:   instID,
		appToken: *appToken,
		appSlug:  *appSlug,
	}
	mux.HandleFunc("GET /mock/config", ctrl.handleConfig)
	mux.HandleFunc("POST /mock/event", ctrl.handleInjectEvent)
	mux.HandleFunc("GET /mock/messages", ctrl.handleListMessages)
	mux.HandleFunc("POST /mock/reset", ctrl.handleReset)

	// Real Hub routes
	mux.Handle("/", srv.Handler())

	httpSrv := &http.Server{Addr: *listen, Handler: mux}

	go func() {
		<-ctx.Done()
		slog.Info("shutting down...")
		mgr.StopAll()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer shutCancel()
		httpSrv.Shutdown(shutCtx)
	}()

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║              OpeniLink Hub — App Mock Server                ║")
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Printf("║  Listen:     http://localhost%s\n", *listen)
	fmt.Printf("║  App Token:  %s\n", *appToken)
	fmt.Printf("║  App Slug:   %s\n", *appSlug)
	fmt.Printf("║  Bot ID:     %s\n", botID)
	if *webhookURL != "" {
		fmt.Printf("║  Webhook:    %s\n", *webhookURL)
	}
	fmt.Println("╠══════════════════════════════════════════════════════════════╣")
	fmt.Println("║  Bot API:                                                   ║")
	fmt.Printf("║    POST http://localhost%s/bot/v1/message/send\n", *listen)
	fmt.Printf("║    GET  http://localhost%s/bot/v1/contact\n", *listen)
	fmt.Printf("║    GET  http://localhost%s/bot/v1/info\n", *listen)
	fmt.Printf("║    WS   ws://localhost%s/bot/v1/ws?token=%s\n", *listen, *appToken)
	fmt.Println("║  Control:                                                   ║")
	fmt.Printf("║    POST http://localhost%s/mock/event          inject event\n", *listen)
	fmt.Printf("║    GET  http://localhost%s/mock/messages       sent messages\n", *listen)
	fmt.Printf("║    GET  http://localhost%s/mock/config         mock config\n", *listen)
	fmt.Printf("║    POST http://localhost%s/mock/reset          reset state\n", *listen)
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")

	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// --- Control handler ---

type controlHandler struct {
	store    *memstore.Store
	provider *mockProvider
	appWSHub *appdelivery.WSHub
	botID    string
	appID    string
	instID   string
	appToken string
	appSlug  string
}

// handleConfig returns the mock server configuration.
func (c *controlHandler) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"bot_id":    c.botID,
		"app_id":    c.appID,
		"inst_id":   c.instID,
		"app_token": c.appToken,
		"app_slug":  c.appSlug,
	})
}

// handleInjectEvent simulates an inbound message arriving at the mock bot.
// The message flows through the real bot.Manager dispatch pipeline,
// triggering webhook/WebSocket delivery to the connected app.
//
// Request body:
//
//	{
//	  "sender": "user_alice",       // optional, default "user_test"
//	  "content": "hello world",     // required for text
//	  "type": "text",               // optional, default "text"
//	  "group_id": "",               // optional, for group messages
//	}
func (c *controlHandler) handleInjectEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Sender  string `json:"sender"`
		Content string `json:"content"`
		Type    string `json:"type"`
		GroupID string `json:"group_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}
	if req.Content == "" {
		http.Error(w, `{"error":"content is required"}`, http.StatusBadRequest)
		return
	}
	if req.Sender == "" {
		req.Sender = "user_test"
	}
	if req.Type == "" {
		req.Type = "text"
	}

	// Generate a unique external ID
	var b [4]byte
	rand.Read(b[:])
	externalID := strconv.FormatInt(time.Now().UnixMilli(), 10) + hex.EncodeToString(b[:])

	msg := provider.InboundMessage{
		ExternalID:   externalID,
		Sender:       req.Sender,
		Recipient:    "bot",
		GroupID:      req.GroupID,
		Timestamp:    time.Now().UnixMilli(),
		MessageState: 0,
		Items: []provider.MessageItem{
			{Type: req.Type, Text: req.Content},
		},
		ContextToken: "mock-context-token",
	}

	// Inject through the mock provider — triggers bot.Manager.onInbound
	c.provider.InjectMessage(msg)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":          true,
		"external_id": externalID,
		"sender":      req.Sender,
	})
}

// handleListMessages returns messages sent by the app via Bot API.
func (c *controlHandler) handleListMessages(w http.ResponseWriter, r *http.Request) {
	sent := c.store.GetSentMessages()

	// Also include provider-level sends (from sync reply flow)
	provSent := c.provider.GetSent()
	type sentMsg struct {
		To       string `json:"to"`
		Text     string `json:"text,omitempty"`
		FileName string `json:"filename,omitempty"`
		HasMedia bool   `json:"has_media,omitempty"`
	}
	var provMsgs []sentMsg
	for _, m := range provSent {
		provMsgs = append(provMsgs, sentMsg{
			To:       m.Recipient,
			Text:     m.Text,
			FileName: m.FileName,
			HasMedia: len(m.Data) > 0,
		})
	}

	// Build response from store messages
	type storeMsg struct {
		ID       int64           `json:"id"`
		To       string          `json:"to"`
		Items    json.RawMessage `json:"items"`
		CreateAt int64           `json:"created_at"`
	}
	var storeMsgs []storeMsg
	for _, m := range sent {
		storeMsgs = append(storeMsgs, storeMsg{
			ID:       m.ID,
			To:       m.ToUserID,
			Items:    m.ItemList,
			CreateAt: m.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"store_messages":    storeMsgs,
		"provider_messages": provMsgs,
	})
}

// handleReset clears all recorded messages and provider state.
func (c *controlHandler) handleReset(w http.ResponseWriter, r *http.Request) {
	c.store.Reset()
	c.provider.ClearSent()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

