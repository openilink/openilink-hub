package api

import (
	"net/http"
	"strings"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/provider"
)

// GET /api/v1/channels/media?bot=xxx&eqp=xxx&aes=xxx
// CDN proxy: downloads from provider CDN via bot, decrypts and streams back.
// Used when no storage backend (S3/FS) is configured.
//
// Auth:
//   - Channel API key (?key=xxx): channel must belong to the bot
//   - Session cookie + bot query param: user must own the bot
func (s *Server) handleChannelMedia(w http.ResponseWriter, r *http.Request) {
	eqp := r.URL.Query().Get("eqp")
	aes := r.URL.Query().Get("aes")
	if eqp == "" || aes == "" {
		http.Error(w, "eqp and aes required", http.StatusBadRequest)
		return
	}

	// Auth path 1: channel API key
	if ch, _ := s.authenticateChannel(r); ch != nil {
		inst, ok := s.BotManager.GetInstance(ch.BotID)
		if !ok {
			http.Error(w, "bot not connected", http.StatusServiceUnavailable)
			return
		}
		s.serveChannelMedia(w, r, inst, eqp, aes)
		return
	}

	// Auth path 2: session cookie + explicit bot ID
	botID := r.URL.Query().Get("bot")
	if botID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if cookie, err := r.Cookie("session"); err == nil {
		if uid, err := auth.ValidateSession(s.Store, cookie.Value); err == nil && uid != "" {
			b, err := s.Store.GetBot(botID)
			if err != nil || b.UserID != uid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			inst, ok := s.BotManager.GetInstance(botID)
			if !ok {
				http.Error(w, "bot not connected", http.StatusServiceUnavailable)
				return
			}
			s.serveChannelMedia(w, r, inst, eqp, aes)
			return
		}
	}

	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

func (s *Server) serveChannelMedia(w http.ResponseWriter, r *http.Request, inst *bot.Instance, eqp, aes string) {
	data, err := inst.Provider.DownloadMedia(r.Context(), &provider.Media{
		EncryptQueryParam: eqp,
		AESKey:            aes,
	})
	if err != nil {
		http.Error(w, "download failed", http.StatusBadGateway)
		return
	}

	ct := http.DetectContentType(data)
	safe := (strings.HasPrefix(ct, "image/") && ct != "image/svg+xml") ||
		strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/")
	if !safe {
		w.Header().Set("Content-Disposition", "attachment")
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}

// GET /api/v1/media/{key...}
// Serves files from storage (S3 or local filesystem) through Hub.
// Key format: {bot_id}/{date}/{filename}
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

	// Extract bot_id from key: {bot_id}/{date}/{filename}
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
	// Only allow safe content types to be rendered inline; everything else
	// is forced to download to prevent same-origin script execution (e.g. HTML/SVG).
	safe := (strings.HasPrefix(ct, "image/") && ct != "image/svg+xml") ||
		strings.HasPrefix(ct, "audio/") || strings.HasPrefix(ct, "video/")
	if !safe {
		w.Header().Set("Content-Disposition", "attachment")
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Write(data)
}
