package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
)

// generateTraceID creates a random trace ID with the "tr_" prefix.
func generateTraceID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "tr_" + hex.EncodeToString(b)
}

// handleBotAPISend handles POST /bot/v1/messages/send.
func (s *Server) handleBotAPISend(w http.ResponseWriter, r *http.Request) {
	inst := installationFromContext(r.Context())
	if inst == nil {
		botAPIError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check scope
	if !s.requireScope(inst, "message:write") {
		botAPIError(w, "missing scope: message:write", http.StatusForbidden)
		return
	}

	// Parse request body
	var req struct {
		To       string `json:"to"`
		Type     string `json:"type"`
		Content  string `json:"content"`
		URL      string `json:"url"`
		Base64   string `json:"base64"`
		FileName string `json:"filename"`
		TraceID  string `json:"trace_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		botAPIError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type == "" {
		req.Type = "text"
	}

	if req.Type == "text" && req.Content == "" {
		botAPIError(w, "content is required for text messages", http.StatusBadRequest)
		return
	}

	if req.Type != "text" && req.Content == "" && req.URL == "" && req.Base64 == "" {
		botAPIError(w, "content, url, or base64 is required", http.StatusBadRequest)
		return
	}

	// Generate trace ID if not provided
	traceID := req.TraceID
	if traceID == "" {
		traceID = r.Header.Get("X-Trace-Id")
	}
	if traceID == "" {
		traceID = generateTraceID()
	}

	// Get the bot instance
	botInst, ok := s.BotManager.GetInstance(inst.BotID)
	if !ok {
		// Check if bot exists and status
		bot, err := s.Store.GetBot(inst.BotID)
		if err != nil {
			botAPIError(w, "bot not found", http.StatusNotFound)
			return
		}
		if bot.Status == "session_expired" {
			botAPIError(w, "bot session expired", http.StatusServiceUnavailable)
			return
		}
		botAPIError(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	// Check if the bot can send (context_token freshness)
	if canSend, reason := s.checkSendability(inst.BotID, botInst.Status()); !canSend {
		botAPIError(w, reason, http.StatusConflict)
		return
	}

	// Auto-fill context_token from latest message if not available
	contextToken := s.Store.GetLatestContextToken(inst.BotID)

	// Build outbound message
	outMsg := provider.OutboundMessage{
		Recipient:    req.To,
		ContextToken: contextToken,
	}

	itemType := req.Type
	if req.Type == "text" {
		outMsg.Text = req.Content
	} else {
		// Media message: resolve data from base64, url, or content
		var mediaData []byte
		if req.Base64 != "" {
			var decErr error
			var mime string
			mediaData, mime, decErr = base64Decode(req.Base64)
			if decErr != nil {
				botAPIError(w, "invalid base64: "+decErr.Error(), http.StatusBadRequest)
				return
			}
			if mime != "" && req.FileName == "" {
				req.FileName = defaultFileNameFromMIME(mime)
			}
		} else if req.URL != "" {
			var dlErr error
			var mime string
			mediaData, mime, dlErr = downloadURL(r.Context(), req.URL)
			if dlErr != nil {
				botAPIError(w, "download failed: "+dlErr.Error(), http.StatusBadGateway)
				return
			}
			if mime != "" && req.FileName == "" {
				req.FileName = defaultFileNameFromMIME(mime)
			}
		} else {
			// Fallback: send content as text
			outMsg.Text = req.Content
			itemType = "text"
		}
		if mediaData != nil {
			outMsg.Data = mediaData
			outMsg.FileName = req.FileName
			if outMsg.FileName == "" {
				outMsg.FileName = defaultFileName(req.Type, mediaData)
			}
		}
	}

	clientID, err := botInst.Send(r.Context(), outMsg)
	if err != nil {
		slog.Error("bot api: send failed", "bot_id", inst.BotID, "err", err)
		botAPIError(w, "send failed: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Store media to ObjectStore if available
	mediaStatus := ""
	mediaKeys := json.RawMessage(`{}`)
	mediaKey := ""
	if len(outMsg.Data) > 0 && s.ObjectStore != nil {
		ct := detectContentType(itemType)
		ext := detectExt(outMsg.FileName, itemType)
		now := time.Now()
		var rnd [4]byte
		rand.Read(rnd[:])
		key := fmt.Sprintf("%s/%s/out_%d_%x%s", inst.BotID,
			now.Format("2006/01/02"), now.UnixMilli(), rnd, ext)
		if _, err := s.ObjectStore.Put(r.Context(), key, ct, outMsg.Data); err == nil {
			mediaStatus = "ready"
			mediaKeys, _ = json.Marshal(map[string]string{"0": key})
			mediaKey = key
		} else {
			slog.Warn("bot api: objectstore put failed", "key", key, "err", err)
		}
	}

	// Save outbound message to DB
	item := map[string]any{"type": itemType}
	if outMsg.Text != "" {
		item["text"] = outMsg.Text
	}
	if outMsg.FileName != "" {
		item["file_name"] = outMsg.FileName
	}
	itemList, _ := json.Marshal([]any{item})
	s.Store.SaveMessage(&store.Message{
		BotID:       inst.BotID,
		Direction:   "outbound",
		ToUserID:    req.To,
		MessageType: 2,
		ItemList:    itemList,
		MediaStatus: mediaStatus,
		MediaKeys:   mediaKeys,
	})

	// Append span to message trace if trace_id links to an existing trace
	if traceID != "" {
		replyContent := req.Content
		if req.Type != "text" {
			replyContent = "[" + req.Type + "] " + outMsg.FileName
		}
		spanAttrs := map[string]any{
			"app.name":      inst.AppName,
			"reply.type":    req.Type,
			"reply.to":      req.To,
			"reply.content": replyContent,
		}
		if mediaKey != "" {
			spanAttrs["reply.media_key"] = mediaKey
		}
		_ = s.Store.AppendSpan(traceID, inst.BotID, "Bot API send_reply", store.SpanKindServer, store.StatusOK, "", spanAttrs)
	}

	// Respond
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":        true,
		"client_id": clientID,
		"trace_id":  traceID,
	})
}

// handleBotAPIContacts handles GET /bot/v1/contacts.
func (s *Server) handleBotAPIContacts(w http.ResponseWriter, r *http.Request) {
	inst := installationFromContext(r.Context())
	if inst == nil {
		botAPIError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check scope
	if !s.requireScope(inst, "contact:read") {
		botAPIError(w, "missing scope: contact:read", http.StatusForbidden)
		return
	}

	contacts, err := s.Store.ListRecentContacts(inst.BotID, 100)
	if err != nil {
		slog.Error("bot api: list contacts failed", "bot_id", inst.BotID, "err", err)
		botAPIError(w, "failed to list contacts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"contacts": contacts,
	})
}

// handleBotAPIBotInfo handles GET /bot/v1/bot.
func (s *Server) handleBotAPIBotInfo(w http.ResponseWriter, r *http.Request) {
	inst := installationFromContext(r.Context())
	if inst == nil {
		botAPIError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check scope
	if !s.requireScope(inst, "bot:read") {
		botAPIError(w, "missing scope: bot:read", http.StatusForbidden)
		return
	}

	bot, err := s.Store.GetBot(inst.BotID)
	if err != nil {
		slog.Error("bot api: get bot failed", "bot_id", inst.BotID, "err", err)
		botAPIError(w, "bot not found", http.StatusNotFound)
		return
	}

	// Get live status from manager if available
	status := bot.Status
	if botInst, ok := s.BotManager.GetInstance(inst.BotID); ok {
		status = botInst.Status()
	}

	// Build response — exclude sensitive fields like credentials
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"bot": map[string]any{
			"id":         bot.ID,
			"name":       bot.Name,
			"provider":   bot.Provider,
			"status":     status,
			"msg_count":  bot.MsgCount,
			"created_at": bot.CreatedAt,
			"updated_at": bot.UpdatedAt,
		},
	})
}

// handleBotAPINotFound returns a 404 for unknown Bot API paths.
func (s *Server) handleBotAPINotFound(w http.ResponseWriter, r *http.Request) {
	_ = time.Now() // ensure time import used
	botAPIError(w, "unknown endpoint", http.StatusNotFound)
}

func defaultFileNameFromMIME(mime string) string {
	switch mime {
	case "image/png":
		return "image.png"
	case "image/jpeg":
		return "image.jpg"
	case "image/gif":
		return "image.gif"
	case "image/webp":
		return "image.webp"
	case "video/mp4":
		return "video.mp4"
	default:
		if strings.HasPrefix(mime, "image/") {
			return "image." + strings.TrimPrefix(mime, "image/")
		}
		if strings.HasPrefix(mime, "video/") {
			return "video." + strings.TrimPrefix(mime, "video/")
		}
		return "file"
	}
}

func defaultFileName(mediaType string, data []byte) string {
	switch mediaType {
	case "image":
		if len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8 {
			return "image.jpg"
		}
		if len(data) > 4 && string(data[:4]) == "GIF8" {
			return "image.gif"
		}
		return "image.png"
	case "video":
		return "video.mp4"
	default:
		return "file"
	}
}

// parseBase64Input extracts pure base64 data and MIME type from a string
// that may be a data URI (data:image/png;base64,...) or plain base64.
func parseBase64Input(s string) (b64, mime string) {
	if strings.HasPrefix(s, "data:") {
		commaIdx := strings.Index(s, ",")
		if commaIdx > 0 {
			header := s[5:commaIdx]
			b64 = s[commaIdx+1:]
			semicolonIdx := strings.Index(header, ";")
			if semicolonIdx > 0 {
				mime = header[:semicolonIdx]
			} else {
				mime = header
			}
			return
		}
	}
	return s, ""
}

func base64Decode(s string) ([]byte, string, error) {
	b64, mime := parseBase64Input(s)
	data, err := base64.StdEncoding.DecodeString(b64)
	return data, mime, err
}

// handleBotAPIUpdateTools handles PUT /bot/v1/app/tools.
// App dynamically updates its own tools. Requires tools:write scope.
// Only works for local apps (registry="").
func (s *Server) handleBotAPIUpdateTools(w http.ResponseWriter, r *http.Request) {
	inst := installationFromContext(r.Context())
	if inst == nil {
		botAPIError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Check scope
	if !s.requireScope(inst, "tools:write") {
		botAPIError(w, "missing scope: tools:write", http.StatusForbidden)
		return
	}

	// Get app and verify it's a local app (not marketplace/builtin)
	app, err := s.Store.GetApp(inst.AppID)
	if err != nil {
		botAPIError(w, "app not found", http.StatusNotFound)
		return
	}
	if app.Registry != "" {
		botAPIError(w, "marketplace and builtin apps cannot be modified via API", http.StatusForbidden)
		return
	}

	var req struct {
		Tools json.RawMessage `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Tools == nil {
		botAPIError(w, "tools required", http.StatusBadRequest)
		return
	}

	// Validate tools is a JSON array
	var toolsCheck []any
	if err := json.Unmarshal(req.Tools, &toolsCheck); err != nil {
		botAPIError(w, "tools must be a JSON array", http.StatusBadRequest)
		return
	}

	// Update the app's tools
	if err := s.Store.UpdateAppTools(app.ID, req.Tools); err != nil {
		botAPIError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Log the update
	slog.Info("app tools updated via bot API", "app_id", app.ID, "app_slug", app.Slug, "tool_count", len(toolsCheck))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "tool_count": len(toolsCheck)})
}

// handleBotAPIUpdateInstallationTools handles PUT /bot/v1/installation/tools.
// Updates this installation's per-installation tools. Requires tools:write scope.
func (s *Server) handleBotAPIUpdateInstallationTools(w http.ResponseWriter, r *http.Request) {
	inst := installationFromContext(r.Context())
	if inst == nil {
		botAPIError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if !s.requireScope(inst, "tools:write") {
		botAPIError(w, "missing scope: tools:write", http.StatusForbidden)
		return
	}

	var req struct {
		Tools json.RawMessage `json:"tools"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Tools == nil {
		botAPIError(w, "tools required", http.StatusBadRequest)
		return
	}

	var toolsCheck []any
	if err := json.Unmarshal(req.Tools, &toolsCheck); err != nil {
		botAPIError(w, "tools must be a JSON array", http.StatusBadRequest)
		return
	}

	if err := s.Store.UpdateInstallationTools(inst.ID, req.Tools); err != nil {
		botAPIError(w, "update failed", http.StatusInternalServerError)
		return
	}

	slog.Info("installation tools updated", "inst_id", inst.ID, "tool_count", len(toolsCheck))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "tool_count": len(toolsCheck)})
}

// isSafeURL checks that a URL is not targeting internal/private network addresses.
func isSafeURL(rawURL string) error {
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", scheme)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("empty host")
	}

	// Block well-known internal hostnames
	lower := strings.ToLower(host)
	if lower == "localhost" || strings.HasSuffix(lower, ".local") || lower == "metadata.google.internal" {
		return fmt.Errorf("internal host not allowed: %s", host)
	}

	// Resolve and check IP
	ip := net.ParseIP(host)
	if ip == nil {
		// Hostname — resolve it
		ips, err := net.LookupIP(host)
		if err != nil || len(ips) == 0 {
			return fmt.Errorf("cannot resolve host: %s", host)
		}
		ip = ips[0]
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
		return fmt.Errorf("private/internal IP not allowed: %s", ip)
	}

	// Block cloud metadata IPs (169.254.169.254, fd00::, etc.)
	if ip.Equal(net.ParseIP("169.254.169.254")) {
		return fmt.Errorf("metadata endpoint not allowed")
	}

	return nil
}

func downloadURL(ctx context.Context, url string) ([]byte, string, error) {
	if err := isSafeURL(url); err != nil {
		return nil, "", fmt.Errorf("blocked: %w", err)
	}

	dlCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(dlCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	ct := strings.SplitN(resp.Header.Get("Content-Type"), ";", 2)[0]
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20*1024*1024))
	return data, strings.TrimSpace(ct), err
}
