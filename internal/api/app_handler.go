package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/database"
)

var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// POST /api/apps
func (s *Server) handleCreateApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	var req struct {
		Name        string          `json:"name"`
		Slug        string          `json:"slug"`
		Description string          `json:"description"`
		Icon        string          `json:"icon"`
		Homepage    string          `json:"homepage"`
		Commands    json.RawMessage `json:"commands"`
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
	if existing, _ := s.DB.GetAppBySlug(slug); existing != nil {
		jsonError(w, "slug already taken", http.StatusConflict)
		return
	}

	app, err := s.DB.CreateApp(&database.App{
		OwnerID:     userID,
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		Icon:        req.Icon,
		Homepage:    req.Homepage,
		Commands:    req.Commands,
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

// GET /api/apps
func (s *Server) handleListApps(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	apps, err := s.DB.ListAppsByOwner(userID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	if apps == nil {
		apps = []database.App{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

// GET /api/apps/{id}
func (s *Server) handleGetApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.DB.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app)
}

// PUT /api/apps/{id}
func (s *Server) handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.DB.GetApp(appID)
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
		Homepage    string          `json:"homepage"`
		SetupURL    string          `json:"setup_url"`
		RedirectURL string          `json:"redirect_url"`
		Commands    json.RawMessage `json:"commands"`
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
	commands := app.Commands
	if req.Commands != nil {
		commands = req.Commands
	}
	events := app.Events
	if req.Events != nil {
		events = req.Events
	}
	scopes := app.Scopes
	if req.Scopes != nil {
		scopes = req.Scopes
	}

	if err := s.DB.UpdateApp(appID, name, description, icon, homepage, setupURL, redirectURL, commands, events, scopes); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// DELETE /api/apps/{id}
func (s *Server) handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.DB.GetApp(appID)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	if err := s.DB.DeleteApp(appID); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// requireApp loads an app by path ID and verifies ownership.
// Returns the app or nil (with error already written to w).
func (s *Server) requireApp(w http.ResponseWriter, r *http.Request) *database.App {
	userID := auth.UserIDFromContext(r.Context())
	appID := r.PathValue("id")

	app, err := s.DB.GetApp(appID)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonError(w, "not found", http.StatusNotFound)
		} else {
			jsonError(w, "not found", http.StatusNotFound)
		}
		return nil
	}
	if app.OwnerID != userID {
		jsonError(w, "not found", http.StatusNotFound)
		return nil
	}
	return app
}

// requireInstallation loads an installation by path IID and verifies it belongs to the app.
// Returns the installation or nil (with error already written to w).
func (s *Server) requireInstallation(w http.ResponseWriter, r *http.Request, appID string) *database.AppInstallation {
	iid := r.PathValue("iid")

	inst, err := s.DB.GetInstallation(iid)
	if err != nil {
		jsonError(w, "installation not found", http.StatusNotFound)
		return nil
	}
	if inst.AppID != appID {
		jsonError(w, "installation not found", http.StatusNotFound)
		return nil
	}
	return inst
}
