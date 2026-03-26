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

func (h *WSHub) Unregister(instID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, instID)
}
