package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

func buildListingSnapshot(app *store.App) string {
	snap := map[string]any{
		"tools":         app.Tools,
		"events":        app.Events,
		"scopes":        app.Scopes,
		"webhook_url":   app.WebhookURL,
		"config_schema": app.ConfigSchema,
		"version":       app.Version,
	}
	b, _ := json.Marshal(snap)
	return string(b)
}

// POST /api/apps
func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		Name             string          `json:"name"`
		Slug             string          `json:"slug"`
		Description      string          `json:"description"`
		Icon             string          `json:"icon"`
		IconURL          string          `json:"icon_url"`
		Homepage         string          `json:"homepage"`
		OAuthSetupURL    string          `json:"oauth_setup_url"`
		OAuthRedirectURL string          `json:"oauth_redirect_url"`
		Tools            json.RawMessage `json:"tools"`
		Events           json.RawMessage `json:"events"`
		Scopes           json.RawMessage `json:"scopes"`
		Version          string          `json:"version"`
		Readme           string          `json:"readme"`
		Guide            string          `json:"guide"`
		ConfigSchema     string `json:"config_schema"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		jsonError(w, "name required", http.StatusBadRequest)
		return
	}

	// Validate slug
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if !slugRe.MatchString(slug) {
		jsonError(w, "slug must be 3-40 chars, lowercase alphanumeric and hyphens", http.StatusBadRequest)
		return
	}

	// Check slug uniqueness among local apps (registry="")
	if existing, _ := s.Store.GetAppBySlug(slug, ""); existing != nil {
		jsonError(w, "slug already taken", http.StatusConflict)
		return
	}

	if req.Homepage != "" {
		u, err := url.ParseRequestURI(req.Homepage)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			jsonError(w, "homepage must be a valid http or https URL", http.StatusBadRequest)
			return
		}
	}

	app, err := s.Store.CreateApp(&store.App{
		OwnerID:          userID,
		Name:             req.Name,
		Slug:             slug,
		Description:      req.Description,
		Icon:             req.Icon,
		IconURL:          req.IconURL,
		Homepage:         req.Homepage,
		OAuthSetupURL:    req.OAuthSetupURL,
		OAuthRedirectURL: req.OAuthRedirectURL,
		Tools:            req.Tools,
		Events:           req.Events,
		Scopes:           req.Scopes,
		Version:          req.Version,
		Readme:           req.Readme,
		Guide:            req.Guide,
		ConfigSchema:     req.ConfigSchema,
	})
	if err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

// GET /api/apps?listing=listed — public marketplace; otherwise my apps
func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	var apps []store.App
	var err error

	if r.URL.Query().Get("listing") == "listed" {
		apps, err = s.Store.ListListedApps()
	} else {
		userID := auth.UserIDFromContext(r.Context())
		apps, err = s.Store.ListAppsByOwner(userID)
	}
	if err != nil {
		slog.Error("list apps failed", "err", err)
		jsonError(w, "list failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []store.App{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// GET /api/apps/{id}
func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	// Owner can see everything; others can see listed apps or apps they have installed
	if app.OwnerID != userID {
		if app.Listing != "listed" && !s.userHasInstallation(userID, appID) {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		// Secret is already hidden via json:"-" on store.App
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(app)
		return
	}

	// Owner sees everything including secrets
	type appWithSecret struct {
		*store.App
		WebhookSecret string `json:"webhook_secret,omitempty"`
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(appWithSecret{App: app, WebhookSecret: app.WebhookSecret})
}

// PUT /api/apps/{id}
func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	// Marketplace apps cannot be edited directly
	if app.Registry != "" {
		jsonError(w, "marketplace apps cannot be edited", http.StatusForbidden)
		return
	}

	var req struct {
		Name             *string         `json:"name"`
		Description      *string         `json:"description"`
		Icon             *string         `json:"icon"`
		IconURL          *string         `json:"icon_url"`
		Homepage         *string         `json:"homepage"`
		OAuthSetupURL    *string         `json:"oauth_setup_url"`
		OAuthRedirectURL *string         `json:"oauth_redirect_url"`
		WebhookURL       *string         `json:"webhook_url"`
		Tools            json.RawMessage `json:"tools"`
		Events           json.RawMessage `json:"events"`
		Scopes           json.RawMessage `json:"scopes"`
		ConfigSchema     *string         `json:"config_schema"`
		Version          *string         `json:"version"`
		Readme           *string         `json:"readme"`
		Guide            *string         `json:"guide"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// During review, only allow cosmetic updates (readme, description, icon).
	// Core changes require withdrawing the listing request first.
	if app.Listing == "pending" {
		if req.WebhookURL != nil || req.Tools != nil || req.Events != nil || req.Scopes != nil || req.ConfigSchema != nil || req.Version != nil {
			jsonError(w, "cannot modify core fields while listing is pending review. Withdraw the request first or wait for review.", http.StatusForbidden)
			return
		}
	}

	name := app.Name
	if req.Name != nil {
		name = *req.Name
	}
	description := app.Description
	if req.Description != nil {
		description = *req.Description
	}
	icon := app.Icon
	if req.Icon != nil {
		icon = *req.Icon
	}
	iconURL := app.IconURL
	if req.IconURL != nil {
		iconURL = *req.IconURL
	}
	homepage := app.Homepage
	if req.Homepage != nil {
		if *req.Homepage != "" {
			u, err := url.ParseRequestURI(*req.Homepage)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
				jsonError(w, "homepage must be a valid http or https URL", http.StatusBadRequest)
				return
			}
		}
		homepage = *req.Homepage
	}
	oauthSetupURL := app.OAuthSetupURL
	if req.OAuthSetupURL != nil {
		oauthSetupURL = *req.OAuthSetupURL
	}
	oauthRedirectURL := app.OAuthRedirectURL
	if req.OAuthRedirectURL != nil {
		oauthRedirectURL = *req.OAuthRedirectURL
	}
	configSchema := app.ConfigSchema
	if req.ConfigSchema != nil {
		configSchema = *req.ConfigSchema
	}
	version := app.Version
	if req.Version != nil {
		version = *req.Version
	}
	readme := app.Readme
	if req.Readme != nil {
		readme = *req.Readme
	}
	guide := app.Guide
	if req.Guide != nil {
		guide = *req.Guide
	}
	tools := app.Tools
	if req.Tools != nil {
		tools = req.Tools
	}
	events := app.Events
	if req.Events != nil {
		events = req.Events
	}
	scopes := app.Scopes
	if req.Scopes != nil {
		scopes = req.Scopes
	}

	if err := s.Store.UpdateApp(appID, name, description, icon, iconURL, homepage, oauthSetupURL, oauthRedirectURL, configSchema, version, readme, guide, tools, events, scopes); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// TODO: webhook_url update is non-atomic with UpdateApp above. If this fails,
	// the DB is partially updated. Consider merging webhook_url into UpdateApp
	// or wrapping both in a transaction.
	if req.WebhookURL != nil && *req.WebhookURL != app.WebhookURL {
		if err := s.Store.UpdateAppWebhookURL(appID, *req.WebhookURL); err != nil {
			jsonError(w, "update webhook_url failed", http.StatusInternalServerError)
			return
		}
	}

	// Auto-revert listed apps to pending when core fields change.
	if app.Listing == "listed" {
		coreChanged := false
		if req.WebhookURL != nil && *req.WebhookURL != app.WebhookURL {
			coreChanged = true
		}
		if req.Tools != nil {
			coreChanged = true
		}
		if req.Events != nil {
			coreChanged = true
		}
		if req.Scopes != nil {
			coreChanged = true
		}
		if req.ConfigSchema != nil && *req.ConfigSchema != app.ConfigSchema {
			coreChanged = true
		}
		if req.Version != nil && *req.Version != app.Version {
			coreChanged = true
		}

		if coreChanged {
			if err := s.Store.SetListing(appID, "pending"); err != nil {
				slog.Error("failed to revert listing to pending", "app", appID, "err", err)
				jsonError(w, "failed to revert listing", http.StatusInternalServerError)
				return
			}
			slog.Info("listed app core fields changed, reverted to pending", "app", appID)
			updatedApp, _ := s.Store.GetApp(appID)
			if updatedApp != nil {
				if err := s.Store.CreateAppReview(&store.AppReview{
					AppID:    appID,
					Action:   "auto_revert",
					ActorID:  "system",
					Version:  updatedApp.Version,
					Snapshot: buildListingSnapshot(updatedApp),
				}); err != nil {
					slog.Warn("failed to create app review record", "app", appID, "action", "auto_revert", "err", err)
				}
			}
		}
	}

	jsonOK(w)
}

// DELETE /api/apps/{id}
func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if err := s.Store.DeleteApp(appID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// POST /api/apps/{id}/request-listing — owner requests to list their app
func (s *Server) handleRequestListing(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
	if app == nil {
		return
	}
	if app.Listing == "listed" {
		jsonError(w, "already listed", http.StatusBadRequest)
		return
	}
	if app.Listing == "pending" {
		jsonError(w, "already pending review", http.StatusBadRequest)
		return
	}

	// Validate required fields before submitting for review.
	var errors []string
	if app.Name == "" {
		errors = append(errors, "name is required")
	}
	if app.Description == "" {
		errors = append(errors, "description is required")
	}
	if app.Readme == "" {
		errors = append(errors, "readme is required")
	}
	if app.Version == "" {
		errors = append(errors, "version is required")
	}
	// Apps need at least one way to receive events: webhook_url or OAuth (setup_url for WS/PKCE)
	if app.WebhookURL == "" && app.OAuthSetupURL == "" && app.Registry != "builtin" {
		errors = append(errors, "webhook_url or oauth_setup_url is required")
	}
	// Must have events or tools.
	hasEvents := len(app.Events) > 0 && string(app.Events) != "[]" && string(app.Events) != "null"
	hasTools := len(app.Tools) > 0 && string(app.Tools) != "[]" && string(app.Tools) != "null"
	if !hasEvents && !hasTools {
		errors = append(errors, "at least one event subscription or tool is required")
	}
	// Must have scopes.
	hasScopes := len(app.Scopes) > 0 && string(app.Scopes) != "[]" && string(app.Scopes) != "null"
	if !hasScopes {
		errors = append(errors, "at least one scope is required")
	}

	if len(errors) > 0 {
		jsonError(w, "cannot submit for listing: "+strings.Join(errors, "; "), http.StatusBadRequest)
		return
	}

	if err := s.Store.RequestListing(app.ID); err != nil {
		jsonError(w, "request failed", http.StatusInternalServerError)
		return
	}
	if err := s.Store.CreateAppReview(&store.AppReview{
		AppID:    app.ID,
		Action:   "request",
		ActorID:  auth.UserIDFromContext(r.Context()),
		Version:  app.Version,
		Snapshot: buildListingSnapshot(app),
	}); err != nil {
		slog.Warn("failed to create app review record", "app", app.ID, "action", "request", "err", err)
	}
	jsonOK(w)
}

// POST /api/apps/{id}/withdraw-listing — owner withdraws a pending listing request
func (s *Server) handleWithdrawListing(w http.ResponseWriter, r *http.Request) {
	app := s.requireApp(w, r)
	if app == nil {
		return
	}
	if app.Listing != "pending" {
		jsonError(w, "not pending", http.StatusBadRequest)
		return
	}
	if err := s.Store.WithdrawListing(app.ID); err != nil {
		jsonError(w, "withdraw failed", http.StatusInternalServerError)
		return
	}
	if err := s.Store.CreateAppReview(&store.AppReview{
		AppID:    app.ID,
		Action:   "withdraw",
		ActorID:  auth.UserIDFromContext(r.Context()),
		Version:  app.Version,
		Snapshot: buildListingSnapshot(app),
	}); err != nil {
		slog.Warn("failed to create app review record", "app", app.ID, "action", "withdraw", "err", err)
	}
	jsonOK(w)
}

// PUT /api/admin/apps/{id}/review-listing — admin approves/rejects listing
func (s *Server) handleReviewListing(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("id")
	var req struct {
		Approve bool   `json:"approve"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if !req.Approve && req.Reason == "" {
		jsonError(w, "reason required for rejection", http.StatusBadRequest)
		return
	}
	if err := s.Store.ReviewListing(appID, req.Approve, req.Reason); err != nil {
		jsonError(w, "review failed", http.StatusInternalServerError)
		return
	}
	app, _ := s.Store.GetApp(appID)
	actorID := auth.UserIDFromContext(r.Context())
	action := "approve"
	if !req.Approve {
		action = "reject"
	}
	if app != nil {
		if err := s.Store.CreateAppReview(&store.AppReview{
			AppID:    appID,
			Action:   action,
			ActorID:  actorID,
			Reason:   req.Reason,
			Version:  app.Version,
			Snapshot: buildListingSnapshot(app),
		}); err != nil {
			slog.Warn("failed to create app review record", "app", appID, "action", action, "err", err)
		}
	}
	jsonOK(w)
}

// GET /api/admin/apps — list all apps (admin only)
func (s *Server) handleAdminListApps(w http.ResponseWriter, r *http.Request) {
	apps, err := s.Store.ListAllApps()
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []store.App{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// userHasInstallation checks if the user owns any bot that has this app installed.
func (s *Server) userHasInstallation(userID, appID string) bool {
	installations, err := s.Store.ListInstallationsByApp(appID)
	if err != nil {
		return false
	}
	for _, inst := range installations {
		bot, err := s.Store.GetBot(inst.BotID)
		if err == nil && bot.UserID == userID {
			return true
		}
	}
	return false
}

// requireApp loads an app by path ID and verifies ownership.
// Returns the app or nil (with error already written to w).
func (s *Server) requireApp(w http.ResponseWriter, r *http.Request) *store.App {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return nil
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return nil
	}
	return app
}

// requireAppForInstall loads an app that the user can install:
// either they own it, or it's publicly listed.
func (s *Server) requireAppForInstall(w http.ResponseWriter, r *http.Request) *store.App {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return nil
	}
	// Admin can access all apps; otherwise must be owner or listed
	user, _ := s.Store.GetUserByID(userID)
	isAdmin := user != nil && store.IsAdmin(user.Role)
	if !isAdmin && app.OwnerID != userID && app.Listing != "listed" {
		jsonError(w, "not found", http.StatusNotFound)
		return nil
	}
	return app
}

// PUT /api/admin/apps/{id}/listing — admin directly sets listing status
// PUT /api/admin/apps/{id}/listing — admin directly sets listing status (listed/unlisted).
// This is an admin privilege bypass of the normal review flow, intended for
// moderation actions (e.g. emergency takedown, re-listing without re-review).
func (s *Server) handleAdminSetListing(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("id")
	if _, err := s.Store.GetApp(appID); err != nil {
		jsonError(w, "app not found", http.StatusNotFound)
		return
	}
	var req struct {
		Listing string `json:"listing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	validListings := map[string]bool{"listed": true, "unlisted": true}
	if !validListings[req.Listing] {
		jsonError(w, "listing must be 'listed' or 'unlisted'", http.StatusBadRequest)
		return
	}
	if err := s.Store.SetListing(appID, req.Listing); err != nil {
		slog.Error("set listing failed", "err", err)
		jsonError(w, "set listing failed", http.StatusInternalServerError)
		return
	}
	slog.Info("admin set listing", "app_id", appID, "listing", req.Listing)
	app, _ := s.Store.GetApp(appID)
	if app != nil {
		if err := s.Store.CreateAppReview(&store.AppReview{
			AppID:    appID,
			Action:   "admin_set",
			ActorID:  auth.UserIDFromContext(r.Context()),
			Reason:   req.Listing,
			Version:  app.Version,
			Snapshot: buildListingSnapshot(app),
		}); err != nil {
			slog.Warn("failed to create app review record", "app", appID, "action", "admin_set", "err", err)
		}
	}
	jsonOK(w)
}

// GET /api/apps/{id}/reviews
func (s *Server) handleListAppReviews(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.Store.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	// Only owner or admin can view review history
	user, _ := s.Store.GetUserByID(userID)
	isAdmin := user != nil && store.IsAdmin(user.Role)
	if app.OwnerID != userID && !isAdmin {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	reviews, err := s.Store.ListAppReviews(appID)
	if err != nil {
		jsonError(w, "list reviews failed", http.StatusInternalServerError)
		return
	}
	if reviews == nil {
		reviews = []store.AppReview{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reviews)
}

// requireInstallation loads an installation by path IID and verifies it belongs to the app
// and the current user owns the bot.
func (s *Server) requireInstallation(w http.ResponseWriter, r *http.Request, appID string) *store.AppInstallation {
	userID := auth.UserIDFromContext(r.Context())
	iid := r.PathValue("iid")

	inst, err := s.Store.GetInstallation(iid)
	if err != nil {
		jsonError(w, "installation not found", http.StatusNotFound)
		return nil
	}
	if inst.AppID != appID {
		jsonError(w, "installation not found", http.StatusNotFound)
		return nil
	}

	// Verify: admin, app owner, or bot owner
	user, _ := s.Store.GetUserByID(userID)
	isAdmin := user != nil && store.IsAdmin(user.Role)
	app, _ := s.Store.GetApp(appID)
	isAppOwner := app != nil && app.OwnerID == userID
	if !isAdmin && !isAppOwner {
		bot, err := s.Store.GetBot(inst.BotID)
		if err != nil || bot.UserID != userID {
			jsonError(w, "installation not found", http.StatusNotFound)
			return nil
		}
	}
	return inst
}
