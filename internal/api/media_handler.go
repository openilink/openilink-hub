package api

import (
	"net/http"
	"strings"

	"github.com/openilink/openilink-hub/internal/auth"
)

// GET /api/v1/channels/media?key=xxx&eqp=xxx&aes=xxx
// Proxy: downloads from CDN via bot provider, decrypts and streams back.
// Used when MinIO is not configured.
func (s *Server) handleChannelMedia(w http.ResponseWriter, r *http.Request) {
	ch, err := s.authenticateChannel(r)
	if ch == nil {
		if err != nil {
			http.Error(w, "invalid key", http.StatusUnauthorized)
		} else {
			http.Error(w, "api key required", http.StatusUnauthorized)
		}
		return
	}

	eqp := r.URL.Query().Get("eqp")
	aes := r.URL.Query().Get("aes")
	if eqp == "" || aes == "" {
		http.Error(w, "eqp and aes required", http.StatusBadRequest)
		return
	}

	inst, ok := s.BotManager.GetInstance(ch.BotID)
	if !ok {
		http.Error(w, "bot not connected", http.StatusServiceUnavailable)
		return
	}

	data, err := inst.Provider.DownloadMedia(r.Context(), eqp, aes)
	if err != nil {
		http.Error(w, "download failed", http.StatusBadGateway)
		return
	}

	ct := r.URL.Query().Get("ct")
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(data)
}

// GET /api/v1/media/{key...}
// Proxy: serves files from MinIO through Hub's own domain.
// Key format: media/{bot_id}/{msg_id}/{index}.ext
//
// Auth:
//   - Session cookie: user must own the bot
//   - Channel API key (?key=xxx): channel must belong to the bot
func (s *Server) handleMediaProxy(w http.ResponseWriter, r *http.Request) {
	if s.ObjectStore == nil {
		http.Error(w, "storage not configured", http.StatusNotFound)
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/api/v1/media/")
	if key == "" {
		http.Error(w, "key required", http.StatusBadRequest)
		return
	}

	// Extract bot_id from key: {bot_id}/{msg_id}/{index}.ext
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 2 {
		http.Error(w, "invalid key", http.StatusBadRequest)
		return
	}
	botID := parts[0]

	// Auth: session cookie → check bot ownership
	authed := false
	if cookie, err := r.Cookie("session"); err == nil {
		if uid, err := auth.ValidateSession(s.Store, cookie.Value); err == nil && uid != "" {
			if bot, err := s.Store.GetBot(botID); err == nil && bot.UserID == uid {
				authed = true
			}
		}
	}
	// Auth: channel API key → check channel belongs to this bot
	if !authed {
		if ch, _ := s.authenticateChannel(r); ch != nil && ch.BotID == botID {
			authed = true
		}
	}
	if !authed {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	data, err := s.ObjectStore.Get(r.Context(), key)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ct := http.DetectContentType(data)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}
