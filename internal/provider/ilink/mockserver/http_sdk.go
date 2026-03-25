package mockserver

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"

	ilink "github.com/openilink/openilink-sdk-go"
)

// --- SDK endpoint handlers ---

func (s *HTTPServer) handleGetUpdates(w http.ResponseWriter, r *http.Request) {
	var req ilink.GetUpdatesReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "bad request"})
		return
	}

	result, err := s.engine.GetUpdates(r.Context(), req.GetUpdatesBuf)
	if err != nil {
		// Session expired returns errcode -14, matching SDK expectations.
		if err.Error() == "session expired" {
			writeJSON(w, http.StatusOK, map[string]any{
				"ret":     0,
				"errcode": -14,
				"errmsg":  "session expired",
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"ret":    0,
			"msgs":  []any{},
			"errmsg": err.Error(),
		})
		return
	}

	// Convert []*WeixinMessage to []WeixinMessage for JSON.
	msgs := make([]ilink.WeixinMessage, len(result.Messages))
	for i, m := range result.Messages {
		msgs[i] = *m
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ret":              0,
		"msgs":             msgs,
		"get_updates_buf":  req.GetUpdatesBuf,
		"sync_buf":         result.SyncBuf,
	})
}

func (s *HTTPServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req ilink.SendMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "bad request"})
		return
	}

	if req.Msg == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "missing msg"})
		return
	}

	if err := s.engine.SendMessage(req.Msg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ret": -1, "errmsg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ret": 0})
}

func (s *HTTPServer) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	var req ilink.GetConfigReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "bad request"})
		return
	}

	result, err := s.engine.GetConfig(req.ILinkUserID, req.ContextToken)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ret": -1, "errmsg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ret":            0,
		"typing_ticket":  result.TypingTicket,
	})
}

func (s *HTTPServer) handleSendTyping(w http.ResponseWriter, r *http.Request) {
	var req ilink.SendTypingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "bad request"})
		return
	}

	typing := req.Status == ilink.Typing
	if err := s.engine.SendTyping(req.ILinkUserID, req.TypingTicket, typing); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ret": -1, "errmsg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"ret": 0})
}

func (s *HTTPServer) handleGetUploadURL(w http.ResponseWriter, r *http.Request) {
	var req ilink.GetUploadURLReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"ret": -1, "errmsg": "bad request"})
		return
	}

	resp, err := s.engine.GetUploadURL(&req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"ret": -1, "errmsg": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ret":          resp.Ret,
		"upload_param": resp.UploadParam,
	})
}

func (s *HTTPServer) handleCDNUpload(w http.ResponseWriter, r *http.Request) {
	uploadParam := r.URL.Query().Get("encrypted_query_param")
	filekey := r.URL.Query().Get("filekey")

	if uploadParam == "" {
		http.Error(w, "missing encrypted_query_param", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 100<<20)) // 100MB max
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}

	downloadEQP, err := s.engine.UploadToCDN(uploadParam, filekey, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("x-encrypted-param", downloadEQP)
	w.WriteHeader(http.StatusOK)
}

func (s *HTTPServer) handleCDNDownload(w http.ResponseWriter, r *http.Request) {
	eqp := r.URL.Query().Get("encrypted_query_param")
	if eqp == "" {
		http.Error(w, "missing encrypted_query_param", http.StatusBadRequest)
		return
	}

	// Return raw ciphertext; the SDK decrypts client-side.
	s.engine.mu.Lock()
	entry, ok := s.engine.media[eqp]
	s.engine.mu.Unlock()

	if !ok {
		http.Error(w, "media not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	w.Write(entry.ciphertext)
}

func (s *HTTPServer) handleFetchQR(w http.ResponseWriter, r *http.Request) {
	result, err := s.engine.FetchQRCode()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"errmsg": err.Error()})
		return
	}

	// qrcode_img_content is base64-encoded image content in the real API.
	imgContent := base64.StdEncoding.EncodeToString([]byte(result.QRContent))

	writeJSON(w, http.StatusOK, map[string]any{
		"qrcode":             result.QRCode,
		"qrcode_img_content": imgContent,
	})
}

func (s *HTTPServer) handlePollQR(w http.ResponseWriter, r *http.Request) {
	qrCode := r.URL.Query().Get("qrcode")
	if qrCode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"status": "error", "errmsg": "missing qrcode"})
		return
	}

	result, err := s.engine.PollQRStatus(r.Context(), qrCode)
	if err != nil {
		// On context cancellation (long-poll timeout), return "wait".
		if r.Context().Err() != nil {
			writeJSON(w, http.StatusOK, map[string]any{"status": "wait"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"status": "error", "errmsg": err.Error()})
		return
	}

	resp := map[string]any{"status": result.Status}

	// The SDK uses "scaned" (one 'n') not "scanned".
	if result.Status == "scanned" {
		resp["status"] = "scaned"
	}

	if result.Creds != nil {
		resp["bot_token"] = result.Creds.BotToken
		resp["ilink_bot_id"] = result.Creds.BotID
		resp["baseurl"] = result.Creds.BaseURL
		resp["ilink_user_id"] = result.Creds.ILinkUserID
	}

	writeJSON(w, http.StatusOK, resp)
}

// writeJSON marshals v as JSON and writes it to w.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
