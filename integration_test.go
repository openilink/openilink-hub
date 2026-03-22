package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/openilink/openilink-hub/internal/api"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/database"
	"github.com/openilink/openilink-hub/internal/provider"
	mockProvider "github.com/openilink/openilink-hub/internal/provider/mock"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/storage"
)

// ==================== Test infrastructure ====================

func testDB(t *testing.T) *database.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://openilink:openilink@localhost:15432/openilink_test?sslmode=disable"
	}
	db, err := database.Open(dsn)
	if err != nil {
		t.Skipf("skip: database unavailable: %v", err)
	}
	for _, table := range []string{"messages", "channels", "bots", "oauth_accounts", "sessions", "credentials", "users", "system_config"} {
		db.Exec("DELETE FROM " + table)
	}
	return db
}

type testEnv struct {
	t      *testing.T
	db     *database.DB
	srv    *httptest.Server
	client *http.Client
	mgr    *bot.Manager
	hub    *relay.Hub
	cfg    *config.Config
}

func setup(t *testing.T) *testEnv {
	t.Helper()
	db := testDB(t)

	cfg := &config.Config{
		RPOrigin: "http://localhost",
		RPID:     "localhost",
		RPName:   "Test",
		Secret:   "test-secret",
	}

	server := &api.Server{
		DB:           db,
		SessionStore: auth.NewSessionStore(),
		Config:       cfg,
		OAuthStates:  api.SetupOAuth(cfg),
	}

	hub := relay.NewHub(server.SetupUpstreamHandler())
	sinks := []sink.Sink{
		&sink.WS{Hub: hub},
		&sink.AI{DB: db},
		&sink.Webhook{DB: db},
	}
	mgr := bot.NewManager(db, hub, sinks, nil, "http://localhost")
	server.BotManager = mgr
	server.Hub = hub

	ts := httptest.NewServer(server.Handler())
	jar, _ := cookiejar.New(nil)

	return &testEnv{
		t: t, db: db, srv: ts, cfg: cfg,
		client: &http.Client{Jar: jar},
		mgr: mgr, hub: hub,
	}
}

func (e *testEnv) close() {
	e.mgr.StopAll()
	e.srv.Close()
	e.db.Close()
}

// newClient returns a fresh HTTP client (separate cookie jar = separate session).
func (e *testEnv) newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

// ==================== HTTP helpers ====================

func (e *testEnv) postRaw(path string, body any) *http.Response {
	e.t.Helper()
	data, _ := json.Marshal(body)
	resp, err := e.client.Post(e.srv.URL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		e.t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func (e *testEnv) postCode(path string, body any) (int, map[string]any) {
	e.t.Helper()
	resp := e.postRaw(path, body)
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func (e *testEnv) post(path string, body any) map[string]any {
	e.t.Helper()
	_, result := e.postCode(path, body)
	return result
}

func (e *testEnv) get(path string) (int, map[string]any) {
	e.t.Helper()
	resp, err := e.client.Get(e.srv.URL + path)
	if err != nil {
		e.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func (e *testEnv) getList(path string) (int, []any) {
	e.t.Helper()
	resp, err := e.client.Get(e.srv.URL + path)
	if err != nil {
		e.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result []any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func (e *testEnv) del(path string) (int, map[string]any) {
	e.t.Helper()
	req, _ := http.NewRequest("DELETE", e.srv.URL+path, nil)
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("DELETE %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func (e *testEnv) put(path string, body any) (int, map[string]any) {
	e.t.Helper()
	data, _ := json.Marshal(body)
	req, _ := http.NewRequest("PUT", e.srv.URL+path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("PUT %s: %v", path, err)
	}
	defer resp.Body.Close()
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func (e *testEnv) register(username, password string) {
	e.t.Helper()
	code, result := e.postCode("/api/auth/register", map[string]string{"username": username, "password": password})
	if code != 200 {
		e.t.Fatalf("register %s failed: %d %v", username, code, result["error"])
	}
}

func (e *testEnv) login(username, password string) {
	e.t.Helper()
	code, result := e.postCode("/api/auth/login", map[string]string{"username": username, "password": password})
	if code != 200 {
		e.t.Fatalf("login %s failed: %d %v", username, code, result["error"])
	}
}

func (e *testEnv) userID() string {
	e.t.Helper()
	_, me := e.get("/api/me")
	return me["id"].(string)
}

// createBotForUser creates a mock bot owned by the current user.
func (e *testEnv) createBotForUser(name string) *database.Bot {
	e.t.Helper()
	uid := e.userID()
	b, err := e.db.CreateBot(uid, name, "mock", mockProvider.Credentials())
	if err != nil {
		e.t.Fatalf("createBot: %v", err)
	}
	return b
}

// ==================== WebSocket helpers ====================

func (e *testEnv) connectWS(t *testing.T, apiKey string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + e.srv.URL[4:] + "/api/v1/channels/connect?key=" + apiKey
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	return ws
}

func readWS(t *testing.T, ws *websocket.Conn) map[string]any {
	t.Helper()
	return readWSTimeout(t, ws, 2*time.Second)
}

func readWSTimeout(t *testing.T, ws *websocket.Conn, d time.Duration) map[string]any {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(d))
	_, msg, err := ws.ReadMessage()
	ws.SetReadDeadline(time.Time{})
	if err != nil {
		return nil
	}
	var m map[string]any
	json.Unmarshal(msg, &m)
	return m
}

func drainWS(t *testing.T, ws *websocket.Conn) {
	t.Helper()
	for readWSTimeout(t, ws, 300*time.Millisecond) != nil {
	}
}

func assertCode(t *testing.T, label string, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %d, want %d", label, got, want)
	}
}

// httpGet/httpPost are helpers for plain (no cookie jar) requests that handle errors.
func httpGet(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.DefaultClient.Do(mustReq(t, "GET", url, nil))
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func httpGetWithHeader(t *testing.T, url, header, value string) *http.Response {
	t.Helper()
	req := mustReq(t, "GET", url, nil)
	req.Header.Set(header, value)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	return resp
}

func httpPost(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	data, _ := json.Marshal(body)
	req := mustReq(t, "POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func httpPostMultipart(t *testing.T, url, contentType string, body []byte) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST multipart %s: %v", url, err)
	}
	return resp
}

func mustReq(t *testing.T, method, url string, body *bytes.Reader) *http.Request {
	t.Helper()
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// ==================== Auth tests ====================

func TestRegisterAndLogin(t *testing.T) {
	env := setup(t)
	defer env.close()

	// First user → admin
	env.register("admin", "password123")
	code, me := env.get("/api/me")
	assertCode(t, "GET /me", code, 200)
	if me["role"] != "admin" {
		t.Errorf("first user role = %v, want admin", me["role"])
	}

	// Logout
	env.post("/api/auth/logout", nil)
	code, _ = env.get("/api/me")
	assertCode(t, "after logout", code, 401)

	// Login
	env.login("admin", "password123")
	code, _ = env.get("/api/me")
	assertCode(t, "after login", code, 200)

	// Wrong password
	env.post("/api/auth/logout", nil)
	code, _ = env.postCode("/api/auth/login", map[string]string{"username": "admin", "password": "wrong"})
	assertCode(t, "wrong password", code, 401)

	// Second user → member
	env.register("member1", "password123")
	_, me = env.get("/api/me")
	if me["role"] != "member" {
		t.Errorf("second user role = %v, want member", me["role"])
	}
}

func TestRegisterValidation(t *testing.T) {
	env := setup(t)
	defer env.close()

	// Empty username
	code, _ := env.postCode("/api/auth/register", map[string]string{"username": "", "password": "password123"})
	assertCode(t, "empty username", code, 400)

	// Short password
	code, _ = env.postCode("/api/auth/register", map[string]string{"username": "u", "password": "short"})
	assertCode(t, "short password", code, 400)

	// Duplicate username
	env.register("taken", "password123")
	env.post("/api/auth/logout", nil)
	code, _ = env.postCode("/api/auth/register", map[string]string{"username": "taken", "password": "password123"})
	assertCode(t, "duplicate username", code, 409)
}

func TestProfileUpdate(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("profileuser", "password123")

	code, _ := env.put("/api/me/profile", map[string]string{
		"display_name": "New Name",
		"email":        "test@example.com",
	})
	assertCode(t, "update profile", code, 200)

	_, me := env.get("/api/me")
	if me["display_name"] != "New Name" {
		t.Errorf("display_name = %v", me["display_name"])
	}
}

func TestPasswordChange(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("pwuser", "oldpass123")

	// Change password
	code, _ := env.put("/api/me/password", map[string]string{
		"old_password": "oldpass123",
		"new_password": "newpass123",
	})
	assertCode(t, "change password", code, 200)

	// Old password should fail
	env.post("/api/auth/logout", nil)
	code, _ = env.postCode("/api/auth/login", map[string]string{"username": "pwuser", "password": "oldpass123"})
	assertCode(t, "old password", code, 401)

	// New password should work
	env.login("pwuser", "newpass123")

	// Wrong old password
	code, _ = env.put("/api/me/password", map[string]string{
		"old_password": "wrongold",
		"new_password": "another123",
	})
	assertCode(t, "wrong old password", code, 401)
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	env := setup(t)
	defer env.close()

	paths := []string{"/api/me", "/api/bots", "/api/bots/stats"}
	for _, p := range paths {
		code, _ := env.get(p)
		assertCode(t, "unauth GET "+p, code, 401)
	}
}

// ==================== OAuth providers ====================

func TestOAuthProviders(t *testing.T) {
	env := setup(t)
	defer env.close()

	code, result := env.get("/api/auth/oauth/providers")
	assertCode(t, "GET providers", code, 200)
	// No providers configured → empty list
	providers := result["providers"].([]any)
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

func TestOAuthRedirectUnknownProvider(t *testing.T) {
	env := setup(t)
	defer env.close()

	// Don't follow redirects
	env.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, _ := env.client.Get(env.srv.URL + "/api/auth/oauth/unknown")
	assertCode(t, "unknown provider", resp.StatusCode, 400)
}

func TestLinkedAccounts(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("oauthuser", "password123")

	// List linked accounts (should be empty)
	code, accounts := env.getList("/api/me/linked-accounts")
	assertCode(t, "list accounts", code, 200)
	if accounts != nil && len(accounts) > 0 {
		t.Errorf("expected 0 linked accounts, got %d", len(accounts))
	}

	// Unlink non-existent
	code, _ = env.del("/api/me/linked-accounts/github")
	assertCode(t, "unlink non-existent", code, 404)
}

// ==================== Bot CRUD ====================

func TestBotCRUD(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("botowner", "password123")
	botObj := env.createBotForUser("TestBot")

	// List bots
	code, bots := env.getList("/api/bots")
	assertCode(t, "list bots", code, 200)
	if len(bots) != 1 {
		t.Fatalf("want 1 bot, got %d", len(bots))
	}

	// Rename bot
	code, _ = env.put("/api/bots/"+botObj.ID, map[string]string{"name": "Renamed"})
	assertCode(t, "rename bot", code, 200)

	// Verify rename
	code, bots = env.getList("/api/bots")
	b := bots[0].(map[string]any)
	if b["name"] != "Renamed" {
		t.Errorf("name after rename = %v", b["name"])
	}

	// Reconnect
	code, _ = env.postCode("/api/bots/"+botObj.ID+"/reconnect", nil)
	assertCode(t, "reconnect", code, 200)

	// Delete bot
	code, _ = env.del("/api/bots/" + botObj.ID)
	assertCode(t, "delete bot", code, 200)

	code, bots = env.getList("/api/bots")
	if len(bots) != 0 {
		t.Errorf("bots after delete = %d", len(bots))
	}
}

func TestBotOwnershipIsolation(t *testing.T) {
	env := setup(t)
	defer env.close()

	// User1 creates bot
	env.register("user1", "password123")
	botObj := env.createBotForUser("User1Bot")

	// Switch to user2
	env.post("/api/auth/logout", nil)
	env.register("user2", "password123")

	// User2 can't see user1's bots
	_, bots := env.getList("/api/bots")
	if len(bots) != 0 {
		t.Error("user2 should not see user1's bots")
	}

	// User2 can't rename user1's bot
	code, _ := env.put("/api/bots/"+botObj.ID, map[string]string{"name": "hacked"})
	assertCode(t, "rename other's bot", code, 404)

	// User2 can't delete user1's bot
	code, _ = env.del("/api/bots/" + botObj.ID)
	assertCode(t, "delete other's bot", code, 404)

	// User2 can't reconnect user1's bot
	code, _ = env.postCode("/api/bots/"+botObj.ID+"/reconnect", nil)
	assertCode(t, "reconnect other's bot", code, 404)
}

// ==================== Channel CRUD ====================

func TestChannelCRUD(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("chowner", "password123")
	botObj := env.createBotForUser("Bot1")

	// Create channel
	code, ch := env.postCode("/api/bots/"+botObj.ID+"/channels", map[string]string{
		"name": "通道1", "handle": "support",
	})
	assertCode(t, "create channel", code, 201)
	chID := ch["id"].(string)
	if ch["handle"] != "support" {
		t.Errorf("handle = %v", ch["handle"])
	}
	if ch["api_key"] == nil || ch["api_key"] == "" {
		t.Error("api_key should be generated")
	}

	// List channels
	code, chs := env.getList("/api/bots/" + botObj.ID + "/channels")
	assertCode(t, "list channels", code, 200)
	if len(chs) != 1 {
		t.Fatalf("want 1 channel, got %d", len(chs))
	}

	// Update channel
	code, _ = env.put("/api/bots/"+botObj.ID+"/channels/"+chID, map[string]any{
		"name": "新名称", "handle": "newhandle", "enabled": false,
	})
	assertCode(t, "update channel", code, 200)

	// Rotate key
	code, rotated := env.postCode("/api/bots/"+botObj.ID+"/channels/"+chID+"/rotate_key", nil)
	assertCode(t, "rotate key", code, 200)
	if rotated["api_key"] == nil || rotated["api_key"] == "" {
		t.Error("rotated key should be returned")
	}

	// Delete channel
	code, _ = env.del("/api/bots/" + botObj.ID + "/channels/" + chID)
	assertCode(t, "delete channel", code, 200)

	code, chs = env.getList("/api/bots/" + botObj.ID + "/channels")
	if len(chs) != 0 {
		t.Errorf("channels after delete = %d", len(chs))
	}
}

func TestChannelValidation(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("chval", "password123")
	botObj := env.createBotForUser("Bot1")

	// Missing name
	code, _ := env.postCode("/api/bots/"+botObj.ID+"/channels", map[string]string{})
	assertCode(t, "missing name", code, 400)

	// Non-existent bot
	code, _ = env.postCode("/api/bots/nonexistent/channels", map[string]string{"name": "test"})
	assertCode(t, "bad bot_id", code, 404)
}

func TestChannelOwnershipIsolation(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("user1", "password123")
	botObj := env.createBotForUser("Bot1")
	ch, _ := env.db.CreateChannel(botObj.ID, "Chan1", "c1", nil, nil)

	env.post("/api/auth/logout", nil)
	env.register("user2", "password123")

	// User2 can't update/delete/rotate user1's channel
	code, _ := env.put("/api/bots/"+botObj.ID+"/channels/"+ch.ID, map[string]any{"name": "hacked"})
	assertCode(t, "update other's channel", code, 404)

	code, _ = env.del("/api/bots/" + botObj.ID + "/channels/" + ch.ID)
	assertCode(t, "delete other's channel", code, 404)

	code, _ = env.postCode("/api/bots/"+botObj.ID+"/channels/"+ch.ID+"/rotate_key", nil)
	assertCode(t, "rotate other's key", code, 404)

	// User2 can't create channel on user1's bot
	code, _ = env.postCode("/api/bots/"+botObj.ID+"/channels", map[string]string{"name": "test"})
	assertCode(t, "create on other's bot", code, 404)
}

// ==================== Messages ====================

func TestMessages(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("msguser", "password123")
	botObj := env.createBotForUser("Bot1")

	// No messages yet
	code, result := env.get(fmt.Sprintf("/api/bots/%s/messages", botObj.ID))
	assertCode(t, "empty messages", code, 200)

	// Save some messages
	payload, _ := json.Marshal(map[string]string{"content": "hello"})
	for i := 0; i < 3; i++ {
		env.db.SaveMessage(&database.Message{
			BotID: botObj.ID, Direction: "inbound", Sender: "user@wechat",
			MsgType: "text", Payload: payload,
		})
	}

	code, result = env.get(fmt.Sprintf("/api/bots/%s/messages", botObj.ID))
	assertCode(t, "list messages", code, 200)
	msgs := result["messages"].([]any)
	if len(msgs) != 3 {
		t.Errorf("want 3 messages, got %d", len(msgs))
	}
	if result["has_more"] != false {
		t.Errorf("has_more should be false")
	}
}

func TestMessageOwnershipIsolation(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("user1", "password123")
	botObj := env.createBotForUser("User1Bot")

	env.post("/api/auth/logout", nil)
	env.register("user2", "password123")

	code, _ := env.get(fmt.Sprintf("/api/bots/%s/messages", botObj.ID))
	assertCode(t, "user2 reading user1 messages", code, 404)
}

// ==================== Stats ====================

func TestStats(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("statsuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.db.CreateChannel(botObj.ID, "Ch1", "", nil, nil)

	code, stats := env.get("/api/bots/stats")
	assertCode(t, "stats", code, 200)
	if stats["total_bots"] != float64(1) {
		t.Errorf("total_bots = %v", stats["total_bots"])
	}
	if stats["total_channels"] != float64(1) {
		t.Errorf("total_channels = %v", stats["total_channels"])
	}
}

// ==================== Bot contacts ====================

func TestBotContacts(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("contactuser", "password123")
	botObj := env.createBotForUser("Bot1")

	// Save inbound messages from different senders
	payload, _ := json.Marshal(map[string]string{"content": "hi"})
	for _, sender := range []string{"alice@wechat", "bob@wechat", "alice@wechat"} {
		env.db.SaveMessage(&database.Message{
			BotID: botObj.ID, Direction: "inbound", Sender: sender,
			MsgType: "text", Payload: payload,
		})
	}

	code, contacts := env.getList(fmt.Sprintf("/api/bots/%s/contacts", botObj.ID))
	assertCode(t, "contacts", code, 200)
	if len(contacts) != 2 {
		t.Errorf("want 2 contacts, got %d", len(contacts))
	}
}

func TestBotContactsOwnership(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("user1", "password123")
	botObj := env.createBotForUser("Bot1")

	env.post("/api/auth/logout", nil)
	env.register("user2", "password123")

	code, _ := env.get(fmt.Sprintf("/api/bots/%s/contacts", botObj.ID))
	assertCode(t, "contacts other's bot", code, 404)
}

// ==================== Bot send ====================

func TestBotSend(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("senduser", "password123")
	botObj := env.createBotForUser("Bot1")

	// Start bot
	env.mgr.StartBot(context.Background(), botObj)

	// Send
	code, result := env.postCode("/api/bots/"+botObj.ID+"/send", map[string]string{
		"text": "hello from api",
	})
	assertCode(t, "send", code, 200)
	if result["client_id"] == nil {
		t.Error("expected client_id in response")
	}

	// Verify mock provider received it
	inst, _ := env.mgr.GetInstance(botObj.ID)
	sent := inst.Provider.(*mockProvider.Provider).SentMessages()
	if len(sent) != 1 || sent[0].Text != "hello from api" {
		t.Errorf("sent = %+v", sent)
	}

	// Send without text
	code, _ = env.postCode("/api/bots/"+botObj.ID+"/send", map[string]string{})
	assertCode(t, "send no text", code, 400)

	// Send to disconnected bot
	env.mgr.StopBot(botObj.ID)
	code, _ = env.postCode("/api/bots/"+botObj.ID+"/send", map[string]string{"text": "fail"})
	assertCode(t, "send disconnected", code, 503)
}

func TestBotSendMedia(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("mediasend", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Send image via multipart
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "test.jpg")
	part.Write([]byte("fake-jpeg-data"))
	writer.WriteField("text", "看看这张图")
	writer.Close()

	req, _ := http.NewRequest("POST", env.srv.URL+"/api/bots/"+botObj.ID+"/send", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	// Copy cookies for auth
	for _, c := range env.client.Jar.Cookies(req.URL) {
		req.AddCookie(c)
	}
	resp, err := env.client.Do(req)
	if err != nil {
		t.Fatalf("send media: %v", err)
	}
	defer resp.Body.Close()
	assertCode(t, "send media", resp.StatusCode, 200)

	// Verify mock provider received media
	inst, _ := env.mgr.GetInstance(botObj.ID)
	sent := inst.Provider.(*mockProvider.Provider).SentMessages()
	var mediaSent *provider.OutboundMessage
	for i := range sent {
		if sent[i].FileName != "" {
			mediaSent = &sent[i]
			break
		}
	}
	if mediaSent == nil {
		t.Fatal("no media message sent to provider")
	}
	if mediaSent.FileName != "test.jpg" {
		t.Errorf("filename = %q, want test.jpg", mediaSent.FileName)
	}
	if string(mediaSent.Data) != "fake-jpeg-data" {
		t.Errorf("data = %q", string(mediaSent.Data))
	}
	if mediaSent.Text != "看看这张图" {
		t.Errorf("text = %q, want caption", mediaSent.Text)
	}

	// Verify message saved in DB
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range msgs {
		if m.Direction == "outbound" && m.MsgType == "image" {
			found = true
		}
	}
	if !found {
		t.Error("outbound image message not saved in DB")
	}
}

func TestChannelSendMedia(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("chsend", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "SendChan", "", nil, nil)

	// Send file via channel API (multipart with API key)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("file", "document.pdf")
	part.Write([]byte("fake-pdf-data"))
	writer.Close()

	resp := httpPostMultipart(t, env.srv.URL+"/api/v1/channels/send?key="+ch.APIKey, writer.FormDataContentType(), body.Bytes())
	defer resp.Body.Close()
	assertCode(t, "channel send media", resp.StatusCode, 200)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	sent := inst.Provider.(*mockProvider.Provider).SentMessages()
	var fileSent *provider.OutboundMessage
	for i := range sent {
		if sent[i].FileName != "" {
			fileSent = &sent[i]
			break
		}
	}
	if fileSent == nil {
		t.Fatal("no file message sent via channel")
	}
	if fileSent.FileName != "document.pdf" {
		t.Errorf("filename = %q", fileSent.FileName)
	}
}

// ==================== Admin user management ====================

func TestAdminUserManagement(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("admin", "password123") // first user = admin
	adminID := env.userID()

	// Create user via admin API
	code, created := env.postCode("/api/admin/users", map[string]string{
		"username": "newuser", "password": "password123", "role": "member",
	})
	assertCode(t, "create user", code, 201)
	newID := created["id"].(string)

	// List users
	code, users := env.getList("/api/admin/users")
	assertCode(t, "list users", code, 200)
	if len(users) != 2 {
		t.Errorf("want 2 users, got %d", len(users))
	}

	// Update role
	code, _ = env.put("/api/admin/users/"+newID+"/role", map[string]string{"role": "admin"})
	assertCode(t, "update role", code, 200)

	// Can't demote self
	code, _ = env.put("/api/admin/users/"+adminID+"/role", map[string]string{"role": "member"})
	assertCode(t, "self demote", code, 400)

	// Update status
	code, _ = env.put("/api/admin/users/"+newID+"/status", map[string]string{"status": "disabled"})
	assertCode(t, "disable user", code, 200)

	// Can't disable self
	code, _ = env.put("/api/admin/users/"+adminID+"/status", map[string]string{"status": "disabled"})
	assertCode(t, "self disable", code, 400)

	// Reset password
	code, _ = env.put("/api/admin/users/"+newID+"/password", map[string]string{"password": "newpass123"})
	assertCode(t, "reset password", code, 200)

	// Delete user
	code, _ = env.del("/api/admin/users/" + newID)
	assertCode(t, "delete user", code, 200)

	// Can't delete self
	code, _ = env.del("/api/admin/users/" + adminID)
	assertCode(t, "self delete", code, 400)
}

func TestAdminRequiresAdminRole(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("admin", "password123")
	env.post("/api/auth/logout", nil)
	env.register("member", "password123")

	// Member can't access admin APIs
	code, _ := env.getList("/api/admin/users")
	assertCode(t, "member list users", code, 403)

	code, _ = env.postCode("/api/admin/users", map[string]string{"username": "x", "password": "password123"})
	assertCode(t, "member create user", code, 403)
}

// ==================== Admin OAuth config ====================

func TestAdminOAuthConfig(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("admin", "password123")

	// Get config (empty)
	code, config := env.get("/api/admin/config/oauth")
	assertCode(t, "get config", code, 200)

	// Set GitHub config
	code, _ = env.put("/api/admin/config/oauth/github", map[string]string{
		"client_id": "test-id", "client_secret": "test-secret",
	})
	assertCode(t, "set github", code, 200)

	// Verify it's set
	code, config = env.get("/api/admin/config/oauth")
	assertCode(t, "get after set", code, 200)
	gh := config["github"].(map[string]any)
	if gh["client_id"] != "test-id" {
		t.Errorf("client_id = %v", gh["client_id"])
	}
	if gh["source"] != "db" {
		t.Errorf("source = %v, want db", gh["source"])
	}
	// Secret should be masked
	secret := gh["client_secret"].(string)
	if secret == "test-secret" {
		t.Error("secret should be masked")
	}

	// OAuth providers should now include github
	code, providers := env.get("/api/auth/oauth/providers")
	assertCode(t, "providers after config", code, 200)
	pList := providers["providers"].([]any)
	found := false
	for _, p := range pList {
		if p == "github" {
			found = true
		}
	}
	if !found {
		t.Error("github should be in providers list after config")
	}

	// Delete config
	code, _ = env.del("/api/admin/config/oauth/github")
	assertCode(t, "delete github config", code, 200)

	// Unknown provider
	code, _ = env.put("/api/admin/config/oauth/unknown", map[string]string{"client_id": "x"})
	assertCode(t, "unknown provider", code, 400)
}

func TestAdminOAuthConfigRequiresAdmin(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("admin", "password123")
	env.post("/api/auth/logout", nil)
	env.register("member", "password123")

	code, _ := env.get("/api/admin/config/oauth")
	assertCode(t, "member get config", code, 403)

	code, _ = env.put("/api/admin/config/oauth/github", map[string]string{"client_id": "x"})
	assertCode(t, "member set config", code, 403)
}

// ==================== WebSocket ====================

func TestWebSocketInitAndPing(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("wsuser", "password123")
	botObj := env.createBotForUser("Bot1")
	ch, _ := env.db.CreateChannel(botObj.ID, "WsChan", "", nil, nil)

	ws := env.connectWS(t, ch.APIKey)
	defer ws.Close()

	// Should receive init message
	init := readWS(t, ws)
	if init == nil || init["type"] != "init" {
		t.Fatalf("expected init message, got %v", init)
	}
	data := init["data"].(map[string]any)
	if data["channel_id"] != ch.ID {
		t.Errorf("channel_id = %v, want %v", data["channel_id"], ch.ID)
	}

	// Ping/pong
	ws.WriteJSON(map[string]string{"type": "ping"})
	pong := readWS(t, ws)
	if pong == nil || pong["type"] != "pong" {
		t.Errorf("expected pong, got %v", pong)
	}
}

func TestWebSocketInvalidKey(t *testing.T) {
	env := setup(t)
	defer env.close()

	wsURL := "ws" + env.srv.URL[4:] + "/api/v1/channels/connect?key=invalid"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Error("should fail with invalid key")
	}
	if resp != nil && resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWebSocketNoKey(t *testing.T) {
	env := setup(t)
	defer env.close()

	wsURL := "ws" + env.srv.URL[4:] + "/api/v1/channels/connect"
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Error("should fail without key")
	}
	if resp != nil && resp.StatusCode != 401 {
		t.Errorf("status = %d, want 401", resp.StatusCode)
	}
}

func TestWebSocketSendText(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("wssend", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "SendChan", "", nil, nil)

	ws := env.connectWS(t, ch.APIKey)
	defer ws.Close()
	readWS(t, ws) // init

	// Send text
	ws.WriteJSON(map[string]any{
		"type":   "send_text",
		"req_id": "r1",
		"data":   map[string]string{"text": "hello via ws"},
	})

	ack := readWS(t, ws)
	if ack == nil || ack["type"] != "send_ack" {
		t.Fatalf("expected send_ack, got %v", ack)
	}
	ackData := ack["data"].(map[string]any)
	if ackData["success"] != true {
		t.Errorf("ack success = %v, error = %v", ackData["success"], ackData["error"])
	}

	// Verify mock provider received
	inst, _ := env.mgr.GetInstance(botObj.ID)
	sent := inst.Provider.(*mockProvider.Provider).SentMessages()
	if len(sent) != 1 || sent[0].Text != "hello via ws" {
		t.Errorf("sent = %+v", sent)
	}
}

// ==================== @Mention routing ====================

func TestMentionRouting(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("mentionuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch1, _ := env.db.CreateChannel(botObj.ID, "支持", "support", nil, nil)
	ch2, _ := env.db.CreateChannel(botObj.ID, "销售", "sales", nil, nil)
	chAll, _ := env.db.CreateChannel(botObj.ID, "全部", "", nil, nil)

	ws1 := env.connectWS(t, ch1.APIKey)
	defer ws1.Close()
	ws2 := env.connectWS(t, ch2.APIKey)
	defer ws2.Close()
	wsAll := env.connectWS(t, chAll.APIKey)
	defer wsAll.Close()
	readWS(t, ws1)
	readWS(t, ws2)
	readWS(t, wsAll)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// @support → ch1 (handle match) + chAll (no handle, receives all)
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "1", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "@support help"}},
	})
	if readWSTimeout(t, ws1, 2*time.Second) == nil {
		t.Error("ch1 should receive @support")
	}
	if readWSTimeout(t, ws2, 300*time.Millisecond) != nil {
		t.Error("ch2 should NOT receive @support")
	}
	if readWSTimeout(t, wsAll, 2*time.Second) == nil {
		t.Error("chAll (no handle) should receive ALL messages")
	}

	// No mention → only chAll (no handle channels receive all)
	wsAll.Close()
	wsAll = env.connectWS(t, chAll.APIKey)
	defer wsAll.Close()
	readWS(t, wsAll)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "2", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "普通消息"}},
	})
	if readWSTimeout(t, ws1, 300*time.Millisecond) != nil {
		t.Error("ch1 (has handle) should NOT receive non-mention")
	}
	if readWSTimeout(t, ws2, 300*time.Millisecond) != nil {
		t.Error("ch2 (has handle) should NOT receive non-mention")
	}
	if readWSTimeout(t, wsAll, 2*time.Second) == nil {
		t.Error("chAll (no handle) should receive non-mention")
	}

	// @unknown → only chAll (no handle channels still receive)
	ws1.Close()
	ws2.Close()
	wsAll.Close()
	ws1 = env.connectWS(t, ch1.APIKey)
	defer ws1.Close()
	ws2 = env.connectWS(t, ch2.APIKey)
	defer ws2.Close()
	wsAll = env.connectWS(t, chAll.APIKey)
	defer wsAll.Close()
	drainWS(t, ws1)
	drainWS(t, ws2)
	drainWS(t, wsAll)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "3", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "@nobody test"}},
	})
	if readWSTimeout(t, ws1, 300*time.Millisecond) != nil {
		t.Error("ch1 should NOT receive @nobody")
	}
	if readWSTimeout(t, ws2, 300*time.Millisecond) != nil {
		t.Error("ch2 should NOT receive @nobody")
	}
	// chAll should receive because it has no handle (receives all)
	// Use longer timeout and drain first to avoid stale messages
	time.Sleep(200 * time.Millisecond)
	msgs, _ := env.db.ListChannelMessages(chAll.ID, "u@wx", 10)
	foundNobody := false
	for _, m := range msgs {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "@nobody test" {
			foundNobody = true
		}
	}
	if !foundNobody {
		t.Error("chAll (no handle) should still receive @nobody in DB")
	}
}

// ==================== Inbound storage with channel_id ====================

func TestInboundStoredWithChannelID(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("storeuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "Default", "", nil, nil)
	ws := env.connectWS(t, ch.APIKey)
	defer ws.Close()
	readWS(t, ws) // init

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// Send inbound message (no @mention, matches first channel)
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "100", Sender: "alice@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "hello"}},
	})
	readWSTimeout(t, ws, 2*time.Second)

	// Verify inbound stored with channel_id
	msgs, _ := env.db.ListChannelMessages(ch.ID, "alice@wx", 10)
	if len(msgs) != 1 {
		t.Fatalf("want 1 message in channel, got %d", len(msgs))
	}
	if msgs[0].Direction != "inbound" {
		t.Errorf("direction = %q, want inbound", msgs[0].Direction)
	}
	if msgs[0].ChannelID == nil || *msgs[0].ChannelID != ch.ID {
		t.Errorf("channel_id = %v, want %s", msgs[0].ChannelID, ch.ID)
	}
}

func TestInboundNoMatchStoredWithoutChannelID(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("nomatch", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Channel with user filter that won't match
	filter := &database.FilterRule{UserIDs: []string{"specific@wx"}}
	env.db.CreateChannel(botObj.ID, "Filtered", "", filter, nil)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// Send from non-matching user
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "200", Sender: "other@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "hello"}},
	})

	time.Sleep(100 * time.Millisecond)

	// Should be stored without channel_id
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range msgs {
		if m.Sender == "other@wx" {
			found = true
			if m.ChannelID != nil {
				t.Error("unmatched inbound should have nil channel_id")
			}
		}
	}
	if !found {
		t.Error("unmatched inbound should still be stored")
	}
}

func TestRawMessageStored(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("rawuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "RawChan", "", nil, nil)
	ws := env.connectWS(t, ch.APIKey)
	defer ws.Close()
	readWS(t, ws)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "raw-1", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "hello raw"}},
	})
	readWSTimeout(t, ws, 2*time.Second)

	// Check bot-level message has raw
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	if len(msgs) == 0 {
		t.Fatal("no messages")
	}
	msg := msgs[0]
	if msg.Raw == nil {
		t.Fatal("raw is nil")
	}

	var raw map[string]any
	json.Unmarshal(*msg.Raw, &raw)

	// Mock sets _mock: true
	if raw["_mock"] != true {
		t.Errorf("raw._mock = %v, want true", raw["_mock"])
	}
	if raw["from_user_id"] != "u@wx" {
		t.Errorf("raw.from_user_id = %v", raw["from_user_id"])
	}

	// Check channel-level copy also has raw
	chMsgs, _ := env.db.ListChannelMessages(ch.ID, "u@wx", 10)
	if len(chMsgs) == 0 {
		t.Fatal("no channel messages")
	}
	if chMsgs[0].Raw == nil {
		t.Error("channel copy raw is nil")
	}
}

func TestRawMessageWithCustomData(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("rawcustom", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// Send with explicit raw (simulating real iLink response)
	customRaw := json.RawMessage(`{"message_id":12345,"item_list":[{"type":3,"voice_item":{"encode_type":6,"sample_rate":24000}}],"_server":"ilink"}`)
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "raw-2", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "voice", Text: "你好"}},
		Raw:   customRaw,
	})

	time.Sleep(200 * time.Millisecond)

	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	if len(msgs) == 0 {
		t.Fatal("no messages")
	}
	if msgs[0].Raw == nil {
		t.Fatal("raw is nil")
	}

	var raw map[string]any
	json.Unmarshal(*msgs[0].Raw, &raw)

	// Should preserve the custom raw, not auto-generate
	if raw["_server"] != "ilink" {
		t.Errorf("raw._server = %v, want ilink", raw["_server"])
	}
	if raw["message_id"] != float64(12345) {
		t.Errorf("raw.message_id = %v", raw["message_id"])
	}

	// Verify voice encode_type is preserved in raw
	items := raw["item_list"].([]any)
	voice := items[0].(map[string]any)["voice_item"].(map[string]any)
	if voice["encode_type"] != float64(6) {
		t.Errorf("encode_type = %v, want 6", voice["encode_type"])
	}
}

func TestMentionRoutesFirstOnly(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("firstonly", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Two channels, second one also has handle "support"
	ch1, _ := env.db.CreateChannel(botObj.ID, "Support1", "support", nil, nil)
	ch2, _ := env.db.CreateChannel(botObj.ID, "Support2", "support", nil, nil)

	ws1 := env.connectWS(t, ch1.APIKey)
	defer ws1.Close()
	ws2 := env.connectWS(t, ch2.APIKey)
	defer ws2.Close()
	readWS(t, ws1)
	readWS(t, ws2)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "300", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "@support help"}},
	})

	// Only first channel receives
	if readWSTimeout(t, ws1, 2*time.Second) == nil {
		t.Error("ch1 (first match) should receive")
	}
	if readWSTimeout(t, ws2, 300*time.Millisecond) != nil {
		t.Error("ch2 (second match) should NOT receive")
	}

	// Stored with ch1's channel_id
	msgs, _ := env.db.ListChannelMessages(ch1.ID, "u@wx", 10)
	if len(msgs) != 1 {
		t.Errorf("ch1 should have 1 message, got %d", len(msgs))
	}
	msgs2, _ := env.db.ListChannelMessages(ch2.ID, "u@wx", 10)
	if len(msgs2) != 0 {
		t.Errorf("ch2 should have 0 messages, got %d", len(msgs2))
	}
}

func TestChannelContextFullIsolation(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("isol", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch1, _ := env.db.CreateChannel(botObj.ID, "Support", "support", nil, nil)
	ch2, _ := env.db.CreateChannel(botObj.ID, "Sales", "sales", nil, nil)

	ws1 := env.connectWS(t, ch1.APIKey)
	defer ws1.Close()
	ws2 := env.connectWS(t, ch2.APIKey)
	defer ws2.Close()
	readWS(t, ws1)
	readWS(t, ws2)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// @support → ch1
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "400", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "@support help me"}},
	})
	readWSTimeout(t, ws1, 2*time.Second)

	// @sales → ch2
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "401", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "@sales price?"}},
	})
	readWSTimeout(t, ws2, 2*time.Second)

	// Add outbound in each channel
	ch1ID := ch1.ID
	ch2ID := ch2.ID
	r1, _ := json.Marshal(map[string]string{"content": "support reply"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch1ID, Direction: "outbound",
		Recipient: "u@wx", MsgType: "text", Payload: r1,
	})
	r2, _ := json.Marshal(map[string]string{"content": "sales reply"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch2ID, Direction: "outbound",
		Recipient: "u@wx", MsgType: "text", Payload: r2,
	})

	// ch1 context: 1 inbound ("@support help me") + 1 outbound ("support reply") = 2
	msgs1, _ := env.db.ListChannelMessages(ch1.ID, "u@wx", 50)
	if len(msgs1) != 2 {
		t.Errorf("ch1: want 2, got %d", len(msgs1))
	}
	for _, m := range msgs1 {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "sales reply" || p.Content == "@sales price?" {
			t.Errorf("ch1 leaked ch2 content: %q", p.Content)
		}
	}

	// ch2 context: 1 inbound ("@sales price?") + 1 outbound ("sales reply") = 2
	msgs2, _ := env.db.ListChannelMessages(ch2.ID, "u@wx", 50)
	if len(msgs2) != 2 {
		t.Errorf("ch2: want 2, got %d", len(msgs2))
	}
	for _, m := range msgs2 {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "support reply" || p.Content == "@support help me" {
			t.Errorf("ch2 leaked ch1 content: %q", p.Content)
		}
	}
}

// ==================== Channel HTTP API ====================

func TestChannelHTTPStatus(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("httpuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "HttpChan", "", nil, nil)

	resp := httpGet(t, env.srv.URL+"/api/v1/channels/status?key="+ch.APIKey)
	defer resp.Body.Close()
	assertCode(t, "channel status", resp.StatusCode, 200)
	var status map[string]any
	json.NewDecoder(resp.Body).Decode(&status)
	if status["bot_status"] != "connected" {
		t.Errorf("bot_status = %v", status["bot_status"])
	}
	if status["channel_name"] != "HttpChan" {
		t.Errorf("channel_name = %v", status["channel_name"])
	}

	// No key
	resp2 := httpGet(t, env.srv.URL+"/api/v1/channels/status")
	assertCode(t, "status no key", resp2.StatusCode, 401)
	resp2.Body.Close()

	// Invalid key
	resp3 := httpGet(t, env.srv.URL+"/api/v1/channels/status?key=invalid")
	assertCode(t, "status invalid key", resp3.StatusCode, 401)
	resp3.Body.Close()
}

func TestChannelHTTPStatusWithHeader(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("headeruser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "HeaderChan", "", nil, nil)

	resp := httpGetWithHeader(t, env.srv.URL+"/api/v1/channels/status", "X-API-Key", ch.APIKey)
	defer resp.Body.Close()
	assertCode(t, "status via header", resp.StatusCode, 200)
}

func TestChannelHTTPMessages(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("msghttp", "password123")
	botObj := env.createBotForUser("Bot1")
	ch, _ := env.db.CreateChannel(botObj.ID, "MsgChan", "", nil, nil)

	payload, _ := json.Marshal(map[string]string{"content": "hello"})
	for i := 0; i < 5; i++ {
		env.db.SaveMessage(&database.Message{
			BotID: botObj.ID, Direction: "inbound", Sender: "u@wx",
			MsgType: "text", Payload: payload,
		})
	}

	// First page
	resp := httpGet(t, env.srv.URL+"/api/v1/channels/messages?key="+ch.APIKey+"&limit=3")
	defer resp.Body.Close()
	assertCode(t, "channel messages", resp.StatusCode, 200)
	var page1 map[string]any
	json.NewDecoder(resp.Body).Decode(&page1)
	msgs := page1["messages"].([]any)
	if len(msgs) != 3 {
		t.Fatalf("want 3 messages, got %d", len(msgs))
	}
	cursor := page1["next_cursor"].(string)
	if cursor == "" {
		t.Fatal("expected next_cursor for pagination")
	}

	// Second page using cursor
	resp2 := httpGet(t, env.srv.URL+"/api/v1/channels/messages?key="+ch.APIKey+"&cursor="+cursor+"&limit=3")
	defer resp2.Body.Close()
	var page2 map[string]any
	json.NewDecoder(resp2.Body).Decode(&page2)
	msgs2 := page2["messages"].([]any)
	if len(msgs2) != 2 {
		t.Errorf("want 2 remaining messages, got %d", len(msgs2))
	}
	// No more pages
	if page2["next_cursor"] != nil && page2["next_cursor"] != "" {
		t.Errorf("expected empty next_cursor, got %v", page2["next_cursor"])
	}

	// Invalid cursor
	resp3 := httpGet(t, env.srv.URL+"/api/v1/channels/messages?key="+ch.APIKey+"&cursor=bad!")
	assertCode(t, "invalid cursor", resp3.StatusCode, 400)
	resp3.Body.Close()
}

func TestChannelHTTPSend(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("sendhttp", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "SendChan", "", nil, nil)

	// Send message
	resp := httpPost(t, env.srv.URL+"/api/v1/channels/send?key="+ch.APIKey,
		map[string]string{"text": "hello via http"})
	defer resp.Body.Close()
	assertCode(t, "channel send", resp.StatusCode, 200)
	var result map[string]any
	json.NewDecoder(resp.Body).Decode(&result)
	if result["ok"] != true {
		t.Errorf("ok = %v", result["ok"])
	}

	// Verify mock provider received
	inst, _ := env.mgr.GetInstance(botObj.ID)
	sent := inst.Provider.(*mockProvider.Provider).SentMessages()
	if len(sent) != 1 || sent[0].Text != "hello via http" {
		t.Errorf("sent = %+v", sent)
	}

	// Verify message saved in DB with channel_id
	dbMsgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range dbMsgs {
		if m.Direction == "outbound" && m.ChannelID != nil && *m.ChannelID == ch.ID {
			found = true
		}
	}
	if !found {
		t.Error("outbound message not saved with channel_id")
	}

	// Send without text
	resp2 := httpPost(t, env.srv.URL+"/api/v1/channels/send?key="+ch.APIKey, map[string]string{})
	assertCode(t, "send no text", resp2.StatusCode, 400)
	resp2.Body.Close()

	// Invalid key
	resp3 := httpPost(t, env.srv.URL+"/api/v1/channels/send?key=invalid",
		map[string]string{"text": "x"})
	assertCode(t, "send invalid key", resp3.StatusCode, 401)
	resp3.Body.Close()

	// Bot disconnected
	env.mgr.StopBot(botObj.ID)
	resp4 := httpPost(t, env.srv.URL+"/api/v1/channels/send?key="+ch.APIKey,
		map[string]string{"text": "fail"})
	assertCode(t, "send bot disconnected", resp4.StatusCode, 503)
	resp4.Body.Close()
}

func TestChannelHTTPDisabledChannel(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("disuser", "password123")
	botObj := env.createBotForUser("Bot1")
	ch, _ := env.db.CreateChannel(botObj.ID, "DisChan", "", nil, nil)

	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig, &ch.WebhookConfig, false)

	resp := httpGet(t, env.srv.URL+"/api/v1/channels/status?key="+ch.APIKey)
	assertCode(t, "disabled channel", resp.StatusCode, 401)
	resp.Body.Close()
}

// ==================== Webhook sink ====================

func TestWebhookDelivery(t *testing.T) {
	env := setup(t)
	defer env.close()

	// Set up a webhook receiver
	var received []map[string]any
	var receivedHeaders http.Header
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		received = append(received, body)
		w.WriteHeader(200)
	}))
	defer hookSrv.Close()

	env.register("hookuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Create channel with webhook
	ch, _ := env.db.CreateChannel(botObj.ID, "HookChan", "", nil, nil)
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL, Auth: &database.WebhookAuth{Type: "bearer", Token: "test-token"}}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "500", Sender: "hook@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "webhook test"}},
	})

	// Wait for async webhook delivery
	time.Sleep(500 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("want 1 webhook delivery, got %d", len(received))
	}

	msg := received[0]
	if msg["event"] != "message" {
		t.Errorf("event = %v", msg["event"])
	}
	if msg["sender"] != "hook@wx" {
		t.Errorf("sender = %v", msg["sender"])
	}
	if msg["content"] != "webhook test" {
		t.Errorf("content = %v", msg["content"])
	}
	if msg["channel_id"] != ch.ID {
		t.Errorf("channel_id = %v, want %s", msg["channel_id"], ch.ID)
	}

	// Verify bearer auth header
	auth := receivedHeaders.Get("Authorization")
	if auth != "Bearer test-token" {
		t.Errorf("Authorization = %q, want Bearer test-token", auth)
	}
}

func TestWebhookHMACSignature(t *testing.T) {
	env := setup(t)
	defer env.close()

	var signature string
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature = r.Header.Get("X-Hub-Signature")
		w.WriteHeader(200)
	}))
	defer hookSrv.Close()

	env.register("hmacuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "HmacChan", "", nil, nil)
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL, Auth: &database.WebhookAuth{Type: "hmac", Secret: "my-secret"}}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "600", Sender: "hmac@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "signed"}},
	})

	time.Sleep(500 * time.Millisecond)

	if !strings.HasPrefix(signature, "sha256=") {
		t.Errorf("signature = %q, want sha256=...", signature)
	}
	if len(signature) != 7+64 { // "sha256=" + 64 hex chars
		t.Errorf("signature length = %d", len(signature))
	}
}

func TestWebhookWithScript(t *testing.T) {
	env := setup(t)
	defer env.close()

	var receivedBody string
	var receivedHeaders http.Header
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		b := new(bytes.Buffer)
		b.ReadFrom(r.Body)
		receivedBody = b.String()
		w.WriteHeader(200)
	}))
	defer hookSrv.Close()

	env.register("scriptuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "ScriptChan", "", nil, nil)

	// Script uses onRequest to modify, onResponse to reply
	script := `
function onRequest(ctx) {
  ctx.req.headers["X-Custom"] = "hello";
  ctx.req.body = JSON.stringify({text: ctx.msg.sender + ": " + ctx.msg.content});
}
`
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "800", Sender: "alice@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "script test"}},
	})

	time.Sleep(500 * time.Millisecond)

	if receivedBody == "" {
		t.Fatal("no webhook received")
	}

	var body map[string]any
	json.Unmarshal([]byte(receivedBody), &body)
	if body["text"] != "alice@wx: script test" {
		t.Errorf("body = %v", body)
	}
	if receivedHeaders.Get("X-Custom") != "hello" {
		t.Errorf("X-Custom = %q", receivedHeaders.Get("X-Custom"))
	}
}

func TestWebhookScriptSkip(t *testing.T) {
	env := setup(t)
	defer env.close()

	received := false
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(200)
	}))
	defer hookSrv.Close()

	env.register("skipuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "SkipChan", "", nil, nil)

	// Script skips non-text messages
	script := `
function onRequest(ctx) {
  if (ctx.msg.msg_type !== "text") skip();
}
`
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	// Text message → should deliver
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "900", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "hello"}},
	})
	time.Sleep(300 * time.Millisecond)
	if !received {
		t.Error("text message should trigger webhook")
	}

	// Image message → script returns null, should skip
	received = false
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "901", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "image"}},
	})
	time.Sleep(300 * time.Millisecond)
	if received {
		t.Error("image message should be skipped by script")
	}
}

func TestWebhookOnResponse(t *testing.T) {
	env := setup(t)
	defer env.close()

	// Webhook server returns {"answer": "42"}
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"answer": "42"}`))
	}))
	defer hookSrv.Close()

	env.register("respuser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "RespChan", "", nil, nil)
	script := `
function onResponse(ctx) {
  var data = JSON.parse(ctx.res.body);
  if (data.answer) reply(data.answer);
}
`
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "850", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "question"}},
	})

	time.Sleep(500 * time.Millisecond)

	// Verify bot sent reply "42" back to user
	sent := mock.SentMessages()
	found := false
	for _, m := range sent {
		if m.Text == "42" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected reply '42', sent = %+v", sent)
	}

	// Verify reply saved in DB
	msgs, _ := env.db.ListChannelMessages(ch.ID, "u@wx", 10)
	replyFound := false
	for _, m := range msgs {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "42" && m.Direction == "outbound" {
			replyFound = true
		}
	}
	if !replyFound {
		t.Error("reply not saved in DB")
	}
}

func TestWebhookAutoReplyWithoutScript(t *testing.T) {
	env := setup(t)
	defer env.close()

	// Server returns {"reply": "auto-reply"}
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"reply": "auto-reply"}`))
	}))
	defer hookSrv.Close()

	env.register("autouser", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	ch, _ := env.db.CreateChannel(botObj.ID, "AutoChan", "", nil, nil)
	// No script — auto-reply from {"reply": "..."} in response
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&database.WebhookConfig{URL: hookSrv.URL}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "860", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "hi"}},
	})

	time.Sleep(500 * time.Millisecond)

	sent := mock.SentMessages()
	found := false
	for _, m := range sent {
		if m.Text == "auto-reply" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected auto-reply, sent = %+v", sent)
	}
}

func TestWebhookNotTriggeredWithoutURL(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("nohook", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Channel without webhook
	ch, _ := env.db.CreateChannel(botObj.ID, "NoHook", "", nil, nil)
	ws := env.connectWS(t, ch.APIKey)
	defer ws.Close()
	readWS(t, ws)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)

	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "700", Sender: "u@wx", Timestamp: time.Now().UnixMilli(),
		Items: []provider.MessageItem{{Type: "text", Text: "no hook"}},
	})

	// WS should still receive
	if readWSTimeout(t, ws, 2*time.Second) == nil {
		t.Error("WS should still receive without webhook")
	}
}

// ==================== AI context isolation ====================

func TestAIContextIsolation(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("aiuser", "password123")
	botObj := env.createBotForUser("Bot1")
	ch1, _ := env.db.CreateChannel(botObj.ID, "Support", "support", nil, nil)
	ch2, _ := env.db.CreateChannel(botObj.ID, "Sales", "sales", nil, nil)

	sender := "user@wechat"
	ch1ID := ch1.ID
	ch2ID := ch2.ID

	// Simulate inbound routed to ch1
	p1, _ := json.Marshal(map[string]string{"content": "help me"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch1ID, Direction: "inbound",
		Sender: sender, MsgType: "text", Payload: p1,
	})

	// Simulate inbound routed to ch2
	p2, _ := json.Marshal(map[string]string{"content": "price?"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch2ID, Direction: "inbound",
		Sender: sender, MsgType: "text", Payload: p2,
	})

	// AI reply in ch1
	r1, _ := json.Marshal(map[string]string{"content": "support reply"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch1ID, Direction: "outbound",
		Recipient: sender, MsgType: "text", Payload: r1,
	})

	// AI reply in ch2
	r2, _ := json.Marshal(map[string]string{"content": "sales reply"})
	env.db.SaveMessage(&database.Message{
		BotID: botObj.ID, ChannelID: &ch2ID, Direction: "outbound",
		Recipient: sender, MsgType: "text", Payload: r2,
	})

	// ch1: 1 inbound + 1 outbound = 2
	msgs1, err := env.db.ListChannelMessages(ch1.ID, sender, 50)
	if err != nil {
		t.Fatalf("ch1: %v", err)
	}
	if len(msgs1) != 2 {
		t.Errorf("ch1: want 2, got %d", len(msgs1))
	}
	for _, m := range msgs1 {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "sales reply" || p.Content == "price?" {
			t.Errorf("ch1 should NOT contain ch2 content: %q", p.Content)
		}
	}

	// ch2: 1 inbound + 1 outbound = 2
	msgs2, err := env.db.ListChannelMessages(ch2.ID, sender, 50)
	if err != nil {
		t.Fatalf("ch2: %v", err)
	}
	if len(msgs2) != 2 {
		t.Errorf("ch2: want 2, got %d", len(msgs2))
	}
	for _, m := range msgs2 {
		var p struct{ Content string }
		json.Unmarshal(m.Payload, &p)
		if p.Content == "support reply" || p.Content == "help me" {
			t.Errorf("ch2 should NOT contain ch1 content: %q", p.Content)
		}
	}

	// Other sender: 0
	msgs3, _ := env.db.ListChannelMessages(ch1.ID, "other@wechat", 50)
	if len(msgs3) != 0 {
		t.Errorf("other sender: want 0, got %d", len(msgs3))
	}
}

// ==================== Media storage ====================

func TestMediaStorageAndProxy(t *testing.T) {
	// Requires MinIO running on localhost:19000
	store, err := storage.New(storage.Config{
		Endpoint:  "localhost:19000",
		AccessKey: "openilink",
		SecretKey: "openilink",
		Bucket:    "openilink-test",
		UseSSL:    false,
		PublicURL: "", // will be set after server starts
	})
	if err != nil {
		t.Skipf("skip: MinIO unavailable: %v", err)
	}

	db := testDB(t)
	cfg := &config.Config{RPOrigin: "http://localhost", RPID: "localhost", RPName: "Test", Secret: "test"}
	server := &api.Server{
		DB: db, SessionStore: auth.NewSessionStore(), Config: cfg,
		OAuthStates: api.SetupOAuth(cfg), Store: store,
	}
	hub := relay.NewHub(server.SetupUpstreamHandler())
	sinks := []sink.Sink{&sink.WS{Hub: hub}, &sink.AI{DB: db}, &sink.Webhook{DB: db}}
	mgr := bot.NewManager(db, hub, sinks, store, "http://localhost")
	server.BotManager = mgr
	server.Hub = hub
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	defer mgr.StopAll()
	defer db.Close()

	// Update storage public URL to test server
	// We need to put files with keys that resolve through the proxy
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Register and login
	data, _ := json.Marshal(map[string]string{"username": "mediauser", "password": "password123"})
	resp, _ := client.Post(ts.URL+"/api/auth/register", "application/json", bytes.NewReader(data))
	resp.Body.Close()

	// Get user ID
	resp, _ = client.Get(ts.URL + "/api/me")
	var me map[string]any
	json.NewDecoder(resp.Body).Decode(&me)
	resp.Body.Close()
	userID := me["id"].(string)

	// Create bot
	botObj, _ := db.CreateBot(userID, "MediaBot", "mock", mockProvider.Credentials())
	mgr.StartBot(context.Background(), botObj)
	ch, _ := db.CreateChannel(botObj.ID, "MediaChan", "", nil, nil)

	// Simulate inbound with image media
	inst, _ := mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockProvider.Provider)
	mock.SimulateInbound(provider.InboundMessage{
		ExternalID: "img-1",
		Sender:     "u@wx",
		Timestamp:  time.Now().UnixMilli(),
		Items: []provider.MessageItem{{
			Type: "image",
			Media: &provider.Media{
				EncryptQueryParam: "test-eqp",
				AESKey:            "test-aes",
				MediaType:         "image",
			},
		}},
	})

	// Message should be saved immediately
	time.Sleep(50 * time.Millisecond)
	msgs, _ := db.ListChannelMessages(ch.ID, "u@wx", 10)
	if len(msgs) == 0 {
		t.Fatal("no messages found")
	}
	var earlyPayload map[string]any
	json.Unmarshal(msgs[0].Payload, &earlyPayload)
	earlyStatus, _ := earlyPayload["media_status"].(string)
	t.Logf("early media_status = %s", earlyStatus)

	// Wait for async download to complete
	time.Sleep(500 * time.Millisecond)
	msgs, _ = db.ListChannelMessages(ch.ID, "u@wx", 10)
	var payload map[string]any
	json.Unmarshal(msgs[0].Payload, &payload)

	status, _ := payload["media_status"].(string)
	if status != "ready" {
		t.Fatalf("media_status = %q, want ready. payload: %v", status, payload)
	}

	mediaKey, ok := payload["media_key"].(string)
	if !ok || mediaKey == "" {
		t.Fatalf("media_key not in payload: %v", payload)
	}
	t.Logf("media_key = %s", mediaKey)

	// Verify key format: {bot_id}/{msg_id}/{index}.jpg
	if !strings.HasPrefix(mediaKey, botObj.ID) {
		t.Errorf("media_key should start with bot ID, got %s", mediaKey)
	}
	if !strings.HasSuffix(mediaKey, ".jpg") {
		t.Errorf("media_key should end with .jpg, got %s", mediaKey)
	}

	// Fetch via media proxy (with session cookie)
	mediaURL := ts.URL + "/api/v1/media/" + mediaKey
	resp, err = client.Get(mediaURL)
	if err != nil {
		t.Fatalf("fetch media: %v", err)
	}
	defer resp.Body.Close()
	assertCode(t, "media proxy", resp.StatusCode, 200)

	var body bytes.Buffer
	body.ReadFrom(resp.Body)
	if body.String() != "mock-media-data" {
		t.Errorf("media content = %q, want mock-media-data", body.String())
	}

	// Fetch without auth → 401
	plainResp := httpGet(t, mediaURL)
	assertCode(t, "media no auth", plainResp.StatusCode, 401)
	plainResp.Body.Close()

	t.Logf("Full media URL: %s", mediaURL)
}
