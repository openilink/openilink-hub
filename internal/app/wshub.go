package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSHub struct {
	mu       sync.RWMutex
	conns    map[string]*WSConn   // installation_id → conn
	appConns map[string]*WSConn   // app_id → conn (app-level WS)
}

type WSConn struct {
	InstID   string
	BotID    string
	AppSlug  string
	AppToken string
	WS       *websocket.Conn
	Send     chan []byte
}

func NewWSHub() *WSHub {
	return &WSHub{
		conns:    make(map[string]*WSConn),
		appConns: make(map[string]*WSConn),
	}
}

func (h *WSHub) Get(instID string) *WSConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.conns[instID]
}

func (h *WSHub) Register(instID string, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[instID] = c
}

func (h *WSHub) Unregister(instID string, conn *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Only delete if the registered connection matches the one being removed.
	// A newer connection may have replaced it via Register, so we must not
	// delete the new connection when the old one's cleanup goroutine fires.
	if h.conns[instID] == conn {
		delete(h.conns, instID)
	}
}

// RegisterAppLevel registers an app-level WS connection (receives events for all installations).
func (h *WSHub) RegisterAppLevel(appID string, c *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.appConns[appID]; ok {
		close(old.Send)
	}
	h.appConns[appID] = c
}

// GetAppLevel returns the app-level WS connection for an app.
func (h *WSHub) GetAppLevel(appID string) *WSConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.appConns[appID]
}

// UnregisterAppLevel removes an app-level connection.
func (h *WSHub) UnregisterAppLevel(appID string, conn *WSConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.appConns[appID] == conn {
		delete(h.appConns, appID)
	}
}

// SendJSON marshals v as JSON and enqueues it on the connection's send buffer.
func (c *WSConn) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.Send <- data:
		return nil
	default:
		return fmt.Errorf("send buffer full")
	}
}

// WritePump pumps messages from the Send channel to the websocket connection.
// It also sends periodic ping frames to keep the connection alive.
func (c *WSConn) WritePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.WS.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.WS.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.WS.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.WS.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPumpHandler is the interface that the read pump calls back into
// for handling upstream messages (e.g. "send").
type ReadPumpHandler interface {
	HandleAppWSSend(conn *WSConn, msg map[string]any)
	GetAppWSHub() *WSHub
}

// ReadPump reads messages from the websocket and dispatches them.
// It blocks until the connection is closed.
func (c *WSConn) ReadPump(h ReadPumpHandler) {
	defer func() {
		hub := h.GetAppWSHub()
		hub.Unregister(c.InstID, c)
		// Also try unregistering as app-level connection
		if len(c.InstID) > 4 && c.InstID[:4] == "app:" {
			hub.UnregisterAppLevel(c.InstID[4:], c)
		}
		c.WS.Close()
	}()

	c.WS.SetReadLimit(64 * 1024) // 64KB
	c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.WS.SetPongHandler(func(string) error {
		c.WS.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.WS.ReadMessage()
		if err != nil {
			slog.Info("ws read ended", "inst", c.InstID, "app", c.AppSlug, "err", err)
			break
		}

		var msg map[string]any
		if err := json.Unmarshal(message, &msg); err != nil {
			slog.Warn("ws unmarshal failed", "inst", c.InstID, "app", c.AppSlug, "err", err)
			continue
		}

		msgType, _ := msg["type"].(string)
		slog.Debug("ws recv", "inst", c.InstID, "app", c.AppSlug, "type", msgType)

		switch msgType {
		case "ping":
			c.SendJSON(map[string]string{"type": "pong"})
		case "send":
			h.HandleAppWSSend(c, msg)
		default:
			slog.Warn("ws unknown msg type", "inst", c.InstID, "app", c.AppSlug, "type", msgType)
		}
	}
}
