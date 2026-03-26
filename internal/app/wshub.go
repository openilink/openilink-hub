package app

import (
	"sync"

	"github.com/gorilla/websocket"
)

type WSHub struct {
	mu    sync.RWMutex
	conns map[string]*WSConn
}

type WSConn struct {
	InstID string
	BotID  string
	WS     *websocket.Conn
	Send   chan []byte
}

func NewWSHub() *WSHub {
	return &WSHub{conns: make(map[string]*WSConn)}
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
