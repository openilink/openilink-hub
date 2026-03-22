package relay

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	maxMsgSize = 64 * 1024
)

// Conn wraps a single WebSocket connection for a sub-level client.
type Conn struct {
	SublevelID string
	BotDBID    string
	ws         *websocket.Conn
	hub        *Hub
	send       chan []byte
}

func NewConn(sublevelID, botDBID string, ws *websocket.Conn, hub *Hub) *Conn {
	return &Conn{
		SublevelID: sublevelID,
		BotDBID:    botDBID,
		ws:         ws,
		hub:        hub,
		send:       make(chan []byte, 64),
	}
}

func (c *Conn) Send(env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("ws send buffer full, dropping", "sublevel", c.SublevelID)
	}
}

// ReadPump reads messages from WebSocket and passes to hub.
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

		if env.Type == "ping" {
			c.Send(Envelope{Type: "pong"})
			continue
		}

		c.hub.HandleUpstream(c, env)
	}
}

// WritePump writes messages from send channel to WebSocket.
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
