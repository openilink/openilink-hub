package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/openilink/openilink-hub/internal/registry"
)

// GET /api/registry/v1/apps.json — public endpoint for this Hub to act as a registry
func (s *Server) handleRegistryApps(w http.ResponseWriter, r *http.Request) {
	// Check if registry is enabled
	enabled, _ := s.Store.GetConfig("registry.enabled")
	if enabled != "true" {
		http.NotFound(w, r)
		return
	}

	// List all apps with listing='listed'
	apps, err := s.Store.ListListedApps()
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}

	// Convert to registry manifest format
	var regApps []registry.App
	for _, app := range apps {
		regApps = append(regApps, registry.App{
			Slug:             app.Slug,
			Name:             app.Name,
			Description:      app.Description,
			Readme:           app.Readme,
			Version:          app.Version,
			Author:           app.OwnerName,
			IconURL:          app.IconURL,
			Homepage:         app.Homepage,
			WebhookURL:       app.WebhookURL,
			OAuthSetupURL:    app.OAuthSetupURL,
			OAuthRedirectURL: app.OAuthRedirectURL,
			Tools:            app.Tools,
			Events:           app.Events,
			Scopes:           app.Scopes,
		})
	}

	manifest := registry.Manifest{
		Version:   1,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Apps:      regApps,
	}
	if manifest.Apps == nil {
		manifest.Apps = []registry.App{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	json.NewEncoder(w).Encode(manifest)
}
