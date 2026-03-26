package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/store"
)

//go:embed app_skill.md
var appSkillMD string

func handleAppSkill(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Write([]byte(appSkillMD))
}

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

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
		Kind             string          `json:"kind"`
		Readme           string          `json:"readme"`
		Guide            string          `json:"guide"`
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

	// Check slug uniqueness
	if existing, _ := s.Store.GetAppBySlug(slug); existing != nil {
		jsonError(w, "slug already taken", http.StatusConflict)
		return
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
		Kind:             req.Kind,
		Readme:           req.Readme,
		Guide:            req.Guide,
	})
	if err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

// GET /api/apps?listed=true — public marketplace; otherwise my apps
func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	var apps []store.App
	var err error

	if r.URL.Query().Get("listed") == "true" {
		apps, err = s.Store.ListListedApps()
	} else {
		userID := auth.UserIDFromContext(r.Context())
		apps, err = s.Store.ListAppsByOwner(userID)
	}
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
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
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

	if err := s.Store.UpdateApp(appID, name, description, icon, iconURL, homepage, oauthSetupURL, oauthRedirectURL, tools, events, scopes); err != nil {
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
	if err := s.Store.RequestListing(app.ID); err != nil {
		jsonError(w, "request failed", http.StatusInternalServerError)
		return
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
