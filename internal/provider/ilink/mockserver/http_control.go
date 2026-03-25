package mockserver

import (
	"encoding/json"
	"net/http"
)

// --- Control endpoint handlers ---

func (s *HTTPServer) handleMockInbound(w http.ResponseWriter, r *http.Request) {
	var req InboundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request: " + err.Error()})
		return
	}
	s.engine.InjectInbound(req)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *HTTPServer) handleMockSent(w http.ResponseWriter, r *http.Request) {
	msgs := s.engine.SentMessages()
	if msgs == nil {
		msgs = []SentMessage{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *HTTPServer) handleMockScan(w http.ResponseWriter, r *http.Request) {
	s.engine.ScanQR()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *HTTPServer) handleMockConfirm(w http.ResponseWriter, r *http.Request) {
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "bad request: " + err.Error()})
		return
	}
	s.engine.ConfirmQR(creds)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *HTTPServer) handleMockExpire(w http.ResponseWriter, r *http.Request) {
	s.engine.ExpireSession()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *HTTPServer) handleMockListMedia(w http.ResponseWriter, r *http.Request) {
	media := s.engine.ListMedia()
	if media == nil {
		media = []MediaInfo{}
	}
	writeJSON(w, http.StatusOK, media)
}

func (s *HTTPServer) handleMockReset(w http.ResponseWriter, r *http.Request) {
	s.engine.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
