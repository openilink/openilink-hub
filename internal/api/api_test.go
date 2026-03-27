package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// testEnv bundles a running httptest.Server, the underlying store, and a
// pre-created admin user + session cookie for authenticated dashboard requests.
type testEnv struct {
	ts      *httptest.Server
	store   store.Store
	user    *store.User
	cookie  *http.Cookie
	handler http.Handler
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	// Create an admin user.
	u, err := s.CreateUserFull("testadmin", "", "Test Admin", "hashed", store.RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUserFull: %v", err)
	}
	// Ensure user is active.
	_ = s.UpdateUserStatus(u.ID, store.StatusActive)

	// Create a session so dashboard (cookie-auth) requests work.
	sessionToken, err := auth.CreateSession(s, u.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	cookie := &http.Cookie{Name: "session", Value: sessionToken}

	srv := &Server{
		Store:       s,
		Config:      &config.Config{RPOrigin: "http://localhost"},
		OAuthStates: newOAuthStateStore(),
	}

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	return &testEnv{
		ts:      ts,
		store:   s,
		user:    u,
		cookie:  cookie,
		handler: handler,
	}
}

// doJSON is a helper that sends a JSON request to the httptest server and
// returns the response. It supports optional cookies and auth headers.
func doJSON(t *testing.T, ts *httptest.Server, method, path string, body any, opts ...func(*http.Request)) *http.Response {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, opt := range opts {
		opt(req)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	return resp
}

func withCookie(c *http.Cookie) func(*http.Request) {
	return func(r *http.Request) { r.AddCookie(c) }
}

func withBearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

func decodeJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	resp.Body.Close()
	return m
}

// createTestBot creates a bot owned by the given user via the store.
func createTestBot(t *testing.T, s store.Store, userID, name string) *store.Bot {
	t.Helper()
	b, err := s.CreateBot(userID, name, "test", "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}
	return b
}

// createTestApp creates an app with the given scopes via the store.
func createTestApp(t *testing.T, s store.Store, ownerID, name, slug string, scopes []string) *store.App {
	t.Helper()
	scopesJSON, _ := json.Marshal(scopes)
	app, err := s.CreateApp(&store.App{
		OwnerID: ownerID,
		Name:    name,
		Slug:    slug,
		Scopes:  scopesJSON,
	})
	if err != nil {
		t.Fatalf("CreateApp(%q): %v", name, err)
	}
	return app
}

// installTestApp installs an app on a bot via the store and snapshots the app's
// scopes onto the installation (matching the Slack-model install flow).
func installTestApp(t *testing.T, s store.Store, appID, botID string) *store.AppInstallation {
	t.Helper()
	inst, err := s.InstallApp(appID, botID)
	if err != nil {
		t.Fatalf("InstallApp: %v", err)
	}
	// Snapshot app scopes at install time (Slack model)
	app, err := s.GetApp(appID)
	if err == nil && app != nil && len(app.Scopes) > 0 {
		_ = s.UpdateInstallation(inst.ID, inst.Handle, inst.Config, app.Scopes, inst.Enabled)
		inst.Scopes = app.Scopes
	}
	return inst
}

// ---------------------------------------------------------------------------
// Test: Bot API scope checks (app_token auth)
// ---------------------------------------------------------------------------

func TestBotAPI_ScopeChecks(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "scope-bot")

	// App with only message:write scope.
	appMsgOnly := createTestApp(t, env.store, env.user.ID, "msg-app", "msg-app", []string{"message:write"})
	instMsgOnly := installTestApp(t, env.store, appMsgOnly.ID, bot.ID)

	// App with all three scopes.
	appAll := createTestApp(t, env.store, env.user.ID, "all-app", "all-app",
		[]string{"message:write", "contact:read", "bot:read"})
	instAll := installTestApp(t, env.store, appAll.ID, bot.ID)

	t.Run("message:write scope allows send", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/bot/v1/message/send",
			map[string]string{"content": "hello"},
			withBearer(instMsgOnly.AppToken))
		defer resp.Body.Close()
		// Scope check should pass. We expect some non-403 error because
		// BotManager is nil (no live bot). 503 or panic-recovery 500 are OK.
		if resp.StatusCode == http.StatusForbidden {
			body := decodeJSON(t, resp)
			t.Fatalf("expected scope check to pass, got 403: %v", body)
		}
	})

	t.Run("missing contact:read scope denied", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/contact", nil,
			withBearer(instMsgOnly.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for missing contact:read scope, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if msg, _ := body["error"].(string); msg != "missing scope: contact:read" {
			t.Errorf("unexpected error message: %q", msg)
		}
	})

	t.Run("missing bot:read scope denied", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/info", nil,
			withBearer(instMsgOnly.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for missing bot:read scope, got %d", resp.StatusCode)
		}
	})

	t.Run("all scopes allow contact endpoint", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/contact", nil,
			withBearer(instAll.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden {
			t.Fatalf("expected scope check to pass for contact:read, got 403")
		}
		// 200 or 500 (if store query fails) are both acceptable.
	})

	t.Run("all scopes allow bot info endpoint", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/info", nil,
			withBearer(instAll.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusForbidden {
			t.Fatalf("expected scope check to pass for bot:read, got 403")
		}
	})

	t.Run("backward compat paths also check scopes", func(t *testing.T) {
		// /bot/v1/contacts (old path) with missing scope
		resp := doJSON(t, env.ts, "GET", "/bot/v1/contacts", nil,
			withBearer(instMsgOnly.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("old path /bot/v1/contacts should deny missing scope, got %d", resp.StatusCode)
		}

		// /bot/v1/bot (old path) with missing scope
		resp2 := doJSON(t, env.ts, "GET", "/bot/v1/bot", nil,
			withBearer(instMsgOnly.AppToken))
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusForbidden {
			t.Errorf("old path /bot/v1/bot should deny missing scope, got %d", resp2.StatusCode)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/contact", nil,
			withBearer("totally-invalid-token"))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for invalid token, got %d", resp.StatusCode)
		}
	})

	t.Run("missing auth header returns 401", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/bot/v1/contact", nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for missing header, got %d", resp.StatusCode)
		}
	})

	t.Run("disabled installation returns 403", func(t *testing.T) {
		// Disable the installation.
		_ = env.store.UpdateInstallation(instMsgOnly.ID, instMsgOnly.Handle, instMsgOnly.Config, instMsgOnly.Scopes, false)

		resp := doJSON(t, env.ts, "POST", "/bot/v1/message/send",
			map[string]string{"content": "hello"},
			withBearer(instMsgOnly.AppToken))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for disabled installation, got %d", resp.StatusCode)
		}

		// Re-enable for other tests.
		_ = env.store.UpdateInstallation(instMsgOnly.ID, instMsgOnly.Handle, instMsgOnly.Config, instMsgOnly.Scopes, true)
	})
}

// ---------------------------------------------------------------------------
// Test: App CRUD via dashboard API (session auth)
// ---------------------------------------------------------------------------

func TestAppAPI_CRUD(t *testing.T) {
	env := setupTestEnv(t)

	var createdAppID string

	t.Run("create app", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name":   "My Test App",
			"slug":   "my-test-app",
			"scopes": []string{"message:write", "bot:read"},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		createdAppID, _ = body["id"].(string)
		if createdAppID == "" {
			t.Fatal("created app has no id")
		}
		if slug, _ := body["slug"].(string); slug != "my-test-app" {
			t.Errorf("slug = %q, want %q", slug, "my-test-app")
		}
	})

	t.Run("get app", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+createdAppID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if name, _ := body["name"].(string); name != "My Test App" {
			t.Errorf("name = %q, want %q", name, "My Test App")
		}
	})

	t.Run("update app", func(t *testing.T) {
		newName := "Updated App"
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+createdAppID, map[string]any{
			"name":   &newName,
			"scopes": []string{"message:write", "contact:read"},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		// Verify update persisted.
		app, _ := env.store.GetApp(createdAppID)
		if app.Name != "Updated App" {
			t.Errorf("name after update = %q, want %q", app.Name, "Updated App")
		}
	})

	t.Run("marketplace app cannot be edited", func(t *testing.T) {
		// Create a marketplace (registry) app directly via store.
		mApp, err := env.store.CreateApp(&store.App{
			OwnerID:  env.user.ID,
			Name:     "Marketplace App",
			Slug:     "marketplace-app",
			Registry: "https://registry.example.com",
		})
		if err != nil {
			t.Fatalf("create marketplace app: %v", err)
		}

		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+mApp.ID, map[string]any{
			"name": "Hacked Name",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for marketplace app edit, got %d", resp.StatusCode)
		}
	})

	t.Run("delete app", func(t *testing.T) {
		resp := doJSON(t, env.ts, "DELETE", "/api/apps/"+createdAppID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		// Verify deleted.
		_, err := env.store.GetApp(createdAppID)
		if err == nil {
			t.Error("app should be deleted")
		}
	})

	t.Run("create app with invalid slug fails", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name": "Bad Slug",
			"slug": "-bad-",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid slug, got %d", resp.StatusCode)
		}
	})

	t.Run("create app without name fails", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"slug": "no-name-app",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for missing name, got %d", resp.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Install endpoint (POST /api/apps/{id}/install)
// ---------------------------------------------------------------------------

func TestInstallApp(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "install-bot")

	t.Run("install by app_id", func(t *testing.T) {
		app := createTestApp(t, env.store, env.user.ID, "Direct App", "direct-app",
			[]string{"message:write"})

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install", map[string]any{
			"bot_id": bot.ID,
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
	})

	t.Run("handle conflict returns 409", func(t *testing.T) {
		app := createTestApp(t, env.store, env.user.ID, "Handle App", "handle-app",
			[]string{"message:write"})

		// First install with a handle.
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install", map[string]any{
			"bot_id": bot.ID,
			"handle": "myhandle",
		}, withCookie(env.cookie))
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("first install expected 201, got %d", resp.StatusCode)
		}

		// Second install with the same handle should conflict.
		app2 := createTestApp(t, env.store, env.user.ID, "Handle App2", "handle-app2",
			[]string{"message:write"})
		resp2 := doJSON(t, env.ts, "POST", "/api/apps/"+app2.ID+"/install", map[string]any{
			"bot_id": bot.ID,
			"handle": "myhandle",
		}, withCookie(env.cookie))
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 for handle conflict, got %d", resp2.StatusCode)
		}
	})

	t.Run("missing bot_id returns 400", func(t *testing.T) {
		app := createTestApp(t, env.store, env.user.ID, "NoBotID App", "nobotid-app",
			[]string{"message:write"})

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install", map[string]any{},
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("nonexistent bot returns 404", func(t *testing.T) {
		app := createTestApp(t, env.store, env.user.ID, "Good App", "good-app",
			[]string{"message:write"})

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install", map[string]any{
			"bot_id": "nonexistent-id",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 for nonexistent bot, got %d", resp.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Slug uniqueness via API
// ---------------------------------------------------------------------------

func TestSlugUniqueness(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("duplicate local slug returns 409", func(t *testing.T) {
		// First app succeeds.
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name": "First App",
			"slug": "unique-slug",
		}, withCookie(env.cookie))
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("first create expected 201, got %d", resp.StatusCode)
		}

		// Second app with same slug should fail.
		resp2 := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name": "Second App",
			"slug": "unique-slug",
		}, withCookie(env.cookie))
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 for duplicate slug, got %d", resp2.StatusCode)
		}
	})

	t.Run("builtin app with same slug as local succeeds", func(t *testing.T) {
		// Create a local app.
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name": "Local App",
			"slug": "shared-slug",
		}, withCookie(env.cookie))
		resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("local create expected 201, got %d", resp.StatusCode)
		}

		// Create a builtin app with the same slug directly via store
		// (builtin apps are created via template install, not the API).
		_, err := env.store.CreateApp(&store.App{
			OwnerID:  env.user.ID,
			Name:     "Builtin App",
			Slug:     "shared-slug",
			Registry: "builtin",
		})
		if err != nil {
			t.Fatalf("expected builtin app creation to succeed, got: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Session auth edge cases
// ---------------------------------------------------------------------------

func TestSessionAuth(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("no cookie returns 401", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps", nil)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid session token returns 401", func(t *testing.T) {
		badCookie := &http.Cookie{Name: "session", Value: "invalid-token-xyz"}
		resp := doJSON(t, env.ts, "GET", "/api/apps", nil, withCookie(badCookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
	})

	t.Run("expired session returns 401", func(t *testing.T) {
		// Create an already-expired session.
		expiredToken := "expired-session-token-123"
		_ = env.store.CreateSession(expiredToken, env.user.ID, time.Now().Add(-1*time.Hour))

		expiredCookie := &http.Cookie{Name: "session", Value: expiredToken}
		resp := doJSON(t, env.ts, "GET", "/api/apps", nil, withCookie(expiredCookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for expired session, got %d", resp.StatusCode)
		}
	})

	t.Run("valid session returns 200", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps", nil, withCookie(env.cookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("disabled user returns 403", func(t *testing.T) {
		// Create a disabled user with a valid session.
		u2, _ := env.store.CreateUserFull("disabled-user", "", "Disabled", "hashed", store.RoleMember)
		_ = env.store.UpdateUserStatus(u2.ID, store.StatusDisabled)
		tok, _ := auth.CreateSession(env.store, u2.ID)
		disabledCookie := &http.Cookie{Name: "session", Value: tok}

		resp := doJSON(t, env.ts, "GET", "/api/apps", nil, withCookie(disabledCookie))
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for disabled user, got %d", resp.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Bot API with legacy vs new scope names
// Ensures new scope format (colon-separated) works and old format would fail.
// ---------------------------------------------------------------------------

func TestBotAPI_NewScopeFormat(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "scope-format-bot")

	// App explicitly using new colon-separated scope names.
	app := createTestApp(t, env.store, env.user.ID, "New Scope App", "new-scope-app",
		[]string{"message:write", "message:read", "contact:read", "bot:read"})
	inst := installTestApp(t, env.store, app.ID, bot.ID)

	endpoints := []struct {
		method string
		path   string
		scope  string
		body   any
	}{
		{"POST", "/bot/v1/message/send", "message:write", map[string]string{"content": "hi"}},
		{"GET", "/bot/v1/contact", "contact:read", nil},
		{"GET", "/bot/v1/info", "bot:read", nil},
	}

	for _, ep := range endpoints {
		t.Run(ep.scope+" allows "+ep.path, func(t *testing.T) {
			resp := doJSON(t, env.ts, ep.method, ep.path, ep.body, withBearer(inst.AppToken))
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusForbidden {
				body := decodeJSON(t, resp)
				t.Fatalf("scope %q should grant access to %s, got 403: %v", ep.scope, ep.path, body)
			}
		})
	}

	// App with hypothetical OLD scope names (if someone stored them wrong).
	appOld := createTestApp(t, env.store, env.user.ID, "Old Scope App", "old-scope-app",
		[]string{"send_message", "read_contacts", "read_bot"})
	instOld := installTestApp(t, env.store, appOld.ID, bot.ID)

	for _, ep := range endpoints {
		t.Run("old scope format denied for "+ep.path, func(t *testing.T) {
			resp := doJSON(t, env.ts, ep.method, ep.path, ep.body, withBearer(instOld.AppToken))
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusForbidden {
				t.Errorf("old scope format should be denied for %s, got %d", ep.path, resp.StatusCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test: App list endpoint returns correct data
// ---------------------------------------------------------------------------

func TestAppAPI_List(t *testing.T) {
	env := setupTestEnv(t)

	// Create some apps.
	createTestApp(t, env.store, env.user.ID, "App A", "app-aaa", []string{"message:write"})
	createTestApp(t, env.store, env.user.ID, "App B", "app-bbb", []string{"bot:read"})

	t.Run("list own apps", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps", nil, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var apps []map[string]any
		json.NewDecoder(resp.Body).Decode(&apps)
		if len(apps) < 2 {
			t.Errorf("expected at least 2 apps, got %d", len(apps))
		}
	})
}

// ---------------------------------------------------------------------------
// Test: CORS headers
// ---------------------------------------------------------------------------

func TestCORSHeaders(t *testing.T) {
	env := setupTestEnv(t)

	req, _ := http.NewRequest("OPTIONS", env.ts.URL+"/api/info", nil)
	req.Header.Set("Origin", "http://example.com")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Errorf("OPTIONS expected 204, got %d", resp.StatusCode)
	}
	if v := resp.Header.Get("Access-Control-Allow-Origin"); v != "http://example.com" {
		t.Errorf("ACAO = %q, want %q", v, "http://example.com")
	}
	if v := resp.Header.Get("Access-Control-Allow-Credentials"); v != "true" {
		t.Errorf("ACAC = %q, want %q", v, "true")
	}
}

// ---------------------------------------------------------------------------
// Test: Bot API unknown endpoint returns 404
// ---------------------------------------------------------------------------

func TestBotAPI_UnknownEndpoint(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "404-bot")
	app := createTestApp(t, env.store, env.user.ID, "404 App", "four-oh-four", []string{"message:write"})
	inst := installTestApp(t, env.store, app.ID, bot.ID)

	resp := doJSON(t, env.ts, "GET", "/bot/v1/nonexistent", nil, withBearer(inst.AppToken))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

// ---------------------------------------------------------------------------
// Test: Listing submission validation
// ---------------------------------------------------------------------------

func TestAppAPI_RequestListingValidation(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("missing required fields rejected", func(t *testing.T) {
		// Create a minimal app (missing description, readme, version, webhook, events/tools, scopes).
		app := createTestApp(t, env.store, env.user.ID, "Bare App", "bare-app", nil)

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/request-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		msg, _ := body["error"].(string)
		// Should mention multiple missing fields.
		for _, want := range []string{"description", "readme", "version", "webhook_url", "event", "scope"} {
			if !containsSubstring(msg, want) {
				t.Errorf("error message %q should mention %q", msg, want)
			}
		}
	})

	t.Run("complete app can be listed", func(t *testing.T) {
		tools, _ := json.Marshal([]map[string]string{{"name": "ping", "description": "ping"}})
		events, _ := json.Marshal([]string{"command"})
		scopes, _ := json.Marshal([]string{"message:write"})
		app, err := env.store.CreateApp(&store.App{
			OwnerID:     env.user.ID,
			Name:        "Complete App",
			Slug:        "complete-app",
			Description: "A complete app",
			Readme:      "# Complete App",
			Version:     "1.0.0",
			WebhookURL:  "https://example.com/webhook",
			Tools:       tools,
			Events:      events,
			Scopes:      scopes,
		})
		if err != nil {
			t.Fatalf("CreateApp: %v", err)
		}
		// Mark webhook as verified.
		_ = env.store.SetAppWebhookVerified(app.ID, true)

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/request-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		got, _ := env.store.GetApp(app.ID)
		if got.Listing != "pending" {
			t.Errorf("listing = %q, want %q", got.Listing, "pending")
		}
	})

	t.Run("unverified webhook allowed", func(t *testing.T) {
		// Webhook verification is no longer required for listing —
		// apps with OAuth/PKCE don't need it, and verification can happen post-listing.
		tools, _ := json.Marshal([]map[string]string{{"name": "ping", "description": "ping"}})
		scopes, _ := json.Marshal([]string{"message:write"})
		app, _ := env.store.CreateApp(&store.App{
			OwnerID:     env.user.ID,
			Name:        "Unverified Webhook App",
			Slug:        "unverified-webhook-app",
			Description: "has unverified webhook",
			Readme:      "# App",
			Version:     "1.0.0",
			WebhookURL:  "https://example.com/webhook",
			Tools:       tools,
			Scopes:      scopes,
		})

		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/request-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 for unverified webhook, got %d", resp.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Pending freeze (cannot modify core fields while pending)
// ---------------------------------------------------------------------------

func TestAppAPI_PendingFreeze(t *testing.T) {
	env := setupTestEnv(t)

	// Create a complete app and set it to pending via store directly.
	tools, _ := json.Marshal([]map[string]string{{"name": "ping", "description": "ping"}})
	events, _ := json.Marshal([]string{"command"})
	scopes, _ := json.Marshal([]string{"message:write"})
	app, _ := env.store.CreateApp(&store.App{
		OwnerID:     env.user.ID,
		Name:        "Freeze Test App",
		Slug:        "freeze-test-app",
		Description: "for freeze test",
		Readme:      "# Freeze",
		Version:     "1.0.0",
		WebhookURL:  "https://example.com/webhook",
		Tools:       tools,
		Events:      events,
		Scopes:      scopes,
	})
	_ = env.store.SetAppWebhookVerified(app.ID, true)
	_ = env.store.RequestListing(app.ID)

	t.Run("core field update blocked during pending", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+app.ID, map[string]any{
			"tools": []map[string]string{{"name": "new-tool", "description": "new"}},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 for core field update during pending, got %d", resp.StatusCode)
		}
	})

	t.Run("cosmetic update allowed during pending", func(t *testing.T) {
		newDesc := "Updated description"
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+app.ID, map[string]any{
			"description": &newDesc,
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200 for cosmetic update during pending, got %d: %v", resp.StatusCode, body)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Withdraw listing request
// ---------------------------------------------------------------------------

func TestAppAPI_WithdrawListing(t *testing.T) {
	env := setupTestEnv(t)

	app, _ := env.store.CreateApp(&store.App{
		OwnerID:     env.user.ID,
		Name:        "Withdraw Test App",
		Slug:        "withdraw-api-test",
		Description: "for withdraw test",
	})
	_ = env.store.RequestListing(app.ID)

	t.Run("withdraw pending listing", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/withdraw-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		got, _ := env.store.GetApp(app.ID)
		if got.Listing != "unlisted" {
			t.Errorf("listing = %q, want %q", got.Listing, "unlisted")
		}
	})

	t.Run("withdraw non-pending fails", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/withdraw-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 for non-pending withdraw, got %d", resp.StatusCode)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Auto-revert to pending on core change for listed apps
// ---------------------------------------------------------------------------

func TestAppAPI_AutoRevertOnCoreChange(t *testing.T) {
	env := setupTestEnv(t)

	tools, _ := json.Marshal([]map[string]string{{"name": "ping", "description": "ping"}})
	scopes, _ := json.Marshal([]string{"message:write"})
	app, _ := env.store.CreateApp(&store.App{
		OwnerID:     env.user.ID,
		Name:        "Revert Test App",
		Slug:        "revert-test-app",
		Description: "for revert test",
		Tools:       tools,
		Scopes:      scopes,
	})
	// Set to listed directly via store.
	_ = env.store.SetListing(app.ID, "listed")

	t.Run("core change reverts listed to pending", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+app.ID, map[string]any{
			"scopes": []string{"message:write", "contact:read"},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		got, _ := env.store.GetApp(app.ID)
		if got.Listing != "pending" {
			t.Errorf("listing = %q, want %q after core change", got.Listing, "pending")
		}
	})

	t.Run("cosmetic change does not revert listed", func(t *testing.T) {
		// Reset to listed.
		_ = env.store.SetListing(app.ID, "listed")

		newName := "Still Listed App"
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+app.ID, map[string]any{
			"name": &newName,
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		got, _ := env.store.GetApp(app.ID)
		if got.Listing != "listed" {
			t.Errorf("listing = %q, want %q (cosmetic change should not revert)", got.Listing, "listed")
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Full App listing lifecycle (end-to-end)
// ---------------------------------------------------------------------------

func TestAppAPI_FullListingLifecycle(t *testing.T) {
	env := setupTestEnv(t)

	// Stand up a test HTTP server that responds to webhook verification challenges.
	webhookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		json.NewDecoder(r.Body).Decode(&req)
		if req["type"] == "url_verification" {
			json.NewEncoder(w).Encode(map[string]string{"challenge": req["challenge"].(string)})
			return
		}
		w.WriteHeader(200)
	}))
	defer webhookServer.Close()

	// Step 1: Create app via API, then set version via store (version is not
	// exposed in the create-app API but is required for listing submission).
	tools, _ := json.Marshal([]map[string]string{{"name": "ping", "description": "pong"}})
	events, _ := json.Marshal([]string{"command"})
	scopes, _ := json.Marshal([]string{"message:write"})

	var appID string
	t.Run("1_create_app", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps", map[string]any{
			"name":        "Lifecycle App",
			"slug":        "lifecycle-app",
			"description": "An app for lifecycle testing",
			"readme":      "# Lifecycle App\nFull lifecycle test.",
			"tools":       json.RawMessage(tools),
			"events":      json.RawMessage(events),
			"scopes":      json.RawMessage(scopes),
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		appID, _ = body["id"].(string)
		if appID == "" {
			t.Fatal("created app has no id")
		}
	})
	if appID == "" {
		t.Fatal("step 1 failed, cannot continue")
	}

	// Delete and recreate via store to include the version field
	// (no API/store method exposes version update independently).
	_ = env.store.DeleteApp(appID)
	storeApp, err := env.store.CreateApp(&store.App{
		OwnerID:     env.user.ID,
		Name:        "Lifecycle App",
		Slug:        "lifecycle-app",
		Description: "An app for lifecycle testing",
		Readme:      "# Lifecycle App\nFull lifecycle test.",
		Version:     "1.0.0",
		Tools:       tools,
		Events:      events,
		Scopes:      scopes,
	})
	if err != nil {
		t.Fatalf("recreate app with version: %v", err)
	}
	appID = storeApp.ID

	// Step 2: Set webhook_url.
	t.Run("2_set_webhook_url", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+appID, map[string]any{
			"webhook_url": &webhookServer.URL,
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		app, _ := env.store.GetApp(appID)
		if app.WebhookURL != webhookServer.URL {
			t.Fatalf("webhook_url = %q, want %q", app.WebhookURL, webhookServer.URL)
		}
	})

	// Step 3: Verify webhook URL.
	t.Run("3_verify_webhook", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/verify-url", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		app, _ := env.store.GetApp(appID)
		if !app.WebhookVerified {
			t.Fatal("webhook should be verified after successful challenge")
		}
	})

	// Step 4: Submit for listing.
	t.Run("4_submit_for_listing", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/request-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		app, _ := env.store.GetApp(appID)
		if app.Listing != "pending" {
			t.Fatalf("listing = %q, want %q", app.Listing, "pending")
		}
	})

	// Step 5: Try modifying core field while pending — should be blocked.
	t.Run("5_core_field_blocked_while_pending", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+appID, map[string]any{
			"tools": []map[string]string{{"name": "new-tool", "description": "blocked"}},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("expected 403 for core field update during pending, got %d", resp.StatusCode)
		}
	})

	// Step 6: Admin review — reject.
	t.Run("6_admin_reject", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/admin/apps/"+appID+"/review-listing", map[string]any{
			"approve": false,
			"reason":  "needs work",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
	})

	// Step 7: Verify app is rejected.
	t.Run("7_verify_rejected", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+appID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if listing, _ := body["listing"].(string); listing != "rejected" {
			t.Fatalf("listing = %q, want %q", listing, "rejected")
		}
		if reason, _ := body["listing_reject_reason"].(string); reason != "needs work" {
			t.Fatalf("listing_reject_reason = %q, want %q", reason, "needs work")
		}
	})

	// Step 8: Fix and resubmit.
	t.Run("8_resubmit_after_rejection", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/request-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		app, _ := env.store.GetApp(appID)
		if app.Listing != "pending" {
			t.Fatalf("listing = %q, want %q", app.Listing, "pending")
		}
	})

	// Step 9: Admin review — approve.
	t.Run("9_admin_approve", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/admin/apps/"+appID+"/review-listing", map[string]any{
			"approve": true,
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
	})

	// Step 10: Verify app is listed.
	t.Run("10_verify_listed", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+appID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if listing, _ := body["listing"].(string); listing != "listed" {
			t.Fatalf("listing = %q, want %q", listing, "listed")
		}
	})

	// Step 11: Check registry output.
	t.Run("11_registry_output", func(t *testing.T) {
		// Enable the registry.
		if err := env.store.SetConfig("registry.enabled", "true"); err != nil {
			t.Fatalf("SetConfig: %v", err)
		}

		// Registry endpoint is public — no auth needed.
		resp := doJSON(t, env.ts, "GET", "/api/registry/v1/apps.json", nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var manifest struct {
			Version int `json:"version"`
			Apps    []struct {
				Slug string `json:"slug"`
				Name string `json:"name"`
			} `json:"apps"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
			t.Fatalf("decode manifest: %v", err)
		}
		if manifest.Version != 1 {
			t.Errorf("manifest version = %d, want 1", manifest.Version)
		}

		found := false
		for _, a := range manifest.Apps {
			if a.Slug == "lifecycle-app" {
				found = true
				if a.Name != "Lifecycle App" {
					t.Errorf("app name = %q, want %q", a.Name, "Lifecycle App")
				}
				break
			}
		}
		if !found {
			t.Fatalf("listed app not found in registry manifest; apps = %+v", manifest.Apps)
		}
	})

	// -----------------------------------------------------------------------
	// Step 11b–11f: Installation verification steps (app is listed at this point)
	// -----------------------------------------------------------------------

	// Create a second user + session for non-owner operations.
	user2, err := env.store.CreateUserFull("user2", "user2@test.com", "User Two", "hashed", string(store.RoleMember))
	if err != nil {
		t.Fatalf("CreateUserFull(user2): %v", err)
	}
	_ = env.store.UpdateUserStatus(user2.ID, store.StatusActive)
	session2Token, err := auth.CreateSession(env.store, user2.ID)
	if err != nil {
		t.Fatalf("CreateSession(user2): %v", err)
	}
	cookie2 := &http.Cookie{Name: "session", Value: session2Token}

	// Step 11b: Another user can see the listed app.
	t.Run("11b_listed_app_visible_to_non_owner", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+appID, nil, withCookie(cookie2))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		if name, _ := body["name"].(string); name != "Lifecycle App" {
			t.Errorf("name = %q, want %q", name, "Lifecycle App")
		}
	})

	// Create a bot for user2 (needed for installation steps).
	bot2 := createTestBot(t, env.store, user2.ID, "User2 Bot")

	// Step 11c: Another user installs the listed app.
	var installationID, appToken string
	t.Run("11c_non_owner_installs_listed_app", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/install", map[string]any{
			"bot_id": bot2.ID,
			"handle": "test-install",
		}, withCookie(cookie2))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		installationID, _ = body["id"].(string)
		appToken, _ = body["app_token"].(string)
		if installationID == "" {
			t.Fatal("installation has no id")
		}
		if appToken == "" {
			t.Fatal("installation has no app_token")
		}
	})
	if appToken == "" {
		t.Fatal("step 11c failed (no token), cannot continue")
	}

	// Step 11d: Installation works — verify token authenticates.
	// The app has message:write scope, so test POST /bot/v1/message/send.
	// The bot isn't actually connected so we won't get 200, but auth/scope
	// checks should pass (i.e. we must NOT get 401 or 403).
	t.Run("11d_app_token_authenticates", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/bot/v1/message/send",
			map[string]string{"content": "hello from installed app"},
			withBearer(appToken))
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			t.Fatal("expected token to be valid, got 401")
		}
		if resp.StatusCode == http.StatusForbidden {
			body := decodeJSON(t, resp)
			t.Fatalf("expected scope check to pass for message:write, got 403: %v", body)
		}
		// Any other status (500, 503, etc.) is acceptable — the bot isn't running.
	})

	// Step 11e: Registry manifest includes full app details, excludes secrets.
	t.Run("11e_registry_manifest_details", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/registry/v1/apps.json", nil)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		// Decode the full manifest to check individual app fields.
		var raw struct {
			Apps []map[string]any `json:"apps"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode manifest: %v", err)
		}

		var found map[string]any
		for _, a := range raw.Apps {
			if slug, _ := a["slug"].(string); slug == "lifecycle-app" {
				found = a
				break
			}
		}
		if found == nil {
			t.Fatal("lifecycle-app not found in manifest")
		}

		// Verify essential fields are present.
		if v, _ := found["webhook_url"].(string); v == "" {
			t.Error("manifest app missing webhook_url")
		}
		if v, ok := found["tools"]; !ok || v == nil {
			t.Error("manifest app missing tools")
		}
		if v, ok := found["events"]; !ok || v == nil {
			t.Error("manifest app missing events")
		}
		if v, ok := found["scopes"]; !ok || v == nil {
			t.Error("manifest app missing scopes")
		}

		// Verify webhook_secret is NOT in the manifest.
		if _, ok := found["webhook_secret"]; ok {
			t.Error("manifest app must not include webhook_secret")
		}
	})

	// Step 11f: Install endpoint works for marketplace-style install.
	// Admin user installs the same listed app to their own bot via
	// POST /api/apps/{id}/install with bot_id.
	adminBot := createTestBot(t, env.store, env.user.ID, "Admin Lifecycle Bot")
	t.Run("11f_install_endpoint", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/install", map[string]any{
			"bot_id": adminBot.ID,
			"handle": "unified-test",
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		if id, _ := body["id"].(string); id == "" {
			t.Error("unified install returned no installation id")
		}
		if tok, _ := body["app_token"].(string); tok == "" {
			t.Error("unified install returned no app_token")
		}
	})

	// Step 12: Update core field on listed app — should auto-revert to pending.
	t.Run("12_core_change_reverts_listed", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/api/apps/"+appID, map[string]any{
			"scopes": []string{"message:write", "contact:read"},
		}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
	})

	// Step 13: Verify reverted to pending.
	t.Run("13_verify_reverted_to_pending", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+appID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if listing, _ := body["listing"].(string); listing != "pending" {
			t.Fatalf("listing = %q, want %q after core change", listing, "pending")
		}
	})

	// Step 14: Withdraw listing.
	t.Run("14_withdraw_listing", func(t *testing.T) {
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+appID+"/withdraw-listing", nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
	})

	// Step 15: Verify unlisted.
	t.Run("15_verify_unlisted", func(t *testing.T) {
		resp := doJSON(t, env.ts, "GET", "/api/apps/"+appID, nil,
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		body := decodeJSON(t, resp)
		if listing, _ := body["listing"].(string); listing != "unlisted" {
			t.Fatalf("listing = %q, want %q", listing, "unlisted")
		}
	})
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && contains(s, sub))
}

// ---------------------------------------------------------------------------
// Test: Bot API UpdateTools
// ---------------------------------------------------------------------------

func TestBotAPI_UpdateTools(t *testing.T) {
	env := setupTestEnv(t)

	// Create app with tools:write scope
	app := createTestApp(t, env.store, env.user.ID, "Dynamic Tools App", "dynamic-tools-app", []string{"tools:write"})

	// Create bot and install
	bot := createTestBot(t, env.store, env.user.ID, "tools-bot")
	inst := installTestApp(t, env.store, app.ID, bot.ID)

	t.Run("update tools succeeds with scope", func(t *testing.T) {
		newTools := []map[string]any{
			{"name": "hn", "description": "HackerNews top", "command": "hn"},
			{"name": "weather", "description": "Check weather", "command": "weather"},
		}
		toolsJSON, _ := json.Marshal(newTools)

		resp := doJSON(t, env.ts, "PUT", "/bot/v1/app/tools",
			map[string]any{"tools": json.RawMessage(toolsJSON)},
			withBearer(inst.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		if body["tool_count"] != float64(2) {
			t.Errorf("tool_count = %v, want 2", body["tool_count"])
		}

		// Verify tools were actually updated
		updated, err := env.store.GetApp(app.ID)
		if err != nil {
			t.Fatalf("GetApp: %v", err)
		}
		var tools []map[string]any
		json.Unmarshal(updated.Tools, &tools)
		if len(tools) != 2 {
			t.Errorf("stored tools count = %d, want 2", len(tools))
		}
	})

	t.Run("update tools fails without scope", func(t *testing.T) {
		// Create app WITHOUT tools:write
		app2 := createTestApp(t, env.store, env.user.ID, "No Scope App", "no-scope-app", []string{"message:write"})
		inst2 := installTestApp(t, env.store, app2.ID, bot.ID)

		resp := doJSON(t, env.ts, "PUT", "/bot/v1/app/tools",
			map[string]any{"tools": json.RawMessage(`[{"name":"test"}]`)},
			withBearer(inst2.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("update tools fails for marketplace app", func(t *testing.T) {
		// Create marketplace app (has registry set)
		mktScopes, _ := json.Marshal([]string{"tools:write"})
		mktApp, err := env.store.CreateApp(&store.App{
			OwnerID:  env.user.ID,
			Name:     "Marketplace App",
			Slug:     "mkt-tools-app",
			Scopes:   mktScopes,
			Registry: "https://some-registry.com",
		})
		if err != nil {
			t.Fatalf("CreateApp: %v", err)
		}
		mktInst := installTestApp(t, env.store, mktApp.ID, bot.ID)

		resp := doJSON(t, env.ts, "PUT", "/bot/v1/app/tools",
			map[string]any{"tools": json.RawMessage(`[{"name":"test"}]`)},
			withBearer(mktInst.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("update tools fails for builtin app", func(t *testing.T) {
		// Create builtin app
		biScopes, _ := json.Marshal([]string{"tools:write"})
		biApp, err := env.store.CreateApp(&store.App{
			OwnerID:  env.user.ID,
			Name:     "Builtin App",
			Slug:     "bi-tools-app",
			Scopes:   biScopes,
			Registry: "builtin",
		})
		if err != nil {
			t.Fatalf("CreateApp: %v", err)
		}
		biInst := installTestApp(t, env.store, biApp.ID, bot.ID)

		resp := doJSON(t, env.ts, "PUT", "/bot/v1/app/tools",
			map[string]any{"tools": json.RawMessage(`[{"name":"test"}]`)},
			withBearer(biInst.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != 403 {
			t.Fatalf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid tools format rejected", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/bot/v1/app/tools",
			map[string]any{"tools": "not an array"},
			withBearer(inst.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
	})
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Test: Scope snapshot at install time (Slack model)
// ---------------------------------------------------------------------------

func TestScopeSnapshotAtInstall(t *testing.T) {
	env := setupTestEnv(t)

	// Create app with initial scopes
	app := createTestApp(t, env.store, env.user.ID, "Scope Test App", "scope-test",
		[]string{"message:write", "message:read"})

	bot := createTestBot(t, env.store, env.user.ID, "scope-bot-2")

	t.Run("installation gets app scopes snapshot", func(t *testing.T) {
		// Install without specifying scopes via the API
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install",
			map[string]any{"bot_id": bot.ID, "handle": "scope-test"},
			withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201, got %d: %v", resp.StatusCode, body)
		}

		body := decodeJSON(t, resp)
		instID, _ := body["id"].(string)

		// Verify installation has scopes
		inst, err := env.store.GetInstallation(instID)
		if err != nil {
			t.Fatalf("GetInstallation: %v", err)
		}
		var instScopes []string
		json.Unmarshal(inst.Scopes, &instScopes)
		if len(instScopes) != 2 {
			t.Fatalf("expected 2 scopes, got %d: %v", len(instScopes), instScopes)
		}
	})

	t.Run("app scope change does not affect existing installation", func(t *testing.T) {
		// Update app scopes (add tools:write)
		newScopes, _ := json.Marshal([]string{"message:write", "message:read", "tools:write"})
		env.store.UpdateApp(app.ID, app.Name, app.Description, app.Icon, app.IconURL,
			app.Homepage, app.OAuthSetupURL, app.OAuthRedirectURL, app.ConfigSchema,
			app.Tools, app.Events, newScopes)

		// Existing installation should still have old scopes
		installations, _ := env.store.ListInstallationsByApp(app.ID)
		for _, inst := range installations {
			var instScopes []string
			json.Unmarshal(inst.Scopes, &instScopes)
			for _, s := range instScopes {
				if s == "tools:write" {
					t.Error("existing installation should not have new scope tools:write")
				}
			}
		}
	})

	t.Run("reauthorize updates scopes", func(t *testing.T) {
		installations, _ := env.store.ListInstallationsByApp(app.ID)
		inst := installations[0]

		resp := doJSON(t, env.ts, "POST",
			"/api/apps/"+app.ID+"/installations/"+inst.ID+"/reauthorize",
			nil, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}

		// Now should have new scope
		updated, _ := env.store.GetInstallation(inst.ID)
		var updatedScopes []string
		json.Unmarshal(updated.Scopes, &updatedScopes)
		found := false
		for _, s := range updatedScopes {
			if s == "tools:write" {
				found = true
			}
		}
		if !found {
			t.Error("reauthorized installation should have tools:write")
		}
	})

	t.Run("widening scopes beyond app is rejected", func(t *testing.T) {
		bot2 := createTestBot(t, env.store, env.user.ID, "widen-bot")
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install",
			map[string]any{
				"bot_id": bot2.ID,
				"handle": "scope-widen-test",
				"scopes": []string{"message:write", "message:read", "tools:write", "admin:write"},
			}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for widened scopes, got %d", resp.StatusCode)
		}
	})

	t.Run("narrowing scopes is allowed", func(t *testing.T) {
		bot3 := createTestBot(t, env.store, env.user.ID, "narrow-bot")
		resp := doJSON(t, env.ts, "POST", "/api/apps/"+app.ID+"/install",
			map[string]any{
				"bot_id": bot3.ID,
				"handle": "scope-narrow-test",
				"scopes": []string{"message:write"},
			}, withCookie(env.cookie))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 201 for narrowed scopes, got %d: %v", resp.StatusCode, body)
		}

		body := decodeJSON(t, resp)
		instID := body["id"].(string)
		inst, _ := env.store.GetInstallation(instID)
		var instScopes []string
		json.Unmarshal(inst.Scopes, &instScopes)
		if len(instScopes) != 1 || instScopes[0] != "message:write" {
			t.Errorf("expected [message:write], got %v", instScopes)
		}
	})
}

// ---------------------------------------------------------------------------
// Test: Bot API update installation tools
// ---------------------------------------------------------------------------

func TestBotAPI_UpdateInstallationTools(t *testing.T) {
	env := setupTestEnv(t)
	bot := createTestBot(t, env.store, env.user.ID, "inst-tools-bot")

	// App with tools:write scope.
	appWithScope := createTestApp(t, env.store, env.user.ID, "tools-app", "tools-app",
		[]string{"tools:write"})
	instWithScope := installTestApp(t, env.store, appWithScope.ID, bot.ID)

	// App without tools:write scope.
	appNoScope := createTestApp(t, env.store, env.user.ID, "no-tools-app", "no-tools-app",
		[]string{"message:write"})
	instNoScope := installTestApp(t, env.store, appNoScope.ID, bot.ID)

	t.Run("update installation tools succeeds", func(t *testing.T) {
		tools := []map[string]string{
			{"name": "hn", "description": "Hacker News", "command": "hn"},
			{"name": "weather", "description": "Weather report", "command": "weather"},
		}
		resp := doJSON(t, env.ts, "PUT", "/bot/v1/installation/tools",
			map[string]any{"tools": tools},
			withBearer(instWithScope.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body := decodeJSON(t, resp)
			t.Fatalf("expected 200, got %d: %v", resp.StatusCode, body)
		}
		body := decodeJSON(t, resp)
		if body["ok"] != true {
			t.Errorf("expected ok=true, got %v", body["ok"])
		}
		if tc, _ := body["tool_count"].(float64); tc != 2 {
			t.Errorf("expected tool_count=2, got %v", tc)
		}

		// Verify tools stored on installation (not app).
		inst, err := env.store.GetInstallation(instWithScope.ID)
		if err != nil {
			t.Fatalf("GetInstallation: %v", err)
		}
		var storedTools []store.AppTool
		if err := json.Unmarshal(inst.Tools, &storedTools); err != nil {
			t.Fatalf("unmarshal tools: %v", err)
		}
		if len(storedTools) != 2 {
			t.Errorf("expected 2 tools on installation, got %d", len(storedTools))
		}

		// App-level tools should remain unchanged (empty).
		app, _ := env.store.GetApp(appWithScope.ID)
		var appTools []store.AppTool
		json.Unmarshal(app.Tools, &appTools)
		if len(appTools) != 0 {
			t.Errorf("app tools should be empty, got %d", len(appTools))
		}
	})

	t.Run("missing tools:write scope returns 403", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/bot/v1/installation/tools",
			map[string]any{"tools": []any{}},
			withBearer(instNoScope.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid tools format returns 400", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/bot/v1/installation/tools",
			map[string]any{"tools": "not-an-array"},
			withBearer(instWithScope.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	t.Run("missing tools field returns 400", func(t *testing.T) {
		resp := doJSON(t, env.ts, "PUT", "/bot/v1/installation/tools",
			map[string]any{},
			withBearer(instWithScope.AppToken))
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})
}
