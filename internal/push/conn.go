package push

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxMsgSize = 4 * 1024
)

// Conn is a single browser push WebSocket connection.
type Conn struct {
	UserID string
	ws     *websocket.Conn
	hub    *Hub
	send   chan []byte

	mu   sync.RWMutex
	subs map[string]struct{} // subscribed bot IDs
}

func NewConn(userID string, ws *websocket.Conn, hub *Hub) *Conn {
	return &Conn{
		UserID: userID,
		ws:     ws,
		hub:    hub,
		send:   make(chan []byte, 64),
		subs:   make(map[string]struct{}),
	}
}

// IsSubscribed checks if this connection is subscribed to a botID.
func (c *Conn) IsSubscribed(botID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subs[botID]
	return ok
}

// ReadPump reads subscribe/unsubscribe commands from the client.
func (c *Conn) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMsgSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			break
		}

		var env Envelope
		if err := json.Unmarshal(msg, &env); err != nil {
			continue
		}

		switch env.Type {
		case "subscribe":
			var sub Subscribe
			if json.Unmarshal(env.Data, &sub) == nil {
				c.mu.Lock()
				for _, id := range sub.BotIDs {
					c.subs[id] = struct{}{}
				}
				c.mu.Unlock()
				slog.Debug("push subscribe", "user", c.UserID, "bots", sub.BotIDs)
			}
		case "unsubscribe":
			var unsub Unsubscribe
			if json.Unmarshal(env.Data, &unsub) == nil {
				c.mu.Lock()
				for _, id := range unsub.BotIDs {
					delete(c.subs, id)
				}
				c.mu.Unlock()
			}
		case "ping":
			select {
			case c.send <- mustMarshal(Envelope{Type: "pong"}):
			default:
			}
		}
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Conn) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
