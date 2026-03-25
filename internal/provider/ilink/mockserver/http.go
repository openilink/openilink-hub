package mockserver

import (
	"net/http"
	"net/http/httptest"
)

// HTTPServer wraps an Engine and exposes SDK-compatible HTTP endpoints
// plus control endpoints for test automation.
type HTTPServer struct {
	engine *Engine
	mux    *http.ServeMux
	srv    *httptest.Server
}

// NewHTTPServer creates a new HTTPServer with the given engine options.
func NewHTTPServer(opts ...Option) *HTTPServer {
	e := NewEngine(opts...)
	s := &HTTPServer{engine: e, mux: http.NewServeMux()}

	// SDK endpoints (paths must match openilink-sdk-go exactly).
	s.mux.HandleFunc("POST /ilink/bot/getupdates", s.handleGetUpdates)
	s.mux.HandleFunc("POST /ilink/bot/sendmessage", s.handleSendMessage)
	s.mux.HandleFunc("POST /ilink/bot/getconfig", s.handleGetConfig)
	s.mux.HandleFunc("POST /ilink/bot/sendtyping", s.handleSendTyping)
	s.mux.HandleFunc("POST /ilink/bot/getuploadurl", s.handleGetUploadURL)
	s.mux.HandleFunc("POST /c2c/upload", s.handleCDNUpload)
	s.mux.HandleFunc("GET /c2c/download", s.handleCDNDownload)
	s.mux.HandleFunc("GET /ilink/bot/get_bot_qrcode", s.handleFetchQR)
	s.mux.HandleFunc("GET /ilink/bot/get_qrcode_status", s.handlePollQR)

	// Control endpoints for test automation.
	s.mux.HandleFunc("POST /mock/inbound", s.handleMockInbound)
	s.mux.HandleFunc("GET /mock/sent", s.handleMockSent)
	s.mux.HandleFunc("POST /mock/qr/scan", s.handleMockScan)
	s.mux.HandleFunc("POST /mock/qr/confirm", s.handleMockConfirm)
	s.mux.HandleFunc("POST /mock/session/expire", s.handleMockExpire)
	s.mux.HandleFunc("GET /mock/media", s.handleMockListMedia)
	s.mux.HandleFunc("POST /mock/reset", s.handleMockReset)

	return s
}

// Start launches the test server and returns its URL.
func (s *HTTPServer) Start() string {
	s.srv = httptest.NewServer(s.mux)
	return s.srv.URL
}

// Handler returns the HTTP handler for embedding in other servers.
func (s *HTTPServer) Handler() http.Handler { return s.mux }

// Engine returns the underlying engine for direct manipulation.
func (s *HTTPServer) Engine() *Engine { return s.engine }

// Close shuts down the test server.
func (s *HTTPServer) Close() {
	if s.srv != nil {
		s.srv.Close()
	}
}
