package storetest

import (
	"encoding/json"
	"testing"

	"github.com/openilink/openilink-hub/internal/store"
)

// TestAppLifecycle exercises the complete App lifecycle through the store layer:
// CRUD, installations with joined fields, OAuth+PKCE, listing flow, registry
// CRUD, marketplace apps, and scope/event data round-trips.
func TestAppLifecycle(t *testing.T, s store.Store) {
	// Seed a user and bot for the entire lifecycle test.
	u := mustCreateUser(t, s, "lifecycle_owner", "Lifecycle Owner")
	b := mustCreateBot(t, s, u.ID, "LifecycleBot")

	// -----------------------------------------------------------------------
	// 1. App CRUD
	// -----------------------------------------------------------------------

	t.Run("CreateAppWithAllFields", func(t *testing.T) {
		tools, _ := json.Marshal([]store.AppTool{
			{Name: "deploy", Command: "deploy", Description: "Deploy to production"},
		})
		events, _ := json.Marshal([]string{"message", "reaction.added"})
		scopes, _ := json.Marshal([]string{"message:read", "message:write"})

		app, err := s.CreateApp(&store.App{
			OwnerID:          u.ID,
			Name:             "Full App",
			Slug:             "full-app",
			Description:      "app with all fields",
			Icon:             "rocket",
			IconURL:          "https://img.example.com/rocket.png",
			Homepage:         "https://example.com",
			Tools:            tools,
			Events:           events,
			Scopes:           scopes,
			WebhookURL:       "https://example.com/webhook",
			OAuthSetupURL:    "https://example.com/setup",
			OAuthRedirectURL: "https://example.com/redirect",
		})
		if err != nil {
			t.Fatalf("CreateApp: %v", err)
		}
		if app.ID == "" {
			t.Fatal("expected non-empty ID")
		}
		if app.Listing != "unlisted" {
			t.Errorf("listing = %q, want %q", app.Listing, "unlisted")
		}
		if app.WebhookSecret == "" {
			t.Error("webhook_secret should be auto-generated")
		}
		if app.CreatedAt == 0 {
			t.Error("created_at should be set")
		}

		got, err := s.GetApp(app.ID)
		if err != nil {
			t.Fatalf("GetApp: %v", err)
		}
		if got.WebhookURL != "https://example.com/webhook" {
			t.Errorf("webhook_url = %q", got.WebhookURL)
		}
		if got.OAuthSetupURL != "https://example.com/setup" {
			t.Errorf("oauth_setup_url = %q", got.OAuthSetupURL)
		}
		if got.OAuthRedirectURL != "https://example.com/redirect" {
			t.Errorf("oauth_redirect_url = %q", got.OAuthRedirectURL)
		}
		if got.Icon != "rocket" {
			t.Errorf("icon = %q", got.Icon)
		}
		if got.IconURL != "https://img.example.com/rocket.png" {
			t.Errorf("icon_url = %q", got.IconURL)
		}
		if got.Homepage != "https://example.com" {
			t.Errorf("homepage = %q", got.Homepage)
		}

		// Verify JSON fields round-trip.
		var gotTools []store.AppTool
		if err := json.Unmarshal(got.Tools, &gotTools); err != nil {
			t.Fatalf("unmarshal tools: %v", err)
		}
		if len(gotTools) != 1 || gotTools[0].Command != "deploy" {
			t.Errorf("tools = %s", got.Tools)
		}
		var gotEvents []string
		if err := json.Unmarshal(got.Events, &gotEvents); err != nil {
			t.Fatalf("unmarshal events: %v", err)
		}
		if len(gotEvents) != 2 {
			t.Errorf("events = %s", got.Events)
		}
		var gotScopes []string
		if err := json.Unmarshal(got.Scopes, &gotScopes); err != nil {
			t.Fatalf("unmarshal scopes: %v", err)
		}
		if len(gotScopes) != 2 {
			t.Errorf("scopes = %s", got.Scopes)
		}
	})

	t.Run("CreateBuiltinApp", func(t *testing.T) {
		app, err := s.CreateApp(&store.App{
			OwnerID:     u.ID,
			Name:        "My Builtin App",
			Slug:        "my-builtin",
			Description: "builtin template app",
			Registry:    "builtin",
		})
		if err != nil {
			t.Fatalf("CreateApp(builtin): %v", err)
		}
		if app.Registry != "builtin" {
			t.Errorf("registry = %q, want %q", app.Registry, "builtin")
		}
		if app.WebhookURL != "" {
			t.Errorf("webhook_url should be empty for builtin, got %q", app.WebhookURL)
		}
	})

	t.Run("GetAppBySlug", func(t *testing.T) {
		got, err := s.GetAppBySlug("full-app", "")
		if err != nil {
			t.Fatalf("GetAppBySlug: %v", err)
		}
		if got.Name != "Full App" {
			t.Errorf("name = %q, want %q", got.Name, "Full App")
		}
	})

	t.Run("GetAppBySlug_NotFound", func(t *testing.T) {
		_, err := s.GetAppBySlug("nonexistent-slug", "")
		if err == nil {
			t.Error("expected error for non-existent slug")
		}
	})

	t.Run("DuplicateSlug_SameRegistry_Conflict", func(t *testing.T) {
		// Creating another app with same slug and same registry should fail
		_, err := s.CreateApp(&store.App{
			OwnerID:     u.ID,
			Name:        "Full App Duplicate",
			Slug:        "full-app",
			Description: "same slug, same registry (empty)",
		})
		if err == nil {
			t.Fatal("expected conflict error for duplicate slug+registry")
		}
	})

	t.Run("LocalAppSameSlugAsBuiltin", func(t *testing.T) {
		// A local app (registry="") with the same slug as a builtin app should succeed
		builtinApp, err := s.CreateApp(&store.App{
			OwnerID:  u.ID,
			Name:     "Builtin Full App",
			Slug:     "full-app",
			Registry: "builtin",
		})
		if err != nil {
			t.Fatalf("CreateApp(builtin same slug): %v", err)
		}
		// Look up each by slug+registry
		gotLocal, err := s.GetAppBySlug("full-app", "")
		if err != nil {
			t.Fatalf("GetAppBySlug(local): %v", err)
		}
		if gotLocal.Name != "Full App" {
			t.Errorf("name = %q, want %q", gotLocal.Name, "Full App")
		}
		gotBuiltin, err := s.GetAppBySlug("full-app", "builtin")
		if err != nil {
			t.Fatalf("GetAppBySlug(builtin): %v", err)
		}
		if gotBuiltin.Name != "Builtin Full App" {
			t.Errorf("name = %q, want %q", gotBuiltin.Name, "Builtin Full App")
		}
		// Clean up
		s.DeleteApp(builtinApp.ID)
	})

	t.Run("DuplicateSlug_DifferentRegistry", func(t *testing.T) {
		// Create two marketplace apps with the same slug but different registries
		regApp1, err := s.CreateApp(&store.App{
			OwnerID:  u.ID,
			Name:     "RegApp Alpha",
			Slug:     "reg-dup-slug",
			Registry: "https://alpha.example.com",
		})
		if err != nil {
			t.Fatalf("CreateApp(registry alpha): %v", err)
		}
		regApp2, err := s.CreateApp(&store.App{
			OwnerID:  u.ID,
			Name:     "RegApp Beta",
			Slug:     "reg-dup-slug",
			Registry: "https://beta.example.com",
		})
		if err != nil {
			t.Fatalf("CreateApp(registry beta): %v", err)
		}
		// Look up each by slug+registry
		gotAlpha, err := s.GetAppBySlug("reg-dup-slug", "https://alpha.example.com")
		if err != nil {
			t.Fatalf("GetAppBySlug(alpha): %v", err)
		}
		if gotAlpha.Name != "RegApp Alpha" {
			t.Errorf("name = %q, want %q", gotAlpha.Name, "RegApp Alpha")
		}
		gotBeta, err := s.GetAppBySlug("reg-dup-slug", "https://beta.example.com")
		if err != nil {
			t.Fatalf("GetAppBySlug(beta): %v", err)
		}
		if gotBeta.Name != "RegApp Beta" {
			t.Errorf("name = %q, want %q", gotBeta.Name, "RegApp Beta")
		}
		// Clean up
		s.DeleteApp(regApp1.ID)
		s.DeleteApp(regApp2.ID)
	})

	t.Run("UpdateAppFields", func(t *testing.T) {
		app, _ := s.GetAppBySlug("full-app", "")
		newTools, _ := json.Marshal([]store.AppTool{
			{Name: "deploy", Command: "deploy"},
			{Name: "status", Command: "status"},
		})
		newEvents, _ := json.Marshal([]string{"message", "reaction.added", "channel.created"})
		newScopes, _ := json.Marshal([]string{"message:read", "message:write", "channel:read"})

		err := s.UpdateApp(app.ID, "Full App Updated", "updated desc", "star", "https://new-icon.png",
			"https://new-home.com", "https://new-setup.com", "https://new-redirect.com",
			newTools, newEvents, newScopes)
		if err != nil {
			t.Fatalf("UpdateApp: %v", err)
		}

		got, _ := s.GetApp(app.ID)
		if got.Name != "Full App Updated" {
			t.Errorf("name = %q, want %q", got.Name, "Full App Updated")
		}
		if got.Description != "updated desc" {
			t.Errorf("description = %q", got.Description)
		}
		var tools []store.AppTool
		json.Unmarshal(got.Tools, &tools)
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("ListAppsByOwner", func(t *testing.T) {
		apps, err := s.ListAppsByOwner(u.ID)
		if err != nil {
			t.Fatalf("ListAppsByOwner: %v", err)
		}
		if len(apps) < 2 {
			t.Errorf("expected >= 2 apps, got %d", len(apps))
		}
	})

	t.Run("ListAppsByOwner_NoApps", func(t *testing.T) {
		apps, err := s.ListAppsByOwner("nonexistent-user")
		if err != nil {
			t.Fatalf("ListAppsByOwner: %v", err)
		}
		if len(apps) != 0 {
			t.Errorf("expected 0 apps, got %d", len(apps))
		}
	})

	// -----------------------------------------------------------------------
	// 2. Marketplace apps (registry != "")
	// -----------------------------------------------------------------------

	var marketplaceAppID string

	t.Run("CreateMarketplaceApp", func(t *testing.T) {
		tools, _ := json.Marshal([]store.AppTool{{Name: "sync", Command: "sync"}})
		events, _ := json.Marshal([]string{"message"})
		scopes, _ := json.Marshal([]string{"message:read"})

		app, err := s.CreateApp(&store.App{
			OwnerID:     u.ID,
			Name:        "Marketplace App",
			Slug:        "marketplace-app",
			Description: "from registry",
			Registry:    "https://registry.example.com",
			Version:     "1.0.0",
			Readme:      "# Marketplace App",
			Guide:       "Install and configure",
			WebhookURL:  "https://marketplace.example.com/webhook",
			Tools:       tools,
			Events:      events,
			Scopes:      scopes,
		})
		if err != nil {
			t.Fatalf("CreateApp(marketplace): %v", err)
		}
		marketplaceAppID = app.ID

		got, _ := s.GetApp(app.ID)
		if got.Registry != "https://registry.example.com" {
			t.Errorf("registry = %q", got.Registry)
		}
		if got.Version != "1.0.0" {
			t.Errorf("version = %q", got.Version)
		}
		if got.Readme != "# Marketplace App" {
			t.Errorf("readme = %q", got.Readme)
		}
		if got.Guide != "Install and configure" {
			t.Errorf("guide = %q", got.Guide)
		}
	})

	t.Run("ListMarketplaceApps", func(t *testing.T) {
		apps, err := s.ListMarketplaceApps()
		if err != nil {
			t.Fatalf("ListMarketplaceApps: %v", err)
		}
		found := false
		for _, a := range apps {
			if a.ID == marketplaceAppID {
				found = true
				break
			}
		}
		if !found {
			t.Error("marketplace app not found in ListMarketplaceApps")
		}
		// Non-marketplace apps should not appear.
		for _, a := range apps {
			if a.Registry == "" {
				t.Errorf("non-marketplace app %q in ListMarketplaceApps", a.Name)
			}
		}
	})

	t.Run("UpdateMarketplaceApp", func(t *testing.T) {
		newTools, _ := json.Marshal([]store.AppTool{{Name: "sync-v2", Command: "sync"}})
		newEvents, _ := json.Marshal([]string{"message", "reaction"})
		newScopes, _ := json.Marshal([]string{"message:read", "reaction:read"})

		err := s.UpdateMarketplaceApp(marketplaceAppID, "Marketplace App v2",
			"updated from registry", "https://new-icon.png", "https://new-home.com",
			"https://new-webhook.com", "https://new-setup.com", "https://new-redirect.com",
			"2.0.0", "Updated guide", newTools, newEvents, newScopes)
		if err != nil {
			t.Fatalf("UpdateMarketplaceApp: %v", err)
		}

		got, _ := s.GetApp(marketplaceAppID)
		if got.Name != "Marketplace App v2" {
			t.Errorf("name = %q", got.Name)
		}
		if got.Version != "2.0.0" {
			t.Errorf("version = %q", got.Version)
		}
		if got.Guide != "Updated guide" {
			t.Errorf("guide = %q", got.Guide)
		}
		if got.WebhookURL != "https://new-webhook.com" {
			t.Errorf("webhook_url = %q", got.WebhookURL)
		}
	})

	// -----------------------------------------------------------------------
	// 3. Installation lifecycle
	// -----------------------------------------------------------------------

	app, _ := s.GetAppBySlug("full-app", "")
	var instID string
	var instToken string

	t.Run("InstallApp", func(t *testing.T) {
		inst, err := s.InstallApp(app.ID, b.ID)
		if err != nil {
			t.Fatalf("InstallApp: %v", err)
		}
		instID = inst.ID
		instToken = inst.AppToken
		if inst.AppToken == "" {
			t.Error("app_token should be generated")
		}
		if !inst.Enabled {
			t.Error("new installation should be enabled by default")
		}
		if inst.CreatedAt == 0 {
			t.Error("created_at should be set")
		}
	})

	t.Run("GetInstallation_JoinedFields", func(t *testing.T) {
		got, err := s.GetInstallation(instID)
		if err != nil {
			t.Fatalf("GetInstallation: %v", err)
		}
		if got.AppID != app.ID {
			t.Errorf("app_id = %q, want %q", got.AppID, app.ID)
		}
		if got.BotID != b.ID {
			t.Errorf("bot_id = %q, want %q", got.BotID, b.ID)
		}
		// Verify joined app fields.
		if got.AppName == "" {
			t.Error("joined app_name should not be empty")
		}
		if got.AppSlug == "" {
			t.Error("joined app_slug should not be empty")
		}
		if got.AppWebhookURL == "" {
			t.Error("joined app_webhook_url should not be empty for an app with webhook_url")
		}
		if got.AppWebhookSecret == "" {
			t.Error("joined app_webhook_secret should not be empty")
		}
	})

	t.Run("GetInstallationByToken", func(t *testing.T) {
		got, err := s.GetInstallationByToken(instToken)
		if err != nil {
			t.Fatalf("GetInstallationByToken: %v", err)
		}
		if got.ID != instID {
			t.Errorf("id = %q, want %q", got.ID, instID)
		}
	})

	t.Run("GetInstallationByToken_NotFound", func(t *testing.T) {
		_, err := s.GetInstallationByToken("nonexistent_token")
		if err == nil {
			t.Error("expected error for non-existent token")
		}
	})

	t.Run("UpdateInstallation", func(t *testing.T) {
		cfg := json.RawMessage(`{"channel":"general"}`)
		scopes := json.RawMessage(`["message:read"]`)
		if err := s.UpdateInstallation(instID, "my-app-handle", cfg, scopes, false); err != nil {
			t.Fatalf("UpdateInstallation: %v", err)
		}
		got, _ := s.GetInstallation(instID)
		if got.Handle != "my-app-handle" {
			t.Errorf("handle = %q, want %q", got.Handle, "my-app-handle")
		}
		if got.Enabled {
			t.Error("expected enabled=false after update")
		}
		var gotCfg map[string]string
		if err := json.Unmarshal(got.Config, &gotCfg); err != nil {
			t.Fatalf("unmarshal config: %v", err)
		}
		if gotCfg["channel"] != "general" {
			t.Errorf("config.channel = %q", gotCfg["channel"])
		}
		var gotScopes []string
		if err := json.Unmarshal(got.Scopes, &gotScopes); err != nil {
			t.Fatalf("unmarshal scopes: %v", err)
		}
		if len(gotScopes) != 1 || gotScopes[0] != "message:read" {
			t.Errorf("scopes = %s", got.Scopes)
		}
	})

	t.Run("GetInstallationByHandle", func(t *testing.T) {
		got, err := s.GetInstallationByHandle(b.ID, "my-app-handle")
		if err != nil {
			t.Fatalf("GetInstallationByHandle: %v", err)
		}
		if got.ID != instID {
			t.Errorf("id = %q, want %q", got.ID, instID)
		}
	})

	t.Run("GetInstallationByHandle_NotFound", func(t *testing.T) {
		_, err := s.GetInstallationByHandle(b.ID, "nonexistent-handle")
		if err == nil {
			t.Error("expected error for non-existent handle")
		}
	})

	t.Run("ListInstallationsByApp", func(t *testing.T) {
		insts, err := s.ListInstallationsByApp(app.ID)
		if err != nil {
			t.Fatalf("ListInstallationsByApp: %v", err)
		}
		if len(insts) < 1 {
			t.Fatal("expected at least 1 installation")
		}
		// Verify joined fields are populated in list results too.
		if insts[0].AppName == "" {
			t.Error("joined app_name should be populated in list results")
		}
	})

	t.Run("ListInstallationsByBot", func(t *testing.T) {
		insts, err := s.ListInstallationsByBot(b.ID)
		if err != nil {
			t.Fatalf("ListInstallationsByBot: %v", err)
		}
		if len(insts) < 1 {
			t.Fatal("expected at least 1 installation")
		}
		// Verify joined app fields are populated in list-by-bot results.
		if insts[0].AppName == "" {
			t.Error("joined app_name should be populated in list-by-bot results")
		}
	})

	t.Run("RegenerateToken", func(t *testing.T) {
		newToken, err := s.RegenerateInstallationToken(instID)
		if err != nil {
			t.Fatalf("RegenerateInstallationToken: %v", err)
		}
		if newToken == instToken {
			t.Error("new token should differ from old token")
		}
		// Old token should no longer work.
		_, err = s.GetInstallationByToken(instToken)
		if err == nil {
			t.Error("old token should no longer resolve")
		}
		// New token should resolve.
		got, err := s.GetInstallationByToken(newToken)
		if err != nil {
			t.Fatalf("GetInstallationByToken(new): %v", err)
		}
		if got.ID != instID {
			t.Errorf("id = %q, want %q", got.ID, instID)
		}
		instToken = newToken
	})

	t.Run("InstallMarketplaceApp_JoinedFields", func(t *testing.T) {
		inst, err := s.InstallApp(marketplaceAppID, b.ID)
		if err != nil {
			t.Fatalf("InstallApp(marketplace): %v", err)
		}
		got, err := s.GetInstallation(inst.ID)
		if err != nil {
			t.Fatalf("GetInstallation: %v", err)
		}
		if got.AppRegistry != "https://registry.example.com" {
			t.Errorf("joined app_registry = %q", got.AppRegistry)
		}
		if got.AppGuide != "Updated guide" {
			t.Errorf("joined app_guide = %q", got.AppGuide)
		}
		// Clean up.
		s.DeleteInstallation(inst.ID)
	})

	// -----------------------------------------------------------------------
	// 4. OAuth + PKCE
	// -----------------------------------------------------------------------

	t.Run("OAuthCode_CreateAndExchange", func(t *testing.T) {
		err := s.CreateOAuthCode("pkce-code-1", app.ID, b.ID, "state-abc", "challenge-xyz")
		if err != nil {
			t.Fatalf("CreateOAuthCode: %v", err)
		}
		gotAppID, gotBotID, gotChallenge, err := s.ExchangeOAuthCode("pkce-code-1")
		if err != nil {
			t.Fatalf("ExchangeOAuthCode: %v", err)
		}
		if gotAppID != app.ID {
			t.Errorf("appID = %q, want %q", gotAppID, app.ID)
		}
		if gotBotID != b.ID {
			t.Errorf("botID = %q, want %q", gotBotID, b.ID)
		}
		if gotChallenge != "challenge-xyz" {
			t.Errorf("codeChallenge = %q, want %q", gotChallenge, "challenge-xyz")
		}
	})

	t.Run("OAuthCode_ConsumedOnExchange", func(t *testing.T) {
		// Code was already consumed above.
		_, _, _, err := s.ExchangeOAuthCode("pkce-code-1")
		if err == nil {
			t.Error("expected error re-using consumed code")
		}
	})

	t.Run("OAuthCode_NotFound", func(t *testing.T) {
		_, _, _, err := s.ExchangeOAuthCode("nonexistent-code")
		if err == nil {
			t.Error("expected error for non-existent code")
		}
	})

	t.Run("CleanExpiredOAuthCodes", func(t *testing.T) {
		// Just verify it doesn't panic or error.
		s.CleanExpiredOAuthCodes()
	})

	// -----------------------------------------------------------------------
	// 5. Listing flow
	// -----------------------------------------------------------------------

	// Create a fresh app for listing tests to avoid interference.
	listingApp := mustCreateApp(t, s, u.ID, "Listing Test App", "listing-test-app")

	t.Run("ListingFlow_StartsUnlisted", func(t *testing.T) {
		got, _ := s.GetApp(listingApp.ID)
		if got.Listing != "unlisted" {
			t.Errorf("listing = %q, want %q", got.Listing, "unlisted")
		}
	})

	t.Run("ListingFlow_RequestPending", func(t *testing.T) {
		if err := s.RequestListing(listingApp.ID); err != nil {
			t.Fatalf("RequestListing: %v", err)
		}
		got, _ := s.GetApp(listingApp.ID)
		if got.Listing != "pending" {
			t.Errorf("listing = %q, want %q", got.Listing, "pending")
		}
	})

	t.Run("ListingFlow_Reject", func(t *testing.T) {
		if err := s.ReviewListing(listingApp.ID, false, "needs more docs"); err != nil {
			t.Fatalf("ReviewListing(reject): %v", err)
		}
		got, _ := s.GetApp(listingApp.ID)
		if got.Listing != "rejected" {
			t.Errorf("listing = %q, want %q", got.Listing, "rejected")
		}
		if got.ListingRejectReason != "needs more docs" {
			t.Errorf("reject_reason = %q, want %q", got.ListingRejectReason, "needs more docs")
		}
	})

	t.Run("ListingFlow_ReRequest", func(t *testing.T) {
		if err := s.RequestListing(listingApp.ID); err != nil {
			t.Fatalf("RequestListing (re-request): %v", err)
		}
		got, _ := s.GetApp(listingApp.ID)
		if got.Listing != "pending" {
			t.Errorf("listing = %q, want %q", got.Listing, "pending")
		}
	})

	t.Run("ListingFlow_Approve", func(t *testing.T) {
		if err := s.ReviewListing(listingApp.ID, true, ""); err != nil {
			t.Fatalf("ReviewListing(approve): %v", err)
		}
		got, _ := s.GetApp(listingApp.ID)
		if got.Listing != "listed" {
			t.Errorf("listing = %q, want %q", got.Listing, "listed")
		}
	})

	t.Run("ListListedApps", func(t *testing.T) {
		apps, err := s.ListListedApps()
		if err != nil {
			t.Fatalf("ListListedApps: %v", err)
		}
		found := false
		for _, a := range apps {
			if a.ID == listingApp.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("approved app should appear in ListListedApps")
		}
	})

	// -----------------------------------------------------------------------
	// 6. Registry CRUD
	// -----------------------------------------------------------------------

	var registryID string

	t.Run("CreateRegistry", func(t *testing.T) {
		r := &store.Registry{
			Name: "Official Registry",
			URL:  "https://registry.openilink.com",
		}
		if err := s.CreateRegistry(r); err != nil {
			t.Fatalf("CreateRegistry: %v", err)
		}
		registryID = r.ID
		if registryID == "" {
			t.Fatal("expected non-empty registry ID")
		}
		if r.CreatedAt == 0 {
			t.Error("created_at should be set")
		}
	})

	t.Run("ListRegistries", func(t *testing.T) {
		regs, err := s.ListRegistries()
		if err != nil {
			t.Fatalf("ListRegistries: %v", err)
		}
		if len(regs) < 1 {
			t.Fatal("expected at least 1 registry")
		}
		found := false
		for _, r := range regs {
			if r.ID == registryID {
				found = true
				if r.Name != "Official Registry" {
					t.Errorf("name = %q", r.Name)
				}
				if r.URL != "https://registry.openilink.com" {
					t.Errorf("url = %q", r.URL)
				}
			}
		}
		if !found {
			t.Error("created registry not found in list")
		}
	})

	t.Run("UpdateRegistryEnabled_Disable", func(t *testing.T) {
		if err := s.UpdateRegistryEnabled(registryID, false); err != nil {
			t.Fatalf("UpdateRegistryEnabled(false): %v", err)
		}
		regs, _ := s.ListRegistries()
		for _, r := range regs {
			if r.ID == registryID && r.Enabled {
				t.Error("registry should be disabled")
			}
		}
	})

	t.Run("UpdateRegistryEnabled_Enable", func(t *testing.T) {
		if err := s.UpdateRegistryEnabled(registryID, true); err != nil {
			t.Fatalf("UpdateRegistryEnabled(true): %v", err)
		}
		regs, _ := s.ListRegistries()
		for _, r := range regs {
			if r.ID == registryID && !r.Enabled {
				t.Error("registry should be enabled")
			}
		}
	})

	t.Run("DeleteRegistry", func(t *testing.T) {
		if err := s.DeleteRegistry(registryID); err != nil {
			t.Fatalf("DeleteRegistry: %v", err)
		}
		regs, _ := s.ListRegistries()
		for _, r := range regs {
			if r.ID == registryID {
				t.Error("deleted registry should not appear in list")
			}
		}
	})

	// -----------------------------------------------------------------------
	// 7. Scope data round-trips (store layer)
	// -----------------------------------------------------------------------

	t.Run("ScopeRoundTrip_AppScopes", func(t *testing.T) {
		scopes, _ := json.Marshal([]string{"message:read", "reaction:read"})
		events, _ := json.Marshal([]string{"message"})
		scopeApp, err := s.CreateApp(&store.App{
			OwnerID:     u.ID,
			Name:        "Scope Test App",
			Slug:        "scope-test-app",
			Description: "tests scope persistence",
			Scopes:      scopes,
			Events:      events,
		})
		if err != nil {
			t.Fatalf("CreateApp: %v", err)
		}
		got, _ := s.GetApp(scopeApp.ID)
		var gotScopes []string
		json.Unmarshal(got.Scopes, &gotScopes)
		if len(gotScopes) != 2 {
			t.Fatalf("expected 2 scopes, got %d", len(gotScopes))
		}
		hasMessageRead := false
		for _, sc := range gotScopes {
			if sc == "message:read" {
				hasMessageRead = true
			}
		}
		if !hasMessageRead {
			t.Error("message:read scope not found")
		}
	})

	t.Run("ScopeRoundTrip_InstallationScopes", func(t *testing.T) {
		scopeApp, _ := s.GetAppBySlug("scope-test-app", "")
		inst, err := s.InstallApp(scopeApp.ID, b.ID)
		if err != nil {
			t.Fatalf("InstallApp: %v", err)
		}

		// Default: installation scopes should be empty array.
		var defaultScopes []string
		json.Unmarshal(inst.Scopes, &defaultScopes)
		if len(defaultScopes) != 0 {
			t.Errorf("default installation scopes should be empty, got %s", inst.Scopes)
		}

		// Set installation-level scopes (subset of app scopes).
		instScopes, _ := json.Marshal([]string{"message:read"})
		if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), instScopes, true); err != nil {
			t.Fatalf("UpdateInstallation: %v", err)
		}

		got, _ := s.GetInstallation(inst.ID)
		var gotScopes []string
		json.Unmarshal(got.Scopes, &gotScopes)
		if len(gotScopes) != 1 || gotScopes[0] != "message:read" {
			t.Errorf("installation scopes = %s, want [\"message:read\"]", got.Scopes)
		}

		// Clear installation scopes back to empty.
		emptyScopes, _ := json.Marshal([]string{})
		if err := s.UpdateInstallation(inst.ID, "", json.RawMessage("{}"), emptyScopes, true); err != nil {
			t.Fatalf("UpdateInstallation (clear scopes): %v", err)
		}
		got, _ = s.GetInstallation(inst.ID)
		var clearedScopes []string
		json.Unmarshal(got.Scopes, &clearedScopes)
		if len(clearedScopes) != 0 {
			t.Errorf("cleared scopes = %s, want []", got.Scopes)
		}

		s.DeleteInstallation(inst.ID)
	})

	// -----------------------------------------------------------------------
	// 8. Delete app cascades installations
	// -----------------------------------------------------------------------

	t.Run("DeleteApp_CascadesInstallations", func(t *testing.T) {
		cascApp, _ := s.CreateApp(&store.App{
			OwnerID:     u.ID,
			Name:        "Cascade App",
			Slug:        "cascade-app",
			Description: "test cascade delete",
		})
		inst1, _ := s.InstallApp(cascApp.ID, b.ID)

		if err := s.DeleteApp(cascApp.ID); err != nil {
			t.Fatalf("DeleteApp: %v", err)
		}

		// App should be gone.
		_, err := s.GetApp(cascApp.ID)
		if err == nil {
			t.Error("expected error after deleting app")
		}

		// Installation should also be gone.
		_, err = s.GetInstallation(inst1.ID)
		if err == nil {
			t.Error("expected installation to be deleted when app is deleted")
		}
	})

	// -----------------------------------------------------------------------
	// 9. Delete installation (standalone)
	// -----------------------------------------------------------------------

	t.Run("DeleteInstallation", func(t *testing.T) {
		if err := s.DeleteInstallation(instID); err != nil {
			t.Fatalf("DeleteInstallation: %v", err)
		}
		_, err := s.GetInstallation(instID)
		if err == nil {
			t.Error("expected error after deleting installation")
		}
	})
}
