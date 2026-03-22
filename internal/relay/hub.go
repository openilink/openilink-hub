package relay

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// UpstreamHandler is called when a sub-level client sends a message upstream.
type UpstreamHandler func(conn *Conn, env Envelope)

// Hub manages all active WebSocket connections.
type Hub struct {
	mu              sync.RWMutex
	conns           map[string]*Conn // sublevelID -> conn
	upstreamHandler UpstreamHandler
}

func NewHub(handler UpstreamHandler) *Hub {
	return &Hub{
		conns:           make(map[string]*Conn),
		upstreamHandler: handler,
	}
}

func (h *Hub) Register(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection for same sublevel
	if old, ok := h.conns[c.SublevelID]; ok {
		close(old.send)
	}
	h.conns[c.SublevelID] = c
	slog.Info("ws registered", "sublevel", c.SublevelID)
}

func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.conns[c.SublevelID]; ok && existing == c {
		delete(h.conns, c.SublevelID)
		close(c.send)
		slog.Info("ws unregistered", "sublevel", c.SublevelID)
	}
}

// SendTo sends an envelope to a specific sub-level.
func (h *Hub) SendTo(sublevelID string, env Envelope) {
	h.mu.RLock()
	c, ok := h.conns[sublevelID]
	h.mu.RUnlock()
	if ok {
		c.Send(env)
	}
}

// Broadcast sends an envelope to all connected sub-levels for a given bot.
func (h *Hub) Broadcast(botDBID string, env Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(env)
	if err != nil {
		return
	}
	for _, c := range h.conns {
		if c.BotDBID == botDBID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

// HandleUpstream routes a message from a sub-level client.
func (h *Hub) HandleUpstream(c *Conn, env Envelope) {
	if h.upstreamHandler != nil {
		h.upstreamHandler(c, env)
	}
}

// ConnectedCount returns the number of active connections.
func (h *Hub) ConnectedCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}
