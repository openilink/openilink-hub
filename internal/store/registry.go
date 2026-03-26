package store

type Registry struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
}

type RegistryStore interface {
	ListRegistries() ([]Registry, error)
	CreateRegistry(r *Registry) error
	UpdateRegistryEnabled(id string, enabled bool) error
	DeleteRegistry(id string) error
}
