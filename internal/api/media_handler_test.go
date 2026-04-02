package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/openilink/openilink-hub/internal/bot"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/provider"
	"github.com/openilink/openilink-hub/internal/store"
	"github.com/openilink/openilink-hub/internal/store/sqlite"
)

// ---------------------------------------------------------------------------
// mock provider (minimal, for media handler tests)
// ---------------------------------------------------------------------------

type mockProvider struct{}

func (m *mockProvider) Name() string                          { return "mock" }
func (m *mockProvider) Start(context.Context, provider.StartOptions) error { return nil }
func (m *mockProvider) Stop()                                 {}
func (m *mockProvider) Send(context.Context, provider.OutboundMessage) (string, error) {
	return "", nil
}
func (m *mockProvider) SendTyping(context.Context, string, string, bool) error { return nil }
func (m *mockProvider) GetConfig(context.Context, string, string) (*provider.BotConfig, error) {
	return nil, nil
}
func (m *mockProvider) DownloadMedia(_ context.Context, _ *provider.Media) ([]byte, error) {
	// Return a tiny PNG so Content-Type detection works
	return []byte("\x89PNG\r\n\x1a\n" + "fake-image-data"), nil
}
func (m *mockProvider) DownloadVoice(context.Context, *provider.Media, int) ([]byte, error) {
	return nil, nil
}
func (m *mockProvider) Status() string { return "connected" }

// ---------------------------------------------------------------------------
// mock object store
// ---------------------------------------------------------------------------

type mockObjectStore struct {
	mu   sync.Mutex
	data map[string][]byte
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{data: make(map[string][]byte)}
}

func (m *mockObjectStore) Put(_ context.Context, key, _ string, data []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = data
	return key, nil
}

func (m *mockObjectStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.data[key]
	if !ok {
		return nil, io.EOF
	}
	return d, nil
}

func (m *mockObjectStore) URL(key string) string { return "/media/" + key }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// setupMediaTestEnv creates a test environment with BotManager and a running
// mock bot instance so the media handlers can resolve the bot.
func setupMediaTestEnv(t *testing.T) (*httptest.Server, store.Store, *store.Bot, *store.AppInstallation) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("sqlite.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	u, err := s.CreateUserFull("testadmin", "", "Test Admin", "hashed", store.RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUserFull: %v", err)
	}
	_ = s.UpdateUserStatus(u.ID, store.StatusActive)

	// Create a bot.
	b, err := s.CreateBot(u.ID, "mediabot", "mock", "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	// Register mock provider and create a Manager with a running instance.
	provider.Register("mock", func() provider.Provider { return &mockProvider{} })
	mgr := bot.NewManager(s, nil, nil, nil, "http://localhost")
	if err := mgr.StartBot(context.Background(), b); err != nil {
		t.Fatalf("StartBot: %v", err)
	}
	t.Cleanup(mgr.StopAll)

	// Create an app and install it on the bot.
	scopesJSON, _ := json.Marshal([]string{"message:read"})
	app, err := s.CreateApp(&store.App{
		OwnerID: u.ID,
		Name:    "MediaTestApp",
		Slug:    "media-test-app",
		Scopes:  scopesJSON,
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}
	inst, err := s.InstallApp(app.ID, b.ID)
	if err != nil {
		t.Fatalf("InstallApp: %v", err)
	}
	// Grant message:read scope so Bearer auth passes scope check.
	if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), scopesJSON, true); err != nil {
		t.Fatalf("UpdateInstallation: %v", err)
	}
	// Re-read to pick up updated scopes.
	inst, err = s.GetInstallation(inst.ID)
	if err != nil {
		t.Fatalf("GetInstallation: %v", err)
	}

	// Set up object store with a test file.
	objStore := newMockObjectStore()
	_, _ = objStore.Put(context.Background(), b.ID+"/2024-01-01/test.png", "image/png",
		[]byte("\x89PNG\r\n\x1a\n"+"fake-image-data"))

	srv := &Server{
		Store:       s,
		BotManager:  mgr,
		Config:      &config.Config{RPOrigin: "http://localhost"},
		OAuthStates: newOAuthStateStore(),
		ObjectStore: objStore,
	}

	handler := srv.Handler()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	return ts, s, b, inst
}

// ---------------------------------------------------------------------------
// Tests: CDN media proxy (handleChannelMedia)
// ---------------------------------------------------------------------------

func TestChannelMedia_BearerAppToken(t *testing.T) {
	ts, _, _, inst := setupMediaTestEnv(t)

	resp, err := doMediaReq(t, ts, "/api/v1/channels/media?eqp=test&aes=test123",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	// The mock provider returns data, so we expect 200 (not 401).
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelMedia_InvalidBearer(t *testing.T) {
	ts, _, _, _ := setupMediaTestEnv(t)

	resp, err := doMediaReq(t, ts, "/api/v1/channels/media?eqp=test&aes=test123",
		"Bearer invalid-token-xxx")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelMedia_NoAuth(t *testing.T) {
	ts, _, _, _ := setupMediaTestEnv(t)

	resp, err := doMediaReq(t, ts, "/api/v1/channels/media?eqp=test&aes=test123", "")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Tests: Storage media proxy (handleMediaProxy)
// ---------------------------------------------------------------------------

func TestMediaProxy_BearerAppToken(t *testing.T) {
	ts, _, b, inst := setupMediaTestEnv(t)

	resp, err := doMediaReq(t, ts, "/api/v1/media/"+b.ID+"/2024-01-01/test.png",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMediaProxy_InvalidBearer(t *testing.T) {
	ts, _, b, _ := setupMediaTestEnv(t)

	resp, err := doMediaReq(t, ts, "/api/v1/media/"+b.ID+"/2024-01-01/test.png",
		"Bearer invalid-token-xxx")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMediaProxy_BearerWrongBot(t *testing.T) {
	ts, s, _, inst := setupMediaTestEnv(t)

	// Create a second bot that the installation does NOT belong to.
	u, _ := s.CreateUserFull("other", "", "Other", "hashed", store.RoleAdmin)
	_ = s.UpdateUserStatus(u.ID, store.StatusActive)
	b2, err := s.CreateBot(u.ID, "otherbot", "mock", "", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("CreateBot: %v", err)
	}

	// Request media for b2 using inst's token (which belongs to a different bot).
	resp, err := doMediaReq(t, ts, "/api/v1/media/"+b2.ID+"/2024-01-01/test.png",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// Tests: disabled installation and missing scope
// ---------------------------------------------------------------------------

func TestChannelMedia_BearerDisabledInstallation(t *testing.T) {
	ts, s, _, inst := setupMediaTestEnv(t)

	// Disable the installation.
	scopesJSON, _ := json.Marshal([]string{"message:read"})
	if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), scopesJSON, false); err != nil {
		t.Fatalf("UpdateInstallation: %v", err)
	}

	resp, err := doMediaReq(t, ts, "/api/v1/channels/media?eqp=test&aes=test123",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for disabled installation, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestChannelMedia_BearerNoScope(t *testing.T) {
	ts, s, _, inst := setupMediaTestEnv(t)

	// Clear scopes on the installation.
	if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), json.RawMessage("[]"), true); err != nil {
		t.Fatalf("UpdateInstallation: %v", err)
	}

	resp, err := doMediaReq(t, ts, "/api/v1/channels/media?eqp=test&aes=test123",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for missing scope, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMediaProxy_BearerDisabledInstallation(t *testing.T) {
	ts, s, b, inst := setupMediaTestEnv(t)

	// Disable the installation.
	scopesJSON, _ := json.Marshal([]string{"message:read"})
	if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), scopesJSON, false); err != nil {
		t.Fatalf("UpdateInstallation: %v", err)
	}

	resp, err := doMediaReq(t, ts, "/api/v1/media/"+b.ID+"/2024-01-01/test.png",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for disabled installation, got %d: %s", resp.StatusCode, string(body))
	}
}

func TestMediaProxy_BearerNoScope(t *testing.T) {
	ts, s, b, inst := setupMediaTestEnv(t)

	// Clear scopes on the installation.
	if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), json.RawMessage("[]"), true); err != nil {
		t.Fatalf("UpdateInstallation: %v", err)
	}

	resp, err := doMediaReq(t, ts, "/api/v1/media/"+b.ID+"/2024-01-01/test.png",
		"Bearer "+inst.AppToken)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 401 for missing scope, got %d: %s", resp.StatusCode, string(body))
	}
}

// ---------------------------------------------------------------------------
// helper
// ---------------------------------------------------------------------------

func doMediaReq(t *testing.T, ts *httptest.Server, path, authHeader string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequest("GET", ts.URL+path, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	return http.DefaultClient.Do(req)
}
