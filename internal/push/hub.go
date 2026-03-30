package push

import (
	"encoding/json"
	"log/slog"
	"sync"
)

// Hub manages browser push WebSocket connections, keyed by user ID.
// Each user can have multiple connections (tabs/devices).
type Hub struct {
	mu    sync.RWMutex
	conns map[string]map[*Conn]struct{} // userID -> set of conns
}

func NewHub() *Hub {
	return &Hub{
		conns: make(map[string]map[*Conn]struct{}),
	}
}

// Register adds a connection for a user.
func (h *Hub) Register(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conns[c.UserID] == nil {
		h.conns[c.UserID] = make(map[*Conn]struct{})
	}
	h.conns[c.UserID][c] = struct{}{}
	slog.Debug("push ws registered", "user", c.UserID)
}

// Unregister removes a connection and closes its send channel.
func (h *Hub) Unregister(c *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.conns[c.UserID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.conns, c.UserID)
		}
	}
	c.Close()
	slog.Debug("push ws unregistered", "user", c.UserID)
}

// Notify sends an event to all connections of a user that are subscribed to the given botID.
func (h *Hub) Notify(userID, botID string, env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}

	h.mu.RLock()
	set := h.conns[userID]
	// Copy to avoid holding lock during send.
	conns := make([]*Conn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		if c.IsSubscribed(botID) {
			if !c.Send(data) {
				slog.Warn("push send failed, dropping", "user", userID)
			}
		}
	}
}

// NotifyUser sends an event to all connections of a user (no subscription filter).
func (h *Hub) NotifyUser(userID string, env Envelope) {
	data, err := json.Marshal(env)
	if err != nil {
		return
	}

	h.mu.RLock()
	set := h.conns[userID]
	conns := make([]*Conn, 0, len(set))
	for c := range set {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	for _, c := range conns {
		c.Send(data)
	}
}
