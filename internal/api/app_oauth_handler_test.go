package api

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

func TestOAuthSetupRedirectUsesRPOrigin(t *testing.T) {
	env := setupTestEnv(t)

	// Create a bot owned by the test user.
	bot := createTestBot(t, env.store, env.user.ID, "testbot")

	// Create an app with an oauth_setup_url.
	app, err := env.store.CreateApp(&store.App{
		OwnerID:       env.user.ID,
		Name:          "oauth-app",
		Slug:          "oauth-app",
		OAuthSetupURL: "https://external-app.example.com/oauth/setup",
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}

	// Hit the setup-redirect endpoint. Disable redirect following so we can
	// inspect the Location header directly.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, err := http.NewRequest("GET", env.ts.URL+"/api/apps/"+app.ID+"/oauth/setup?bot_id="+bot.ID, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.AddCookie(env.cookie)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302, got %d", resp.StatusCode)
	}

	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}

	// The hub and return_url query params must use RPOrigin ("http://localhost"),
	// NOT the test server's actual address (e.g. "http://127.0.0.1:xxxxx").
	const expectedOrigin = "http://localhost"

	hub := loc.Query().Get("hub")
	if hub != expectedOrigin {
		t.Errorf("hub = %q, want %q (must use RPOrigin, not r.Host)", hub, expectedOrigin)
	}

	returnURL := loc.Query().Get("return_url")
	if returnURL != expectedOrigin+"/oauth/complete" {
		t.Errorf("return_url = %q, want %q", returnURL, expectedOrigin+"/oauth/complete")
	}
}
