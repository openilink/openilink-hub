package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
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
	appdelivery "github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/provider/ilink/mockserver"
	"github.com/openilink/openilink-hub/internal/relay"
	"github.com/openilink/openilink-hub/internal/sink"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/postgres"
	"github.com/openilink/openilink-hub/internal/storage"
)

// ==================== Test infrastructure ====================

func testDB(t *testing.T) *postgres.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://openilink:openilink@localhost:15432/openilink_test?sslmode=disable"
	}
	// Pre-connect to reset schema if migrations were consolidated
	preDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Skipf("skip: database unavailable: %v", err)
	}
	// Drop goose version table and legacy tables so migrations re-run from scratch
	preDB.Exec("DROP TABLE IF EXISTS goose_db_version, schema_version, plugin_installs, plugin_versions, plugins CASCADE")
	preDB.Close()

	db, err := postgres.Open(dsn)
	if err != nil {
		t.Skipf("skip: database unavailable: %v", err)
	}
	for _, table := range []string{"app_event_logs", "app_api_logs", "app_installations", "apps", "plugin_installs", "plugin_versions", "plugins", "webhook_logs", "messages", "channels", "bots", "oauth_accounts", "sessions", "credentials", "users", "system_config"} {
		db.Exec("DELETE FROM " + table)
	}
	return db
}

type testEnv struct {
	t      *testing.T
	db     store.Store
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
		Store:        db,
		SessionStore: auth.NewSessionStore(),
		Config:       cfg,
		OAuthStates:  api.SetupOAuth(cfg),
	}

	hub := relay.NewHub(server.SetupUpstreamHandler())
	aiSink := &sink.AI{Store: db}
	mgr := bot.NewManager(db, hub, aiSink, nil, "http://localhost")
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
func (e *testEnv) createBotForUser(name string) *store.Bot {
	e.t.Helper()
	uid := e.userID()
	b, err := e.db.CreateBot(uid, name, "mock", "", mockserver.MockCredentials())
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
	if me["role"] != "superadmin" {
		t.Errorf("first user role = %v, want superadmin", me["role"])
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
	itemList, _ := json.Marshal([]map[string]any{{"type": "text", "text": "hello"}})
	for i := 0; i < 3; i++ {
		env.db.SaveMessage(&store.Message{
			BotID: botObj.ID, Direction: "inbound", FromUserID: "user@wechat",
			MessageType: 1, ItemList: itemList,
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
	contactItems, _ := json.Marshal([]map[string]any{{"type": "text", "text": "hi"}})
	for _, sender := range []string{"alice@wechat", "bob@wechat", "alice@wechat"} {
		env.db.SaveMessage(&store.Message{
			BotID: botObj.ID, Direction: "inbound", FromUserID: sender,
			MessageType: 1, ItemList: contactItems,
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
	sent := inst.Provider.(*mockserver.Provider).Engine().SentMessages()
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
	sent := inst.Provider.(*mockserver.Provider).Engine().SentMessages()
	var mediaSent *mockserver.SentMessage
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
	if string(mediaSent.MediaData) != "fake-jpeg-data" {
		t.Errorf("data = %q", string(mediaSent.MediaData))
	}
	if mediaSent.Text != "看看这张图" {
		t.Errorf("text = %q, want caption", mediaSent.Text)
	}

	// Verify message saved in DB
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range msgs {
		if m.Direction == "outbound" && strings.Contains(string(m.ItemList), `"image"`) {
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
	sent := inst.Provider.(*mockserver.Provider).Engine().SentMessages()
	var fileSent *mockserver.SentMessage
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

	// Superadmin cannot be demoted
	code, _ = env.put("/api/admin/users/"+adminID+"/role", map[string]string{"role": "member"})
	assertCode(t, "superadmin demote", code, 403)

	// Update status
	code, _ = env.put("/api/admin/users/"+newID+"/status", map[string]string{"status": "disabled"})
	assertCode(t, "disable user", code, 200)

	// Superadmin cannot be disabled
	code, _ = env.put("/api/admin/users/"+adminID+"/status", map[string]string{"status": "disabled"})
	assertCode(t, "superadmin disable", code, 403)

	// Reset password
	code, _ = env.put("/api/admin/users/"+newID+"/password", nil)
	assertCode(t, "reset password", code, 200)

	// Delete user
	code, _ = env.del("/api/admin/users/" + newID)
	assertCode(t, "delete user", code, 200)

	// Superadmin cannot be deleted
	code, _ = env.del("/api/admin/users/" + adminID)
	assertCode(t, "superadmin delete", code, 403)
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
	sent := inst.Provider.(*mockserver.Provider).Engine().SentMessages()
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
	mock := inst.Provider.(*mockserver.Provider)

	// @support → ch1 (handle match) + chAll (no handle, receives all)
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "@support help",
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

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "普通消息",
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

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "@nobody test",
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
		if strings.Contains(string(m.ItemList), "@nobody test") {
			foundNobody = true
		}
	}
	if !foundNobody {
		t.Error("chAll (no handle) should still receive @nobody in DB")
	}
}

// ==================== Inbound stored globally (no channel_id) ====================

func TestInboundStoredGlobally(t *testing.T) {
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
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "alice@wx",
		Text:   "hello",
	})
	readWSTimeout(t, ws, 2*time.Second)

	// Inbound stored globally (channel_id IS NULL), not per-channel
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	if len(msgs) != 1 {
		t.Fatalf("want 1 message, got %d", len(msgs))
	}
	if msgs[0].Direction != "inbound" {
		t.Errorf("direction = %q", msgs[0].Direction)
	}
	if msgs[0].ChannelID != nil {
		t.Errorf("channel_id should be nil, got %v", *msgs[0].ChannelID)
	}

	// ListChannelMessages still finds it via bot_id + sender
	chMsgs, _ := env.db.ListChannelMessages(ch.ID, "alice@wx", 10)
	if len(chMsgs) != 1 {
		t.Fatalf("channel query: want 1, got %d", len(chMsgs))
	}
}

func TestInboundNoMatchStoredWithoutChannelID(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("nomatch", "password123")
	botObj := env.createBotForUser("Bot1")
	env.mgr.StartBot(context.Background(), botObj)

	// Channel with user filter that won't match
	filter := &store.FilterRule{UserIDs: []string{"specific@wx"}}
	env.db.CreateChannel(botObj.ID, "Filtered", "", filter, nil)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	// Send from non-matching user
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "other@wx",
		Text:   "hello",
	})

	time.Sleep(100 * time.Millisecond)

	// Should be stored without channel_id
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range msgs {
		if m.FromUserID == "other@wx" {
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
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "hello raw",
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

	// Mockserver serializes the WeixinMessage as raw
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
	mock := inst.Provider.(*mockserver.Provider)

	// Send a voice message via mockserver
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Items:  []mockserver.ItemRequest{{Type: "voice"}},
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

	// Raw is auto-generated from the WeixinMessage
	if raw["from_user_id"] != "u@wx" {
		t.Errorf("raw.from_user_id = %v, want u@wx", raw["from_user_id"])
	}
	// message_id should be a positive number (auto-assigned by engine)
	if raw["message_id"] == nil || raw["message_id"] == float64(0) {
		t.Errorf("raw.message_id = %v, want >0", raw["message_id"])
	}

	// Verify item_list contains the voice item
	items := raw["item_list"].([]any)
	if len(items) == 0 {
		t.Fatal("raw.item_list is empty")
	}
	firstItem := items[0].(map[string]any)
	// ilink.ItemVoice = 3
	if firstItem["type"] != float64(3) {
		t.Errorf("raw item type = %v, want 3 (voice)", firstItem["type"])
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
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "@support help",
	})

	// Only first channel receives
	if readWSTimeout(t, ws1, 2*time.Second) == nil {
		t.Error("ch1 (first match) should receive")
	}
	if readWSTimeout(t, ws2, 300*time.Millisecond) != nil {
		t.Error("ch2 (second match) should NOT receive")
	}

	// Inbound stored globally — both channels can see it via bot_id
	msgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	if len(msgs) != 1 {
		t.Errorf("should have 1 global message, got %d", len(msgs))
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
	mock := inst.Provider.(*mockserver.Provider)

	// @support → ch1
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "@support help me",
	})
	readWSTimeout(t, ws1, 2*time.Second)

	// @sales → ch2
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "@sales price?",
	})
	readWSTimeout(t, ws2, 2*time.Second)

	// Add outbound (stored globally, no channel_id)
	r1Items, _ := json.Marshal([]map[string]any{{"type": "text", "text": "support reply"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "outbound",
		ToUserID: "u@wx", MessageType: 2, ItemList: r1Items,
	})
	r2Items, _ := json.Marshal([]map[string]any{{"type": "text", "text": "sales reply"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "outbound",
		ToUserID: "u@wx", MessageType: 2, ItemList: r2Items,
	})

	// All messages shared at bot level: 2 inbound + 2 outbound = 4
	msgs1, _ := env.db.ListChannelMessages(ch1.ID, "u@wx", 50)
	if len(msgs1) != 4 {
		t.Errorf("ch1: want 4, got %d", len(msgs1))
	}
	msgs2, _ := env.db.ListChannelMessages(ch2.ID, "u@wx", 50)
	if len(msgs2) != 4 {
		t.Errorf("ch2: want 4, got %d", len(msgs2))
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

	paginationItems, _ := json.Marshal([]map[string]any{{"type": "text", "text": "hello"}})
	for i := 0; i < 5; i++ {
		env.db.SaveMessage(&store.Message{
			BotID: botObj.ID, Direction: "inbound", FromUserID: "u@wx",
			MessageType: 1, ItemList: paginationItems,
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
	sent := inst.Provider.(*mockserver.Provider).Engine().SentMessages()
	if len(sent) != 1 || sent[0].Text != "hello via http" {
		t.Errorf("sent = %+v", sent)
	}

	// Verify message saved in DB (globally, no channel_id)
	dbMsgs, _ := env.db.ListMessages(botObj.ID, 10, 0)
	found := false
	for _, m := range dbMsgs {
		if m.Direction == "outbound" {
			found = true
		}
	}
	if !found {
		t.Error("outbound message not saved")
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
		&store.WebhookConfig{URL: hookSrv.URL, Auth: &store.WebhookAuth{Type: "bearer", Token: "test-token"}}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "hook@wx",
		Text:   "webhook test",
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
		&store.WebhookConfig{URL: hookSrv.URL, Auth: &store.WebhookAuth{Type: "hmac", Secret: "my-secret"}}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "hmac@wx",
		Text:   "signed",
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
		&store.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "alice@wx",
		Text:   "script test",
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
		&store.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	// Text message → should deliver
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "hello",
	})
	time.Sleep(300 * time.Millisecond)
	if !received {
		t.Error("text message should trigger webhook")
	}

	// Image message → script returns null, should skip
	received = false
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Items:  []mockserver.ItemRequest{{Type: "image"}},
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
		&store.WebhookConfig{URL: hookSrv.URL, Script: script}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "question",
	})

	time.Sleep(500 * time.Millisecond)

	// Verify bot sent reply "42" back to user
	sent := mock.Engine().SentMessages()
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
		if strings.Contains(string(m.ItemList), "42") && m.Direction == "outbound" {
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
		&store.WebhookConfig{URL: hookSrv.URL}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "hi",
	})

	time.Sleep(500 * time.Millisecond)

	sent := mock.Engine().SentMessages()
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
	mock := inst.Provider.(*mockserver.Provider)

	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "no hook",
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

	// Inbound stored globally (no channel_id)
	items1, _ := json.Marshal([]map[string]any{{"type": "text", "text": "help me"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "inbound",
		FromUserID: sender, MessageType: 1, ItemList: items1,
	})
	items2, _ := json.Marshal([]map[string]any{{"type": "text", "text": "price?"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "inbound",
		FromUserID: sender, MessageType: 1, ItemList: items2,
	})

	// Outbound replies (stored globally, no channel_id)
	reply1, _ := json.Marshal([]map[string]any{{"type": "text", "text": "support reply"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "outbound",
		ToUserID: sender, MessageType: 2, ItemList: reply1,
	})
	reply2, _ := json.Marshal([]map[string]any{{"type": "text", "text": "sales reply"}})
	env.db.SaveMessage(&store.Message{
		BotID: botObj.ID, Direction: "outbound",
		ToUserID: sender, MessageType: 2, ItemList: reply2,
	})

	// All messages shared at bot level: 2 inbound + 2 outbound = 4
	msgs1, err := env.db.ListChannelMessages(ch1.ID, sender, 50)
	if err != nil {
		t.Fatalf("ch1: %v", err)
	}
	if len(msgs1) != 4 {
		t.Errorf("ch1: want 4, got %d", len(msgs1))
	}
	msgs2, err := env.db.ListChannelMessages(ch2.ID, sender, 50)
	if err != nil {
		t.Fatalf("ch2: %v", err)
	}
	if len(msgs2) != 4 {
		t.Errorf("ch2: want 4, got %d", len(msgs2))
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
	objStore, err := storage.New(storage.Config{
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
		Store: db, SessionStore: auth.NewSessionStore(), Config: cfg,
		OAuthStates: api.SetupOAuth(cfg), ObjectStore: objStore,
	}
	hub := relay.NewHub(server.SetupUpstreamHandler())
	aiSink := &sink.AI{Store: db}
	mgr := bot.NewManager(db, hub, aiSink, objStore, "http://localhost")
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
	botObj, _ := db.CreateBot(userID, "MediaBot", "mock", "", mockserver.MockCredentials())
	mgr.StartBot(context.Background(), botObj)
	ch, _ := db.CreateChannel(botObj.ID, "MediaChan", "", nil, nil)

	// Simulate inbound with image media
	inst, _ := mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Items: []mockserver.ItemRequest{{
			Type: "image",
			Data: []byte("mock-media-data"),
		}},
	})

	// Message should be saved immediately
	time.Sleep(50 * time.Millisecond)
	msgs, _ := db.ListChannelMessages(ch.ID, "u@wx", 10)
	if len(msgs) == 0 {
		t.Fatal("no messages found")
	}
	earlyStatus := msgs[0].MediaStatus
	t.Logf("early media_status = %s", earlyStatus)

	// Wait for async download to complete
	time.Sleep(500 * time.Millisecond)
	msgs, _ = db.ListChannelMessages(ch.ID, "u@wx", 10)

	status := msgs[0].MediaStatus
	if status != "ready" {
		t.Fatalf("media_status = %q, want ready", status)
	}

	var mediaKeys map[string]string
	json.Unmarshal(msgs[0].MediaKeys, &mediaKeys)
	mediaKey := mediaKeys["0"]
	if mediaKey == "" {
		t.Fatalf("media_keys[0] not found: %s", string(msgs[0].MediaKeys))
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


// ==================== Webhook Plugin E2E (two-table schema) ====================

// submitPlugin is a helper that submits a plugin and returns (pluginID, versionID).
func (e *testEnv) submitPlugin(script string) (pluginID, versionID string) {
	e.t.Helper()
	code, result := e.postCode("/api/webhook-plugins/submit", map[string]string{"script": script})
	if code != 200 {
		e.t.Fatalf("submit: %d %v", code, result)
	}
	pid, _ := result["plugin_id"].(string)
	vid, _ := result["version_id"].(string)
	if pid == "" || vid == "" {
		e.t.Fatalf("submit returned empty IDs: %v", result)
	}
	return pid, vid
}

// approveVersion approves a plugin version by version ID.
func (e *testEnv) approveVersion(versionID string) {
	e.t.Helper()
	e.put("/api/admin/webhook-plugins/"+versionID+"/review", map[string]string{"status": "approved"})
}

func TestWebhookPluginFullLifecycle(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("plugadmin", "password123")
	u, _ := env.db.GetUserByUsername("plugadmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	pluginScript := `// @name 测试通知
// @author testauthor
// @version 1.0.0
// @config target_url string "目标 URL"

function onRequest(ctx) {
	ctx.req.headers["X-Plugin"] = "test-notify";
	ctx.req.body = JSON.stringify({text: ctx.msg.sender + ": " + ctx.msg.content});
}

function onResponse(ctx) {
	var data = JSON.parse(ctx.res.body);
	if (data.reply) reply(data.reply);
}`

	// 1. Submit
	pluginID, versionID := env.submitPlugin(pluginScript)
	if pluginID == "" || versionID == "" {
		t.Fatal("empty IDs")
	}

	// 2. Pending not in default list
	code, approved := env.getList("/api/webhook-plugins")
	assertCode(t, "list approved", code, 200)
	if len(approved) != 0 {
		t.Errorf("approved should be empty, got %d", len(approved))
	}

	// Admin sees pending versions
	code, pending := env.getList("/api/webhook-plugins?status=pending")
	assertCode(t, "list pending", code, 200)
	if len(pending) != 1 {
		t.Fatalf("pending: want 1, got %d", len(pending))
	}

	// 3. Approve
	env.approveVersion(versionID)

	code, approved = env.getList("/api/webhook-plugins")
	assertCode(t, "after approve", code, 200)
	if len(approved) != 1 {
		t.Fatalf("approved: want 1, got %d", len(approved))
	}

	// 4. Install
	code, installResult := env.postCode("/api/webhook-plugins/"+pluginID+"/install", nil)
	assertCode(t, "install", code, 200)
	installedScript := installResult["script"].(string)
	if !strings.Contains(installedScript, "onRequest") {
		t.Error("script should contain onRequest")
	}

	// 5. Check install count
	code, detail := env.get("/api/webhook-plugins/" + pluginID)
	assertCode(t, "detail", code, 200)
	p := detail["plugin"].(map[string]any)
	if p["install_count"] != float64(1) {
		t.Errorf("install_count = %v", p["install_count"])
	}

	// 6. Execute via webhook
	var received []map[string]any
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		received = append(received, body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"reply": "auto-reply"})
	}))
	defer hookSrv.Close()

	botObj := env.createBotForUser("PlugBot")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "PlugChan", "", nil, nil)
	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&store.WebhookConfig{URL: hookSrv.URL, VersionID: versionID}, true)

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "alice@wx",
		Text:   "hello",
	})
	time.Sleep(500 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("webhook: want 1, got %d", len(received))
	}
	sent := mock.Engine().SentMessages()
	replyFound := false
	for _, m := range sent {
		if m.Text == "auto-reply" { replyFound = true }
	}
	if !replyFound {
		t.Error("reply not sent")
	}
}

func TestWebhookPluginRejectWithReason(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("rejectadmin", "password123")
	u, _ := env.db.GetUserByUsername("rejectadmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	pluginID, versionID := env.submitPlugin("// @name BadPlugin\nfunction onRequest(ctx) {}")

	code, _ := env.put("/api/admin/webhook-plugins/"+versionID+"/review", map[string]any{
		"status": "rejected", "reason": "infinite loop",
	})
	assertCode(t, "reject", code, 200)

	// Plugin exists but no approved version → install fails
	code, _ = env.postCode("/api/webhook-plugins/"+pluginID+"/install", nil)
	if code != 404 {
		t.Errorf("install rejected: got %d, want 404", code)
	}
}

func TestWebhookPluginNonAdminCannotReview(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("firstadmin", "password123")
	env.post("/api/auth/logout", nil)
	env.register("normaluser", "password123")

	_, versionID := env.submitPlugin("// @name NormalPlugin\nfunction onRequest(ctx) {}")

	code, _ := env.put("/api/admin/webhook-plugins/"+versionID+"/review", map[string]string{"status": "approved"})
	if code != 403 {
		t.Errorf("non-admin review: got %d, want 403", code)
	}

	// Non-admin cannot see pending
	code, pending := env.getList("/api/webhook-plugins?status=pending")
	if code != 403 {
		t.Errorf("non-admin pending: got %d, want 403", code)
	}
	_ = pending
}

func TestWebhookPluginSubmitRequiresAuth(t *testing.T) {
	env := setup(t)
	defer env.close()

	resp := env.postRaw("/api/webhook-plugins/submit", map[string]string{
		"script": "// @name Test\nfunction onRequest(ctx) {}",
	})
	resp.Body.Close()
	if resp.StatusCode != 401 {
		t.Errorf("got %d, want 401", resp.StatusCode)
	}
}

func TestWebhookPluginSubmitNoName(t *testing.T) {
	env := setup(t)
	defer env.close()
	env.register("noname", "password123")

	code, result := env.postCode("/api/webhook-plugins/submit", map[string]string{
		"script": "function onRequest(ctx) {}",
	})
	if code != 400 {
		t.Errorf("got %d, want 400", code)
	}
	if result["error"] == nil || !strings.Contains(result["error"].(string), "@name") {
		t.Errorf("error = %v", result["error"])
	}
}

func TestWebhookPluginDeleteByAdmin(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("deladmin", "password123")
	u, _ := env.db.GetUserByUsername("deladmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	pluginID, _ := env.submitPlugin("// @name DeleteMe\nfunction onRequest(ctx) {}")

	code, _ := env.del("/api/admin/webhook-plugins/" + pluginID)
	assertCode(t, "delete", code, 200)

	code, _ = env.get("/api/webhook-plugins/" + pluginID)
	if code != 404 {
		t.Errorf("after delete: got %d, want 404", code)
	}
}

func TestWebhookPluginVersionHistory(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("veradmin", "password123")
	u, _ := env.db.GetUserByUsername("veradmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	// Submit v1
	pluginID, v1ID := env.submitPlugin("// @name VersionedPlugin\n// @version 1.0.0\nfunction onRequest(ctx) {}")
	env.approveVersion(v1ID)

	// Submit v2 (same name, same plugin)
	pluginID2, v2ID := env.submitPlugin("// @name VersionedPlugin\n// @version 2.0.0\nfunction onRequest(ctx) {}")
	if pluginID != pluginID2 {
		t.Errorf("same name should reuse plugin: %s vs %s", pluginID, pluginID2)
	}
	if v1ID == v2ID {
		t.Error("versions should have different IDs")
	}

	env.approveVersion(v2ID)

	// List versions
	code, versions := env.getList("/api/webhook-plugins/" + pluginID + "/versions")
	assertCode(t, "versions", code, 200)
	if len(versions) != 2 {
		t.Fatalf("want 2 versions, got %d", len(versions))
	}
}

func TestWebhookPluginResubmitSupersedesPending(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("resubuser", "password123")

	pluginID, v1ID := env.submitPlugin("// @name ResubPlugin\n// @version 1.0.0\nfunction onRequest(ctx) {}")
	pluginID2, v2ID := env.submitPlugin("// @name ResubPlugin\n// @version 2.0.0\nfunction onRequest(ctx) {}")

	// Same plugin, different version IDs
	if pluginID != pluginID2 {
		t.Errorf("should reuse plugin: %s vs %s", pluginID, pluginID2)
	}
	if v1ID == v2ID {
		t.Error("should create new version, not overwrite")
	}

	// v1 should be superseded, v2 should be pending
	u, _ := env.db.GetUserByUsername("resubuser"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }
	code, versions := env.getList("/api/webhook-plugins/" + pluginID + "/versions")
	assertCode(t, "versions", code, 200)
	if len(versions) != 2 {
		t.Fatalf("want 2, got %d", len(versions))
	}
	for _, v := range versions {
		ver := v.(map[string]any)
		if ver["id"] == v1ID && ver["status"] != "superseded" {
			t.Errorf("v1 should be superseded, got %v", ver["status"])
		}
		if ver["id"] == v2ID && ver["status"] != "pending" {
			t.Errorf("v2 should be pending, got %v", ver["status"])
		}
	}
}

func TestWebhookPluginNameOwnership(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("owner1", "password123")
	env.submitPlugin("// @name UniquePlugin\nfunction onRequest(ctx) {}")
	env.post("/api/auth/logout", nil)

	env.register("owner2", "password123")
	code, result := env.postCode("/api/webhook-plugins/submit", map[string]string{
		"script": "// @name UniquePlugin\nfunction onRequest(ctx) {}",
	})
	if code != 409 {
		t.Errorf("got %d, want 409", code)
	}
	if result["error"] == nil || !strings.Contains(result["error"].(string), "taken") {
		t.Errorf("error = %v", result["error"])
	}
}

func TestWebhookPluginInstallToChannel(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("chtadmin", "password123")
	u, _ := env.db.GetUserByUsername("chtadmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	pluginID, versionID := env.submitPlugin(`// @name ChanPlugin
// @version 1.0.0
function onRequest(ctx) {
	ctx.req.headers["X-Test"] = "yes";
	ctx.req.body = JSON.stringify({ok: true});
}`)
	env.approveVersion(versionID)

	botObj := env.createBotForUser("ChBot")
	env.mgr.StartBot(context.Background(), botObj)
	ch, _ := env.db.CreateChannel(botObj.ID, "Ch1", "", nil, nil)

	var received []map[string]any
	hookSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		received = append(received, body)
		w.WriteHeader(200)
	}))
	defer hookSrv.Close()

	env.db.UpdateChannel(ch.ID, ch.Name, ch.Handle, &ch.FilterRule, &ch.AIConfig,
		&store.WebhookConfig{URL: hookSrv.URL}, true)

	code, _ := env.postCode("/api/webhook-plugins/"+pluginID+"/install-to-channel", map[string]string{
		"bot_id": botObj.ID, "channel_id": ch.ID,
	})
	assertCode(t, "install to channel", code, 200)

	// Verify channel references version ID
	updatedCh, _ := env.db.GetChannel(ch.ID)
	if updatedCh.WebhookConfig.VersionID == "" {
		t.Error("plugin_id not set")
	}

	inst, _ := env.mgr.GetInstance(botObj.ID)
	mock := inst.Provider.(*mockserver.Provider)
	mock.Engine().InjectInbound(mockserver.InboundRequest{
		Sender: "u@wx",
		Text:   "test",
	})
	time.Sleep(500 * time.Millisecond)

	if len(received) != 1 {
		t.Fatalf("want 1, got %d", len(received))
	}
	if received[0]["ok"] != true {
		t.Error("plugin did not run")
	}
}

func TestWebhookPluginInstallCountTracksUsers(t *testing.T) {
	env := setup(t)
	defer env.close()

	env.register("countadmin", "password123")
	u, _ := env.db.GetUserByUsername("countadmin"); if u != nil { env.db.UpdateUserRole(u.ID, "admin") }

	pluginID, versionID := env.submitPlugin("// @name CountPlugin\nfunction onRequest(ctx) {}")
	env.approveVersion(versionID)

	env.postCode("/api/webhook-plugins/"+pluginID+"/install", nil)
	_, detail := env.get("/api/webhook-plugins/" + pluginID)
	p := detail["plugin"].(map[string]any)
	if p["install_count"] != float64(1) {
		t.Errorf("count = %v, want 1", p["install_count"])
	}

	// Same user again — no double count
	env.postCode("/api/webhook-plugins/"+pluginID+"/install", nil)
	_, detail = env.get("/api/webhook-plugins/" + pluginID)
	p = detail["plugin"].(map[string]any)
	if p["install_count"] != float64(1) {
		t.Errorf("count = %v, want 1", p["install_count"])
	}

	// Different user
	env.post("/api/auth/logout", nil)
	env.register("countuser2", "password123")
	env.postCode("/api/webhook-plugins/"+pluginID+"/install", nil)
	_, detail = env.get("/api/webhook-plugins/" + pluginID)
	p = detail["plugin"].(map[string]any)
	if p["install_count"] != float64(2) {
		t.Errorf("count = %v, want 2", p["install_count"])
	}
}

// ==================== AI image handling ====================

// TestAIToolImageReply verifies the full flow when an AI tool call returns an image:
// 1. User sends text → AI calls a tool
// 2. App returns image in reply_base64
// 3. Image is sent directly to user AND passed as multimodal content to LLM
// 4. LLM returns final text reply
func TestAIToolImageReply(t *testing.T) {
	// --- PNG test image (1x1 red pixel) ---
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xD7, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x36, 0x28, 0x19,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
		0x44, 0xAE, 0x42, 0x60, 0x82,
	}
	pngBase64 := base64.StdEncoding.EncodeToString(pngData)

	// --- Mock app server (returns image on tool call) ---
	appSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"reply":        "Here is the chart",
			"reply_type":   "image",
			"reply_base64": "data:image/png;base64," + pngBase64,
			"reply_name":   "chart.png",
		})
	}))
	defer appSrv.Close()

	// --- Mock LLM server ---
	var llmCallCount int
	var gotMultimodalToolResult bool
	var instIDForLLM string // set after InstallApp, read by LLM handler

	llmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []json.RawMessage `json:"messages"`
			Tools    []json.RawMessage `json:"tools"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		llmCallCount++

		w.Header().Set("Content-Type", "application/json")

		if llmCallCount == 1 {
			// First call: return tool_call
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id":   "call_img",
							"type": "function",
							"function": map[string]any{
								"name":      instIDForLLM + "__generate_chart",
								"arguments": `{"metric":"cpu"}`,
							},
						}},
					},
					"finish_reason": "tool_calls",
				}},
			})
			return
		}

		// Second call: tool result should contain multimodal content
		for _, raw := range req.Messages {
			var msg struct {
				Role    string `json:"role"`
				Content any    `json:"content"`
			}
			json.Unmarshal(raw, &msg)
			if msg.Role == "tool" {
				// Check if content is array (multimodal) vs string
				var contentArr []map[string]any
				contentBytes, _ := json.Marshal(msg.Content)
				if json.Unmarshal(contentBytes, &contentArr) == nil && len(contentArr) > 0 {
					for _, part := range contentArr {
						if part["type"] == "image_url" {
							gotMultimodalToolResult = true
						}
					}
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"role":    "assistant",
					"content": "The CPU chart shows normal utilization.",
				},
				"finish_reason": "stop",
			}},
		})
	}))
	defer llmSrv.Close()

	// --- Setup environment ---
	db := testDB(t)
	defer db.Close()

	cfg := &config.Config{RPOrigin: "http://localhost", RPID: "localhost", RPName: "Test", Secret: "test"}
	server := &api.Server{
		Store: db, SessionStore: auth.NewSessionStore(), Config: cfg,
		OAuthStates: api.SetupOAuth(cfg),
	}
	hub := relay.NewHub(server.SetupUpstreamHandler())
	appDisp := appdelivery.NewDispatcher(db)
	aiSink := &sink.AI{Store: db, AppDisp: appDisp}
	mgr := bot.NewManager(db, hub, aiSink, nil, "http://localhost")
	server.BotManager = mgr
	server.Hub = hub
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	defer mgr.StopAll()

	// Configure AI to use mock LLM
	db.SetConfig("ai.api_key", "test-key")
	db.SetConfig("ai.base_url", llmSrv.URL)
	db.SetConfig("ai.model", "test-model")

	// Create user + bot
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}
	body, _ := json.Marshal(map[string]string{"username": "imguser", "password": "password123"})
	resp, _ := client.Post(ts.URL+"/api/auth/register", "application/json", bytes.NewReader(body))
	resp.Body.Close()
	resp, _ = client.Get(ts.URL + "/api/me")
	var me map[string]any
	json.NewDecoder(resp.Body).Decode(&me)
	resp.Body.Close()
	userID := me["id"].(string)

	botObj, _ := db.CreateBot(userID, "AIImgBot", "mock", "", mockserver.MockCredentials())
	db.UpdateBotAIEnabled(botObj.ID, true)

	// Create app with a tool
	tools, _ := json.Marshal([]store.AppTool{{
		Name:        "generate_chart",
		Description: "Generate a chart image",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"metric":{"type":"string"}}}`),
	}})
	app, err := db.CreateApp(&store.App{
		OwnerID:       userID,
		Name:          "ChartApp",
		Slug:          "chart-app",
		Tools:         tools,
		WebhookURL:    appSrv.URL,
		WebhookSecret: "secret",
		Status:        "active",
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}

	inst, err := db.InstallApp(app.ID, botObj.ID)
	if err != nil {
		t.Fatalf("InstallApp: %v", err)
	}
	instIDForLLM = inst.ID

	// Start bot
	if err := mgr.StartBot(context.Background(), botObj); err != nil {
		t.Fatalf("StartBot: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Get mock engine
	botInst, ok := mgr.GetInstance(botObj.ID)
	if !ok {
		t.Fatal("bot instance not found")
	}
	engine := botInst.Provider.(*mockserver.Provider).Engine()

	// --- Inject inbound text message ---
	engine.InjectInbound(mockserver.InboundRequest{
		Sender: "user@wechat",
		Text:   "show me cpu chart",
	})

	// Wait for all replies: status msg + image + text reply
	time.Sleep(3 * time.Second)
	sent := engine.SentMessages()

	// --- Verify results ---

	// Should have at least 3 messages: tool status, image, and text reply
	if len(sent) < 3 {
		t.Fatalf("expected >= 3 sent messages, got %d: %+v", len(sent), sent)
	}

	// Find the image message (has MediaData)
	var hasImage, hasTextReply bool
	for _, msg := range sent {
		if len(msg.MediaData) > 0 {
			hasImage = true
			// Verify it's the PNG we sent
			if !bytes.HasPrefix(msg.MediaData, []byte{0x89, 0x50, 0x4E, 0x47}) {
				t.Error("image data is not a valid PNG")
			}
		}
		if msg.Text == "The CPU chart shows normal utilization." {
			hasTextReply = true
		}
	}

	if !hasImage {
		t.Error("image was not sent to user")
	}
	if !hasTextReply {
		t.Error("final text reply was not sent")
	}
	if !gotMultimodalToolResult {
		t.Error("LLM did not receive multimodal image content in tool result")
	}
	if llmCallCount != 2 {
		t.Errorf("LLM called %d times, want 2", llmCallCount)
	}
}
