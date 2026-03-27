package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/provider"
	ilinkProvider "github.com/openilink/openilink-hub/internal/provider/ilink"
)

// --- iLink scan login ---
//
// Allows users to log in (or register) by scanning a WeChat Bot QR code.
// On confirmation the hub finds or creates a user by ilink_user_id, auto-binds
// the bot, and sets a session cookie — all in one step.

func (s *Server) handleScanLoginStart(w http.ResponseWriter, r *http.Request) {
	sessionID, qrURL, err := ilinkProvider.StartBind(r.Context(), "scan-login")
	if err != nil {
		jsonError(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sessionID,
		"qr_url":     qrURL,
	})
}

func (s *Server) handleScanLoginStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	ilinkProvider.PendingBinds.Lock()
	_, ok := ilinkProvider.PendingBinds.M[sessionID]
	ilinkProvider.PendingBinds.Unlock()
	if !ok {
		jsonError(w, "session not found", http.StatusNotFound)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
		return
	}
	defer ws.Close()

	// Read pump: detect client disconnect
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				return
			}
		}
	}()

	sendEvent := func(event, data string) {
		var parsed json.RawMessage
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			parsed, _ = json.Marshal(data)
		}
		msg := map[string]any{"event": event}
		// Merge parsed data fields into the message
		var fields map[string]any
		if json.Unmarshal(parsed, &fields) == nil {
			for k, v := range fields {
				msg[k] = v
			}
		}
		ws.WriteJSON(msg)
	}

	for {
		select {
		case <-done:
			return
		default:
		}

		result, err := ilinkProvider.PollBind(context.Background(), sessionID)
		if err != nil {
			sendEvent("error", `{"message":"poll failed"}`)
			return
		}

		switch result.Status {
		case "wait":
			sendEvent("status", `{"status":"wait"}`)
		case "scanned":
			sendEvent("status", `{"status":"scanned"}`)
		case "expired":
			j, _ := json.Marshal(map[string]string{"status": "refreshed", "qr_url": result.QRURL})
			sendEvent("status", string(j))
		case "confirmed":
			s.completeScanLogin(result, sendEvent)
			return
		}
	}
}

// completeScanLogin handles the "confirmed" state: resolve user via bots table,
// bind the bot, and create a session token.
//
// User resolution order:
//  1. bot_id match → existing bot's user_id (rebind)
//  2. ilink_user_id match → another bot from same iLink user → that user_id
//  3. No match → auto-create a new Hub user
func (s *Server) completeScanLogin(result *provider.BindPollResult, sendEvent func(string, string)) {
	var creds struct {
		BotID       string `json:"bot_id"`
		ILinkUserID string `json:"ilink_user_id"`
	}
	json.Unmarshal(result.Credentials, &creds)

	if creds.ILinkUserID == "" {
		sendEvent("error", `{"message":"no user id from provider"}`)
		return
	}

	// 1. Try to find existing bot by provider_id → get its owner
	var userID string
	var bot *store.Bot
	if creds.BotID != "" {
		existing, _ := s.Store.FindBotByProviderID("ilink", creds.BotID)
		if existing != nil {
			userID = existing.UserID
			// Rebind: update credentials
			s.BotManager.StopBot(existing.ID)
			if err := s.Store.UpdateBotCredentials(existing.ID, creds.BotID, result.Credentials); err != nil {
				sendEvent("error", `{"message":"rebind failed"}`)
				return
			}
			existing.Credentials = result.Credentials
			existing.Status = "connected"
			bot = existing
		}
	}

	// 2. No bot_id match → find another bot from the same ilink_user_id → rebind it
	if bot == nil && creds.ILinkUserID != "" {
		sibling, _ := s.Store.FindBotByCredential("ilink_user_id", creds.ILinkUserID)
		if sibling != nil {
			userID = sibling.UserID
			// Rebind the existing bot with the new credentials/provider_id
			s.BotManager.StopBot(sibling.ID)
			if err := s.Store.UpdateBotCredentials(sibling.ID, creds.BotID, result.Credentials); err != nil {
				sendEvent("error", `{"message":"rebind failed"}`)
				return
			}
			sibling.Credentials = result.Credentials
			sibling.ProviderID = creds.BotID
			sibling.Status = "connected"
			bot = sibling
		}
	}

	// 3. Still no match → create a new Hub user (username: ilink_<bot_id_prefix>)
	if userID == "" {
		// Check registration gate (always allow first user for bootstrap)
		count, _ := s.Store.UserCount()
		if count > 0 && !s.registrationEnabled() {
			sendEvent("error", `{"message":"registration is disabled"}`)
			return
		}
		suffix := creds.BotID
		if len(suffix) > 8 {
			suffix = suffix[:8]
		}
		user, err := s.Store.CreateUser("ilink_"+suffix, "iLink User")
		if err != nil {
			slog.Error("scan-login create user failed", "err", err)
			sendEvent("error", `{"message":"create user failed"}`)
			return
		}
		userID = user.ID
		slog.Info("scan-login auto-created user", "user", userID, "ilink_user_id", creds.ILinkUserID)
	}

	// Verify user is active
	user, err := s.Store.GetUserByID(userID)
	if err != nil {
		sendEvent("error", `{"message":"user not found"}`)
		return
	}
	if user.Status != store.StatusActive {
		sendEvent("error", `{"message":"account disabled"}`)
		return
	}

	// Create new bot if not rebinding
	isNew := bot == nil
	if isNew {
		bot, err = s.Store.CreateBot(userID, "", "ilink", creds.BotID, result.Credentials)
		if err != nil {
			slog.Error("scan-login create bot failed", "err", err)
			sendEvent("error", `{"message":"save failed"}`)
			return
		}
		if _, err := s.Store.CreateChannel(bot.ID, "默认", "", nil, nil); err != nil {
			slog.Error("scan-login create channel failed", "bot", bot.ID, "err", err)
		}
	}

	s.BotManager.StartBot(context.Background(), bot)

	// Login: create session — send token via WS (can't set cookie on WS)
	sessionToken, _ := auth.CreateSession(s.Store, user.ID)

	resp := map[string]any{"status": "connected", "bot_id": bot.ID, "session_token": sessionToken, "is_new": isNew}
	j, _ := json.Marshal(resp)
	sendEvent("status", string(j))
}
