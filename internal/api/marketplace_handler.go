package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/registry"
	"github.com/openilink/openilink-hub/internal/store"
)

// GET /api/marketplace — list all available apps from registries, merged with local installs
func (s *Server) handleMarketplace(w http.ResponseWriter, r *http.Request) {
	if s.Registry == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	// 1. Fetch apps from all registries
	registryApps, err := s.Registry.ListApps()
	if err != nil {
		slog.Error("marketplace: failed to fetch registry apps", "err", err)
		jsonError(w, "failed to fetch registry apps", http.StatusBadGateway)
		return
	}

	// 2. Get locally installed marketplace apps
	localApps, err := s.Store.ListMarketplaceApps()
	if err != nil {
		jsonError(w, "list local apps failed", http.StatusInternalServerError)
		return
	}

	// Build lookup: slug → local app
	localBySlug := make(map[string]struct {
		ID      string `json:"id"`
		Version string `json:"version"`
	})
	for _, la := range localApps {
		localBySlug[la.Slug] = struct {
			ID      string `json:"id"`
			Version string `json:"version"`
		}{ID: la.ID, Version: la.Version}
	}

	// 3. Merge
	type marketplaceEntry struct {
		registry.AppWithSource
		Installed       bool   `json:"installed"`
		LocalID         string `json:"local_id,omitempty"`
		UpdateAvailable bool   `json:"update_available"`
	}

	var result []marketplaceEntry
	for _, ra := range registryApps {
		entry := marketplaceEntry{AppWithSource: ra}
		if local, ok := localBySlug[ra.Slug]; ok {
			entry.Installed = true
			entry.LocalID = local.ID
			if ra.Version != "" && ra.Version != local.Version {
				entry.UpdateAvailable = true
			}
		}
		result = append(result, entry)
	}

	if result == nil {
		result = []marketplaceEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GET /api/marketplace/builtin — list all builtin apps
func (s *Server) handleBuiltinApps(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	apps, err := s.Store.ListMarketplaceApps()
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}

	installedIDs, err := s.Store.InstalledAppIDs(userID)
	if err != nil {
		slog.Warn("failed to load installed app IDs", "user_id", userID, "err", err)
	}

	type builtinEntry struct {
		store.App
		Installed bool `json:"installed"`
	}

	var result []builtinEntry
	for _, a := range apps {
		if a.Registry == "builtin" {
			result = append(result, builtinEntry{App: a, Installed: installedIDs[a.ID]})
		}
	}
	if result == nil {
		result = []builtinEntry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// POST /api/marketplace/sync/{slug} — sync a marketplace app from registry (create or update)
func (s *Server) handleMarketplaceSync(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		jsonError(w, "slug required", http.StatusBadRequest)
		return
	}

	if s.Registry == nil {
		jsonError(w, "no registry configured", http.StatusBadRequest)
		return
	}

	// Find the registry app
	regApp, err := s.Registry.GetApp(slug)
	if err != nil || regApp == nil {
		jsonError(w, "app not found in registry", http.StatusNotFound)
		return
	}

	// Find or create the local app
	localApp, err := s.Store.GetAppBySlug(slug, regApp.RegistryURL)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.Error("marketplace: failed to lookup app", "slug", slug, "err", err)
		jsonError(w, "lookup app failed", http.StatusInternalServerError)
		return
	}
	if localApp == nil {
		// First install: create local app from registry data
		localApp, err = s.Store.CreateApp(&store.App{
			Name:             regApp.Name,
			Slug:             regApp.Slug,
			Description:      regApp.Description,
			IconURL:          regApp.IconURL,
			Homepage:         regApp.Homepage,
			WebhookURL:       regApp.WebhookURL,
			OAuthSetupURL:    regApp.OAuthSetupURL,
			OAuthRedirectURL: regApp.OAuthRedirectURL,
			Tools:            regApp.Tools,
			Events:           regApp.Events,
			Scopes:           regApp.Scopes,
			Registry:         regApp.RegistryURL,
			Version:          regApp.Version,
			Readme:           regApp.Readme,
			Guide:            regApp.Guide,
			Listing:          "listed",
		})
		if err != nil {
			slog.Error("marketplace: failed to create app", "slug", slug, "err", err)
			jsonError(w, "create app failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(localApp)
		return
	}

	// Already installed and up to date
	if regApp.Version != "" && regApp.Version == localApp.Version {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(localApp)
		return
	}

	// Update local app fields from registry manifest
	if err := s.Store.UpdateMarketplaceApp(
		localApp.ID,
		regApp.Name,
		regApp.Description,
		regApp.IconURL,
		regApp.Homepage,
		regApp.WebhookURL,
		regApp.OAuthSetupURL,
		regApp.OAuthRedirectURL,
		regApp.Version,
		regApp.Readme,
		regApp.Guide,
		regApp.Tools,
		regApp.Events,
		regApp.Scopes,
	); err != nil {
		slog.Error("marketplace: failed to update app", "slug", slug, "err", err)
		jsonError(w, "sync failed", http.StatusInternalServerError)
		return
	}

	// Return updated app
	localApp, err = s.Store.GetApp(localApp.ID)
	if err != nil {
		jsonError(w, "fetch updated app failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(localApp)
}
