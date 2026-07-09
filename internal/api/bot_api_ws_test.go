package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
)

// setupWSEnv creates a test environment with AppWSHub wired up.
func setupWSEnv(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	u, err := s.CreateUserFull("testadmin", "", "Test Admin", "hashed", "admin")
	if err != nil {
		t.Fatalf("CreateUserFull: %v", err)
	}
	_ = s.UpdateUserStatus(u.ID, "active")

	srv := &Server{
		Store:       s,
		Config:      &config.Config{RPOrigin: "http://localhost"},
		OAuthStates: newOAuthStateStore(),
		AppWSHub:    app.NewWSHub(),
	}

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	// Store user on server for convenience (tests create bots under this user).
	// We pass it via the returned srv.Store.
	_ = u

	return ts, srv
}

func TestBotAPIWebSocket_Connect(t *testing.T) {
	ts, srv := setupWSEnv(t)

	// Create user, bot, app, installation
	users, _ := srv.Store.ListUsers()
	userID := users[0].ID

	bot := createTestBot(t, srv.Store, userID, "ws-bot")
	appObj := createTestApp(t, srv.Store, userID, "WS App", "ws-app",
		[]string{"message:read", "message:write"})
	inst := installTestApp(t, srv.Store, appObj.ID, bot.ID)

	t.Run("connect with valid token", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=" + inst.AppToken
		ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		defer ws.Close()
		if resp.StatusCode != http.StatusSwitchingProtocols {
			t.Fatalf("expected 101, got %d", resp.StatusCode)
		}

		// Read init message
		ws.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, message, err := ws.ReadMessage()
		if err != nil {
			t.Fatalf("read init: %v", err)
		}
		var initMsg map[string]any
		if err := json.Unmarshal(message, &initMsg); err != nil {
			t.Fatalf("unmarshal init: %v", err)
		}
		if initMsg["type"] != "init" {
			t.Errorf("expected type=init, got %v", initMsg["type"])
		}
		data, ok := initMsg["data"].(map[string]any)
		if !ok {
			t.Fatal("init message missing data")
		}
		if data["installation_id"] != inst.ID {
			t.Errorf("installation_id = %v, want %v", data["installation_id"], inst.ID)
		}
		if data["bot_id"] != bot.ID {
			t.Errorf("bot_id = %v, want %v", data["bot_id"], bot.ID)
		}
	})

	t.Run("connect with invalid token", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=bad-token"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
		if resp != nil && resp.StatusCode == http.StatusSwitchingProtocols {
			t.Fatal("should not upgrade with invalid token")
		}
	})

	t.Run("connect without token", func(t *testing.T) {
		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws"
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("expected error for missing token")
		}
		if resp != nil && resp.StatusCode == http.StatusSwitchingProtocols {
			t.Fatal("should not upgrade without token")
		}
	})

	t.Run("connect with disabled installation", func(t *testing.T) {
		// Only flip Enabled; preserve all other settings (including ReplyPrefixHandle)
		// so the subtest stays focused on the disabled-installation code path.
		if err := srv.Store.UpdateInstallation(inst.ID, inst.Handle, inst.Config, inst.Scopes, false, inst.ReplyPrefixHandle); err != nil {
			t.Fatalf("UpdateInstallation(disable): %v", err)
		}

		wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=" + inst.AppToken
		_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err == nil {
			t.Fatal("expected error for disabled installation")
		}
		if resp != nil && resp.StatusCode == http.StatusSwitchingProtocols {
			t.Fatal("should not upgrade with disabled installation")
		}

		// Re-enable
		if err := srv.Store.UpdateInstallation(inst.ID, inst.Handle, inst.Config, inst.Scopes, true, inst.ReplyPrefixHandle); err != nil {
			t.Fatalf("UpdateInstallation(re-enable): %v", err)
		}
	})
}

func TestBotAPIWebSocket_PingPong(t *testing.T) {
	ts, srv := setupWSEnv(t)

	users, _ := srv.Store.ListUsers()
	userID := users[0].ID

	bot := createTestBot(t, srv.Store, userID, "ping-bot")
	appObj := createTestApp(t, srv.Store, userID, "Ping App", "ping-app",
		[]string{"message:read", "message:write"})
	inst := installTestApp(t, srv.Store, appObj.ID, bot.ID)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=" + inst.AppToken
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	// Read init message first
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err = ws.ReadMessage()
	if err != nil {
		t.Fatalf("read init: %v", err)
	}

	// Send ping
	pingMsg, _ := json.Marshal(map[string]string{"type": "ping"})
	if err := ws.WriteMessage(websocket.TextMessage, pingMsg); err != nil {
		t.Fatalf("write ping: %v", err)
	}

	// Expect pong
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	var pongMsg map[string]any
	if err := json.Unmarshal(message, &pongMsg); err != nil {
		t.Fatalf("unmarshal pong: %v", err)
	}
	if pongMsg["type"] != "pong" {
		t.Errorf("expected type=pong, got %v", pongMsg["type"])
	}
}

func TestBotAPIWebSocket_SendNoBot(t *testing.T) {
	ts, srv := setupWSEnv(t)

	users, _ := srv.Store.ListUsers()
	userID := users[0].ID

	bot := createTestBot(t, srv.Store, userID, "send-bot")
	appObj := createTestApp(t, srv.Store, userID, "Send App", "send-app",
		[]string{"message:read", "message:write"})
	inst := installTestApp(t, srv.Store, appObj.ID, bot.ID)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=" + inst.AppToken
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	// Read init
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, _ = ws.ReadMessage()

	// Send message (will fail because BotManager is nil / bot not connected)
	sendMsg, _ := json.Marshal(map[string]any{
		"type":    "send",
		"req_id":  "r1",
		"content": "hello",
		"to":      "user1",
	})
	if err := ws.WriteMessage(websocket.TextMessage, sendMsg); err != nil {
		t.Fatalf("write send: %v", err)
	}

	// Expect error response
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["type"] != "error" {
		t.Errorf("expected type=error, got %v", resp["type"])
	}
	if resp["req_id"] != "r1" {
		t.Errorf("expected req_id=r1, got %v", resp["req_id"])
	}
}

func TestBotAPIWebSocket_SendMissingScope(t *testing.T) {
	ts, srv := setupWSEnv(t)

	users, _ := srv.Store.ListUsers()
	userID := users[0].ID

	bot := createTestBot(t, srv.Store, userID, "scope-ws-bot")
	// App with only message:read scope (no message:write)
	appObj := createTestApp(t, srv.Store, userID, "ReadOnly App", "readonly-app",
		[]string{"message:read"})
	inst := installTestApp(t, srv.Store, appObj.ID, bot.ID)

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/bot/v1/ws?token=" + inst.AppToken
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer ws.Close()

	// Read init
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, _ = ws.ReadMessage()

	// Try to send (should fail with missing scope)
	sendMsg, _ := json.Marshal(map[string]any{
		"type":    "send",
		"req_id":  "r2",
		"content": "hello",
		"to":      "user1",
	})
	if err := ws.WriteMessage(websocket.TextMessage, sendMsg); err != nil {
		t.Fatalf("write send: %v", err)
	}

	// Expect error about missing scope
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(message, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["type"] != "error" {
		t.Errorf("expected type=error, got %v", resp["type"])
	}
	errMsg, _ := resp["error"].(string)
	if !strings.Contains(errMsg, "message:write") {
		t.Errorf("expected error about message:write scope, got %q", errMsg)
	}
}
