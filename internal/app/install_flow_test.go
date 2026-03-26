package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/openilink/openilink-hub/internal/store"
)

// --- Mock install DB that tracks UpdateAppWebhookURL and SetAppWebhookVerified calls ---

type mockInstallDB struct {
	mu              sync.Mutex
	updatedURL      string
	urlVerified     bool
	urlVerifiedID   string
	updateCallCount int
	verifyCallCount int
}

func (m *mockInstallDB) UpdateAppWebhookURL(id, requestURL string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updatedURL = requestURL
	m.updateCallCount++
	return nil
}

func (m *mockInstallDB) SetAppWebhookVerified(id string, verified bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.urlVerifiedID = id
	m.urlVerified = verified
	m.verifyCallCount++
	return nil
}

// --- Mock App server for install flow ---

// installCallback tracks what the mock App server receives on /callback.
type installCallback struct {
	InstallationID string `json:"installation_id"`
	AppToken       string `json:"app_token"`
	SigningSecret   string `json:"signing_secret"`
	BotID          string `json:"bot_id"`
	Handle         string `json:"handle"`
}

// mockInstallAppServer simulates an external App that accepts install
// notifications and responds with a request_url, plus handles url_verification.
type mockInstallAppServer struct {
	mu              sync.Mutex
	receivedInstall *installCallback
	returnURL       string
	returnStatus    int
	returnBody      string
	challenges      []string
	server          *httptest.Server
}

func newMockInstallAppServer(requestURL string) *mockInstallAppServer {
	m := &mockInstallAppServer{
		returnURL:    requestURL,
		returnStatus: http.StatusOK,
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.handler))
	// If requestURL is empty, default to the server's own URL + /hub/webhook
	if m.returnURL == "" {
		m.returnURL = m.server.URL + "/hub/webhook"
	}
	return m
}

func (m *mockInstallAppServer) handler(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	// Check for url_verification challenge
	var probe struct {
		Type      string `json:"type"`
		Challenge string `json:"challenge"`
	}
	if json.Unmarshal(body, &probe) == nil && probe.Type == "url_verification" {
		m.mu.Lock()
		m.challenges = append(m.challenges, probe.Challenge)
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"challenge": probe.Challenge})
		return
	}

	// Otherwise, it's an install callback
	var cb installCallback
	if err := json.Unmarshal(body, &cb); err == nil {
		m.mu.Lock()
		m.receivedInstall = &cb
		m.mu.Unlock()
	}

	if m.returnStatus != http.StatusOK {
		w.WriteHeader(m.returnStatus)
		if m.returnBody != "" {
			w.Write([]byte(m.returnBody))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if m.returnBody != "" {
		w.Write([]byte(m.returnBody))
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"request_url": m.returnURL})
}

func (m *mockInstallAppServer) close() {
	m.server.Close()
}

func (m *mockInstallAppServer) getReceivedInstall() *installCallback {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.receivedInstall
}

func (m *mockInstallAppServer) getChallenges() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]string, len(m.challenges))
	copy(cp, m.challenges)
	return cp
}

// --- simulateNotifyAppInstalled replicates the core logic of
// api.Server.notifyAppInstalled without requiring a full Server setup. ---

func simulateNotifyAppInstalled(
	client *http.Client,
	redirectURL string,
	appID string,
	signingSecret string,
	inst *store.AppInstallation,
	db *mockInstallDB,
) (requestURL string, err error) {
	payload, _ := json.Marshal(map[string]string{
		"installation_id": inst.ID,
		"app_token":       inst.AppToken,
		"signing_secret":  signingSecret,
		"bot_id":          inst.BotID,
		"handle":          inst.Handle,
	})

	resp, err := client.Post(redirectURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil
	}

	var result struct {
		RequestURL string `json:"request_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || result.RequestURL == "" {
		return "", nil
	}

	_ = db.UpdateAppWebhookURL(appID, result.RequestURL)
	return result.RequestURL, nil
}

// simulateAutoVerifyURL replicates the core logic of api.Server.autoVerifyURL.
func simulateAutoVerifyURL(
	client *http.Client,
	appID, requestURL string,
	db *mockInstallDB,
) bool {
	challenge := "test-verify-challenge-abc123"

	payload, _ := json.Marshal(map[string]any{
		"v":         1,
		"type":      "url_verification",
		"challenge": challenge,
	})

	resp, err := client.Post(requestURL, "application/json", bytes.NewReader(payload))
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var result struct {
		Challenge string `json:"challenge"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}
	if result.Challenge == challenge {
		_ = db.SetAppWebhookVerified(appID, true)
		return true
	}
	return false
}

// ==================== Test 1: Full install + notify flow ====================

func TestInstallFlow_FullNotifyAndVerify(t *testing.T) {
	mock := newMockInstallAppServer("")
	defer mock.close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	inst := &store.AppInstallation{
		ID:            "inst-001",
		AppID:         "app-001",
		BotID:         "bot-001",
		AppToken:      "tok_abc123",
		AppWebhookSecret: "sec_xyz789",
		Handle:        "echo-work",
	}

	// Step 1: Notify the App
	requestURL, err := simulateNotifyAppInstalled(client, mock.server.URL+"/callback", inst.AppID, "sec_xyz789", inst, db)
	if err != nil {
		t.Fatalf("notifyAppInstalled failed: %v", err)
	}
	if requestURL == "" {
		t.Fatal("expected request_url from App, got empty")
	}

	// Verify mock received the correct credentials
	received := mock.getReceivedInstall()
	if received == nil {
		t.Fatal("mock did not receive install callback")
	}
	if received.AppToken != "tok_abc123" {
		t.Errorf("app_token = %q, want %q", received.AppToken, "tok_abc123")
	}
	if received.SigningSecret != "sec_xyz789" {
		t.Errorf("signing_secret = %q, want %q", received.SigningSecret, "sec_xyz789")
	}
	if received.BotID != "bot-001" {
		t.Errorf("bot_id = %q, want %q", received.BotID, "bot-001")
	}
	if received.Handle != "echo-work" {
		t.Errorf("handle = %q, want %q", received.Handle, "echo-work")
	}

	// Verify DB was updated with request_url
	if db.updatedURL != requestURL {
		t.Errorf("db.updatedURL = %q, want %q", db.updatedURL, requestURL)
	}

	// Step 2: Auto-verify the URL
	verified := simulateAutoVerifyURL(client, inst.AppID, requestURL, db)
	if !verified {
		t.Fatal("auto verify should have succeeded")
	}
	if !db.urlVerified {
		t.Error("db.urlVerified should be true")
	}
	if db.urlVerifiedID != inst.AppID {
		t.Errorf("db.urlVerifiedID = %q, want %q", db.urlVerifiedID, inst.AppID)
	}

	// Verify the mock received the challenge
	challenges := mock.getChallenges()
	if len(challenges) == 0 {
		t.Fatal("mock did not receive any url_verification challenge")
	}
}

// ==================== Test 2: Notify with failed App (500) ====================

func TestInstallFlow_NotifyAppReturns500(t *testing.T) {
	mock := newMockInstallAppServer("")
	mock.returnStatus = http.StatusInternalServerError
	defer mock.close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	inst := &store.AppInstallation{
		ID:            "inst-002",
		AppID:         "app-002",
		BotID:         "bot-002",
		AppToken:      "tok_fail",
		AppWebhookSecret: "sec_fail",
		Handle:        "failbot",
	}

	requestURL, err := simulateNotifyAppInstalled(client, mock.server.URL+"/callback", inst.AppID, "sec_fail", inst, db)
	// Should not crash, should handle gracefully
	if err != nil {
		t.Fatalf("should not return error for 500, got: %v", err)
	}
	if requestURL != "" {
		t.Errorf("should return empty request_url for 500, got %q", requestURL)
	}
	if db.updateCallCount != 0 {
		t.Errorf("db.UpdateAppWebhookURL should not be called on failure, called %d times", db.updateCallCount)
	}
}

// ==================== Test 3: Notify with invalid response (no request_url) ====================

func TestInstallFlow_NotifyAppReturnsNoRequestURL(t *testing.T) {
	mock := newMockInstallAppServer("")
	mock.returnBody = `{"ok": true}` // 200 but no request_url
	defer mock.close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	inst := &store.AppInstallation{
		ID:            "inst-003",
		AppID:         "app-003",
		BotID:         "bot-003",
		AppToken:      "tok_nurl",
		AppWebhookSecret: "sec_nurl",
		Handle:        "no-url-bot",
	}

	requestURL, err := simulateNotifyAppInstalled(client, mock.server.URL+"/callback", inst.AppID, "sec_nurl", inst, db)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestURL != "" {
		t.Errorf("should return empty request_url, got %q", requestURL)
	}
	if db.updateCallCount != 0 {
		t.Errorf("db.UpdateAppWebhookURL should not be called without request_url, called %d times", db.updateCallCount)
	}
}

// ==================== Test 4: Auto-verify with wrong challenge ====================

func TestInstallFlow_AutoVerifyWrongChallenge(t *testing.T) {
	// Create a server that returns a wrong challenge
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"challenge": "wrong-challenge-value"})
	}))
	defer srv.Close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	verified := simulateAutoVerifyURL(client, "app-004", srv.URL, db)
	if verified {
		t.Error("should not verify with wrong challenge")
	}
	if db.urlVerified {
		t.Error("db.urlVerified should remain false")
	}
	if db.verifyCallCount != 0 {
		t.Errorf("SetAppWebhookVerified should not be called with wrong challenge, called %d times", db.verifyCallCount)
	}
}

// ==================== Test: Auto-verify with server error ====================

func TestInstallFlow_AutoVerifyServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	verified := simulateAutoVerifyURL(client, "app-005", srv.URL, db)
	if verified {
		t.Error("should not verify with server error")
	}
	if db.verifyCallCount != 0 {
		t.Errorf("SetAppWebhookVerified should not be called on error, called %d times", db.verifyCallCount)
	}
}

// ==================== Test: Auto-verify with unreachable server ====================

func TestInstallFlow_AutoVerifyUnreachable(t *testing.T) {
	db := &mockInstallDB{}
	client := &http.Client{Timeout: 1 * time.Second}

	verified := simulateAutoVerifyURL(client, "app-006", "http://127.0.0.1:1", db)
	if verified {
		t.Error("should not verify with unreachable server")
	}
	if db.verifyCallCount != 0 {
		t.Error("SetAppWebhookVerified should not be called")
	}
}

// ==================== Test: Full flow end-to-end with dynamic request_url ====================

func TestInstallFlow_EndToEnd_DynamicURL(t *testing.T) {
	// This mock App server returns its own URL as request_url and handles challenges.
	mock := newMockInstallAppServer("")
	defer mock.close()

	db := &mockInstallDB{}
	client := &http.Client{Timeout: 5 * time.Second}

	inst := &store.AppInstallation{
		ID:            "inst-e2e",
		AppID:         "app-e2e",
		BotID:         "bot-e2e",
		AppToken:      "tok_e2e",
		AppWebhookSecret: "sec_e2e",
		Handle:        "my-app",
	}

	// Step 1: Notify
	requestURL, err := simulateNotifyAppInstalled(client, mock.server.URL, inst.AppID, "sec_e2e", inst, db)
	if err != nil {
		t.Fatalf("notify failed: %v", err)
	}
	if requestURL == "" {
		t.Fatal("expected request_url")
	}
	if db.updateCallCount != 1 {
		t.Errorf("expected 1 UpdateAppWebhookURL call, got %d", db.updateCallCount)
	}

	// Step 2: Verify
	verified := simulateAutoVerifyURL(client, inst.AppID, requestURL, db)
	if !verified {
		t.Fatal("verify should succeed")
	}
	if db.verifyCallCount != 1 {
		t.Errorf("expected 1 SetAppWebhookVerified call, got %d", db.verifyCallCount)
	}
	if !db.urlVerified {
		t.Error("should be verified")
	}
}
