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
		Name        string          `json:"name"`
		Slug        string          `json:"slug"`
		Description string          `json:"description"`
		Icon        string          `json:"icon"`
		IconURL     string          `json:"icon_url"`
		Homepage    string          `json:"homepage"`
		SetupURL    string          `json:"setup_url"`
		RedirectURL string          `json:"redirect_url"`
		Tools       json.RawMessage `json:"tools"`
		Events      json.RawMessage `json:"events"`
		Scopes      json.RawMessage `json:"scopes"`
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
		OwnerID:     userID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
		IconURL:     req.IconURL,
		Homepage:    req.Homepage,
		SetupURL:    req.SetupURL,
		RedirectURL: req.RedirectURL,
		Tools:       req.Tools,
		Events:      req.Events,
		Scopes:      req.Scopes,
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
	// Owner can see everything; others can only see listed apps (without client_secret)
	if app.OwnerID != userID {
		if !app.Listed {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		app.ClientSecret = ""  // hide secrets from non-owners
		app.SigningSecret = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
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

	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Icon        string          `json:"icon"`
		IconURL     string          `json:"icon_url"`
		Homepage    string          `json:"homepage"`
		SetupURL    string          `json:"setup_url"`
		RedirectURL string          `json:"redirect_url"`
		RequestURL  *string         `json:"request_url"`
		Tools       json.RawMessage `json:"tools"`
		Events      json.RawMessage `json:"events"`
		Scopes      json.RawMessage `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	name := app.Name
	if req.Name != "" {
		name = req.Name
	}
	description := app.Description
	if req.Description != "" {
		description = req.Description
	}
	icon := app.Icon
	if req.Icon != "" {
		icon = req.Icon
	}
	iconURL := app.IconURL
	if req.IconURL != "" {
		iconURL = req.IconURL
	}
	homepage := app.Homepage
	if req.Homepage != "" {
		homepage = req.Homepage
	}
	setupURL := app.SetupURL
	if req.SetupURL != "" {
		setupURL = req.SetupURL
	}
	redirectURL := app.RedirectURL
	if req.RedirectURL != "" {
		redirectURL = req.RedirectURL
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

	if err := s.Store.UpdateApp(appID, name, description, icon, iconURL, homepage, setupURL, redirectURL, tools, events, scopes); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Update request_url separately (resets url_verified)
	if req.RequestURL != nil && *req.RequestURL != app.RequestURL {
		if err := s.Store.UpdateAppRequestURL(appID, *req.RequestURL); err != nil {
			jsonError(w, "update request_url failed", http.StatusInternalServerError)
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
	if app.Listed {
		jsonError(w, "already listed", http.StatusBadRequest)
		return
	}
	if app.ListingStatus == "pending" {
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

// PUT /api/admin/apps/{id}/listed — toggle listed status (admin only)
func (s *Server) handleSetAppListed(w http.ResponseWriter, r *http.Request) {
	appID := r.PathValue("id")
	var req struct {
		Listed bool `json:"listed"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := s.Store.SetAppListed(appID, req.Listed); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
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
	if !isAdmin && app.OwnerID != userID && !app.Listed {
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
