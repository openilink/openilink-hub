package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/openilink/openilink-hub/internal/store"
)

// GET /api/admin/registries — list all registry sources
func (s *Server) handleListRegistries(w http.ResponseWriter, r *http.Request) {
	registries, err := s.Store.ListRegistries()
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	if registries == nil {
		registries = []store.Registry{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(registries)
}

// POST /api/admin/registries — add a new registry source
func (s *Server) handleCreateRegistry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.URL == "" {
		jsonError(w, "name and url required", http.StatusBadRequest)
		return
	}

	reg := &store.Registry{
		Name:    req.Name,
		URL:     req.URL,
		Enabled: true,
	}
	if err := s.Store.CreateRegistry(reg); err != nil {
		jsonError(w, "create failed", http.StatusInternalServerError)
		return
	}

	// Refresh registry client sources
	s.refreshRegistrySources()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reg)
}

// PUT /api/admin/registries/{id} — update a registry source (enable/disable)
func (s *Server) handleUpdateRegistry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Enabled == nil {
		jsonError(w, "enabled field required", http.StatusBadRequest)
		return
	}

	if err := s.Store.UpdateRegistryEnabled(id, *req.Enabled); err != nil {
		jsonError(w, "update failed", http.StatusInternalServerError)
		return
	}

	// Refresh registry client sources
	s.refreshRegistrySources()

	jsonOK(w)
}

// DELETE /api/admin/registries/{id} — remove a registry source
func (s *Server) handleDeleteRegistry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	if err := s.Store.DeleteRegistry(id); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}

	// Refresh registry client sources
	s.refreshRegistrySources()

	jsonOK(w)
}

// refreshRegistrySources reloads registry sources from DB into the registry client.
// Returns an error if the reload fails so callers can decide how to handle it.
func (s *Server) refreshRegistrySources() error {
	if s.Registry == nil {
		return nil
	}
	registries, err := s.Store.ListRegistries()
	if err != nil {
		slog.Error("refreshRegistrySources: failed to list registries", "err", err)
		return err
	}
	var sources []struct{ Name, URL string }
	for _, reg := range registries {
		if reg.Enabled {
			sources = append(sources, struct{ Name, URL string }{Name: reg.Name, URL: reg.URL})
		}
	}
	s.Registry.SetSources(sources)
	return nil
}
