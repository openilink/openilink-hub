package api

import (
	"encoding/json"
	"net/http"

	"github.com/openilink/openilink-hub/internal/registry"
)

// GET /api/marketplace — list all available apps from registries, merged with local installs
func (s *Server) handleMarketplace(w http.ResponseWriter, r *http.Request) {
	if s.Registry == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	// 1. Fetch apps from all registries
	registryApps, _ := s.Registry.ListApps()

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

// POST /api/marketplace/sync/{slug} — update a marketplace app to latest version from registry
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

	// Find the local app (must be a marketplace app with registry set)
	localApp, err := s.Store.GetAppBySlug(slug)
	if err != nil || localApp == nil || localApp.Registry == "" {
		jsonError(w, "app not installed from marketplace", http.StatusNotFound)
		return
	}

	// Compare versions
	if regApp.Version != "" && regApp.Version == localApp.Version {
		jsonOK(w)
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
		regApp.Tools,
		regApp.Events,
		regApp.Scopes,
	); err != nil {
		jsonError(w, "sync failed", http.StatusInternalServerError)
		return
	}

	jsonOK(w)
}
