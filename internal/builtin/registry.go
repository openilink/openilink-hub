package builtin

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/openilink/openilink-hub/internal/app"
	"github.com/openilink/openilink-hub/internal/store"
)

// Handler processes events for a builtin app.
type Handler interface {
	HandleEvent(inst *store.AppInstallation, event *app.Event) error
}

// AppManifest defines a builtin app's metadata.
type AppManifest struct {
	Slug         string          `json:"slug"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Icon         string          `json:"icon"`
	Readme       string          `json:"readme"`
	Guide        string          `json:"guide"`
	Scopes       []string        `json:"scopes"`
	Events       []string        `json:"events"`
	Tools        []store.AppTool `json:"tools"`
	ConfigSchema json.RawMessage `json:"config_schema"`
}

var (
	manifests = map[string]AppManifest{}
	handlers  = map[string]Handler{}
)

// Register adds a builtin app with its manifest and handler.
func Register(manifest AppManifest, handler Handler) {
	manifests[manifest.Slug] = manifest
	if handler != nil {
		handlers[manifest.Slug] = handler
	}
}

// Get returns the handler for a builtin app slug.
func Get(slug string) Handler {
	return handlers[slug]
}

// Manifests returns all registered builtin app manifests.
func Manifests() map[string]AppManifest {
	return manifests
}

// SeedApps creates or updates builtin App records in the store.
func SeedApps(s store.Store) error {
	for _, m := range manifests {
		existing, _ := s.GetAppBySlug(m.Slug, "builtin")
		scopes, _ := json.Marshal(m.Scopes)
		events, _ := json.Marshal(m.Events)
		tools, _ := json.Marshal(m.Tools)

		if existing == nil {
			_, err := s.CreateApp(&store.App{
				Slug:         m.Slug,
				Name:         m.Name,
				Description:  m.Description,
				Icon:         m.Icon,
				Readme:       m.Readme,
				Guide:        m.Guide,
				Scopes:       scopes,
				Events:       events,
				Tools:        tools,
				ConfigSchema: string(m.ConfigSchema),
				Registry:     "builtin",
				Listing:      "listed",
			})
			if err != nil {
				return fmt.Errorf("seed builtin app %s: %w", m.Slug, err)
			}
			slog.Info("seeded builtin app", "slug", m.Slug)
		} else {
			// Update existing builtin app to match manifest
			err := s.UpdateMarketplaceApp(existing.ID,
				m.Name, m.Description, "", "",
				"", "", "", "", m.Readme, m.Guide,
				tools, events, scopes,
			)
			if err != nil {
				slog.Warn("failed to update builtin app", "slug", m.Slug, "err", err)
			}
		}
	}
	return nil
}
