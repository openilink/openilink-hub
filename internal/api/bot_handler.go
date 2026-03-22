package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	ilink "github.com/openilink/openilink-sdk-go"
	"github.com/openilink/openilink-hub/internal/auth"
)

// pendingBinds tracks in-flight QR bind sessions.
var pendingBinds = struct {
	sync.Mutex
	m map[string]*bindSession
}{m: make(map[string]*bindSession)}

type bindSession struct {
	client *ilink.Client
	qrCode string
	userID string
}

func (s *Server) handleListBots(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	bots, err := s.DB.ListBotsByUser(userID)
	if err != nil {
		http.Error(w, `{"error":"list failed"}`, http.StatusInternalServerError)
		return
	}

	// Enrich with live status
	type botResp struct {
		ID          string `json:"id"`
		BotID       string `json:"bot_id"`
		BaseURL     string `json:"base_url"`
		ILinkUserID string `json:"ilink_user_id"`
		Status      string `json:"status"`
		CreatedAt   int64  `json:"created_at"`
	}
	var result []botResp
	for _, b := range bots {
		status := b.Status
		if inst, ok := s.BotManager.GetInstance(b.ID); ok {
			status = inst.Status()
		}
		result = append(result, botResp{
			ID: b.ID, BotID: b.BotID, BaseURL: b.BaseURL,
			ILinkUserID: b.ILinkUserID, Status: status, CreatedAt: b.CreatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleBindStart(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	client := ilink.NewClient("")
	qr, err := client.FetchQRCode(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	sessionID := fmt.Sprintf("bind-%d", r.Context().Value(struct{}{}) ) // simple
	// Use a proper unique ID
	sessionID = fmt.Sprintf("bind-%s-%d", userID[:8], unixMs())

	pendingBinds.Lock()
	pendingBinds.m[sessionID] = &bindSession{
		client: client,
		qrCode: qr.QRCode,
		userID: userID,
	}
	pendingBinds.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sessionID,
		"qr_url":     qr.QRCodeImgContent,
	})
}

func unixMs() int64 {
	return time.Now().UnixMilli()
}

func (s *Server) handleBindStatus(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	pendingBinds.Lock()
	bs, ok := pendingBinds.m[sessionID]
	pendingBinds.Unlock()

	if !ok {
		http.Error(w, `{"error":"session not found"}`, http.StatusNotFound)
		return
	}

	// SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, _ := w.(http.Flusher)

	sendEvent := func(event, data string) {
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		if flusher != nil {
			flusher.Flush()
		}
	}

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		status, err := bs.client.PollQRStatus(ctx, bs.qrCode)
		if err != nil {
			sendEvent("error", `{"message":"poll failed"}`)
			return
		}

		switch status.Status {
		case "wait":
			sendEvent("status", `{"status":"wait"}`)
		case "scaned":
			sendEvent("status", `{"status":"scanned"}`)
		case "expired":
			sendEvent("status", `{"status":"expired"}`)
			// Try to refresh
			newQR, err := bs.client.FetchQRCode(ctx)
			if err != nil {
				sendEvent("error", `{"message":"refresh failed"}`)
				return
			}
			bs.qrCode = newQR.QRCode
			j, _ := json.Marshal(map[string]string{"status": "refreshed", "qr_url": newQR.QRCodeImgContent})
			sendEvent("status", string(j))
		case "confirmed":
			if status.ILinkBotID == "" {
				sendEvent("error", `{"message":"no bot ID"}`)
				return
			}

			bs.client.SetToken(status.BotToken)
			if status.BaseURL != "" {
				bs.client.SetBaseURL(status.BaseURL)
			}

			// Save to DB
			bot, err := s.DB.CreateBot(bs.userID, status.ILinkBotID, status.BotToken, status.BaseURL, status.ILinkUserID)
			if err != nil {
				slog.Error("save bot failed", "err", err)
				sendEvent("error", `{"message":"save failed"}`)
				return
			}

			// Start monitoring
			s.BotManager.StartBot(context.Background(), bot)

			j, _ := json.Marshal(map[string]string{"status": "connected", "bot_id": bot.ID})
			sendEvent("status", string(j))

			// Cleanup
			pendingBinds.Lock()
			delete(pendingBinds.m, sessionID)
			pendingBinds.Unlock()
			return
		}
	}
}

func (s *Server) handleReconnect(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	s.BotManager.StartBot(r.Context(), bot)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleDeleteBot(w http.ResponseWriter, r *http.Request) {
	botID := r.PathValue("id")
	userID := auth.UserIDFromContext(r.Context())

	bot, err := s.DB.GetBot(botID)
	if err != nil || bot.UserID != userID {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}

	s.BotManager.StopBot(botID)
	s.DB.DeleteBot(botID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}
