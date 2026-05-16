package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

// handleBotAPIWebSocket handles GET /bot/v1/ws?token={app_token}.
// It authenticates via query param (not the appTokenAuth middleware),
// upgrades to WebSocket, and starts read/write pumps.
func (s *Server) handleBotAPIWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Auth: extract token from query param
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "token required", http.StatusUnauthorized)
		return
	}

	// 2. Look up installation
	inst, err := s.Store.GetInstallationByToken(token)
	if err != nil || !inst.Enabled {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 3. Upgrade to WebSocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// 4. Create connection and register
	conn := &app.WSConn{
		InstID:   inst.ID,
		BotID:    inst.BotID,
		AppSlug:  inst.AppSlug,
		AppToken: inst.AppToken,
		WS:       ws,
		Send:     make(chan []byte, 64),
	}
	s.AppWSHub.Register(inst.ID, conn)

	slog.Info("app ws connected", "inst", inst.ID, "app", inst.AppSlug, "bot", inst.BotID)

	// 5. Send init message
	conn.SendJSON(map[string]any{
		"type": "init",
		"data": map[string]any{
			"installation_id": inst.ID,
			"bot_id":          inst.BotID,
			"app_name":        inst.AppName,
			"app_slug":        inst.AppSlug,
		},
	})

	// 6. Start read/write pumps
	go conn.WritePump()
	conn.ReadPump(s) // blocks until disconnect

	slog.Info("app ws disconnected", "inst", inst.ID, "app", inst.AppSlug)
}

// HandleAppWSSend handles a "send" message received over WebSocket.
func (s *Server) HandleAppWSSend(conn *app.WSConn, msg map[string]any) {
	reqID, _ := msg["req_id"].(string)
	content, _ := msg["content"].(string)
	to, _ := msg["to"].(string)
	msgType, _ := msg["msg_type"].(string)
	if msgType == "" {
		msgType = "text"
	}

	sendErr := func(errMsg string) {
		resp := map[string]any{"type": "error", "error": errMsg}
		if reqID != "" {
			resp["req_id"] = reqID
		}
		conn.SendJSON(resp)
	}

	sendAck := func() {
		resp := map[string]any{"type": "ack", "ok": true}
		if reqID != "" {
			resp["req_id"] = reqID
		}
		conn.SendJSON(resp)
	}

	// Check scope: need message:write
	instFull, err := s.Store.GetInstallationByToken(conn.AppToken)
	if err != nil {
		sendErr("installation not found")
		return
	}
	if !s.requireScope(instFull, "message:write") {
		sendErr("missing scope: message:write")
		return
	}

	if to == "" {
		sendErr("to is required")
		return
	}

	if msgType == "text" && content == "" {
		sendErr("content is required for text messages")
		return
	}

	// Get the bot instance
	if s.BotManager == nil {
		sendErr("bot not connected")
		return
	}
	botInst, ok := s.BotManager.GetInstance(conn.BotID)
	if !ok {
		sendErr("bot not connected")
		return
	}

	// Check if the bot can send
	if canSend, reason := s.checkSendability(conn.BotID, botInst.Status()); !canSend {
		sendErr(reason)
		return
	}

	contextToken := s.Store.GetLatestContextToken(conn.BotID)

	outMsg := provider.OutboundMessage{
		Recipient:    to,
		Text:         content,
		ContextToken: contextToken,
	}

	clientID, err := botInst.Send(context.Background(), outMsg)
	if err != nil {
		slog.Error("app ws send failed", "bot_id", conn.BotID, "err", err)
		sendErr("send failed: " + err.Error())
		return
	}

	// Save outbound message to DB
	item := map[string]any{"type": msgType, "text": content}
	itemList, _ := json.Marshal([]any{item})
	s.Store.SaveMessage(&store.Message{
		BotID:       conn.BotID,
		Direction:   "outbound",
		ToUserID:    to,
		MessageType: 2,
		ItemList:    itemList,
	})
	s.writePlatformOutboundMessage(context.Background(), conn.BotID, "", to, contextToken, content, itemList, clientID, map[string]any{
		"source":       "openilink_hub_app_ws",
		"message_type": msgType,
	})

	slog.Info("app ws send ok", "bot_id", conn.BotID, "client_id", clientID, "to", to)
	sendAck()
}

// handleAppLevelWebSocket handles GET /bot/v1/app/ws?app_id={app_id}&secret={webhook_secret}.
// App-level WS: one connection receives events for ALL installations of this app on this Hub.
func (s *Server) handleAppLevelWebSocket(w http.ResponseWriter, r *http.Request) {
	appID := r.URL.Query().Get("app_id")
	secret := r.URL.Query().Get("secret")
	if appID == "" || secret == "" {
		http.Error(w, "app_id and secret required", http.StatusUnauthorized)
		return
	}

	a, err := s.Store.GetApp(appID)
	if err != nil {
		http.Error(w, "app not found", http.StatusUnauthorized)
		return
	}
	if a.WebhookSecret != secret {
		http.Error(w, "invalid secret", http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn := &app.WSConn{
		InstID:  "app:" + appID, // special prefix for app-level connections
		BotID:   "",             // receives from all bots
		AppSlug: a.Slug,
		WS:      ws,
		Send:    make(chan []byte, 64),
	}
	s.AppWSHub.RegisterAppLevel(appID, conn)

	slog.Info("app-level ws connected", "app_id", appID, "slug", a.Slug)

	conn.SendJSON(map[string]any{
		"type": "init",
		"data": map[string]any{
			"app_id":   appID,
			"app_slug": a.Slug,
			"scope":    "app",
		},
	})

	go conn.WritePump()
	conn.ReadPump(s)

	slog.Info("app-level ws disconnected", "app_id", appID)
}

// GetAppWSHub returns the App WebSocket hub. Implements app.ReadPumpHandler.
func (s *Server) GetAppWSHub() *app.WSHub {
	return s.AppWSHub
}

// Ensure Server implements ReadPumpHandler at compile time.
var _ app.ReadPumpHandler = (*Server)(nil)

// NewAppWSHub creates a new WSHub for app installations. Called in main.go.
func NewAppWSHub() *app.WSHub {
	return app.NewWSHub()
}
