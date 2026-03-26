package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Supported OAuth provider names for validation.
var knownOAuthProviders = map[string]bool{
	"github": true, "linuxdo": true,
}

// GET /api/admin/config/oauth — get OAuth config (secrets masked)
func (s *Server) handleGetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	dbConf, err := s.Store.ListConfigByPrefix("oauth.")
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	type providerConfig struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Enabled      bool   `json:"enabled"`
		Source       string `json:"source"` // "db" or "env"
	}

	result := map[string]*providerConfig{}
	for name := range oauthProviderDefs {
		pc := &providerConfig{}

		// Check DB first
		if id := dbConf["oauth."+name+".client_id"]; id != "" {
			pc.ClientID = id
			pc.ClientSecret = maskSecret(dbConf["oauth."+name+".client_secret"])
			pc.Enabled = true
			pc.Source = "db"
		} else {
			// Check env fallback
			var envID, envSecret string
			switch name {
			case "github":
				envID = s.Config.GitHubClientID
				envSecret = s.Config.GitHubClientSecret
			case "linuxdo":
				envID = s.Config.LinuxDoClientID
				envSecret = s.Config.LinuxDoClientSecret
			}
			if envID != "" {
				pc.ClientID = envID
				pc.ClientSecret = maskSecret(envSecret)
				pc.Enabled = true
				pc.Source = "env"
			}
		}

		result[name] = pc
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PUT /api/admin/config/oauth/{provider} — set OAuth config for a provider
func (s *Server) handleSetOAuthConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	if !knownOAuthProviders[name] {
		jsonError(w, "unknown provider", http.StatusBadRequest)
		return
	}

	var req struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		jsonError(w, "client_id required", http.StatusBadRequest)
		return
	}

	if err := s.Store.SetConfig("oauth."+name+".client_id", req.ClientID); err != nil {
		jsonError(w, "save failed", http.StatusInternalServerError)
		return
	}
	if req.ClientSecret != "" {
		if err := s.Store.SetConfig("oauth."+name+".client_secret", req.ClientSecret); err != nil {
			jsonError(w, "save failed", http.StatusInternalServerError)
			return
		}
	}
	jsonOK(w)
}

// DELETE /api/admin/config/oauth/{provider} — remove OAuth config (revert to env)
func (s *Server) handleDeleteOAuthConfig(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	if !knownOAuthProviders[name] {
		jsonError(w, "unknown provider", http.StatusBadRequest)
		return
	}

	s.Store.DeleteConfig("oauth." + name + ".client_id")
	s.Store.DeleteConfig("oauth." + name + ".client_secret")
	jsonOK(w)
}

// GET /api/admin/config/ai — get global AI config
func (s *Server) handleGetAIConfig(w http.ResponseWriter, r *http.Request) {
	dbConf, err := s.Store.ListConfigByPrefix("ai.")
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}
	result := map[string]string{
		"base_url":      dbConf["ai.base_url"],
		"api_key":       maskSecret(dbConf["ai.api_key"]),
		"model":         dbConf["ai.model"],
		"system_prompt": dbConf["ai.system_prompt"],
		"max_history":   dbConf["ai.max_history"],
	}
	if dbConf["ai.api_key"] != "" {
		result["enabled"] = "true"
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PUT /api/admin/config/ai — set global AI config
func (s *Server) handleSetAIConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BaseURL      string `json:"base_url"`
		APIKey       string `json:"api_key"`
		Model        string `json:"model"`
		SystemPrompt string `json:"system_prompt"`
		MaxHistory   string `json:"max_history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	if req.BaseURL != "" {
		s.Store.SetConfig("ai.base_url", req.BaseURL)
	}
	if req.APIKey != "" {
		s.Store.SetConfig("ai.api_key", req.APIKey)
	}
	if req.Model != "" {
		s.Store.SetConfig("ai.model", req.Model)
	}
	// These can be set to empty to clear
	s.Store.SetConfig("ai.system_prompt", req.SystemPrompt)
	if req.MaxHistory != "" {
		s.Store.SetConfig("ai.max_history", req.MaxHistory)
	}
	jsonOK(w)
}

// DELETE /api/admin/config/ai — remove global AI config
func (s *Server) handleDeleteAIConfig(w http.ResponseWriter, r *http.Request) {
	s.Store.DeleteConfig("ai.base_url")
	s.Store.DeleteConfig("ai.api_key")
	s.Store.DeleteConfig("ai.model")
	jsonOK(w)
}

// GET /api/info — public endpoint to check which features are available
func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	globalAI, _ := s.Store.ListConfigByPrefix("ai.")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{
		"ai":      globalAI["ai.api_key"] != "",
		"storage": s.Config.StorageEndpoint != "",
	})
}

// GET /api/admin/config/registry — get registry config
func (s *Server) handleGetRegistryConfig(w http.ResponseWriter, r *http.Request) {
	enabled, _ := s.Store.GetConfig("registry.enabled")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"enabled": enabled,
	})
}

// PUT /api/admin/config/registry — set registry config
func (s *Server) handleSetRegistryConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled string `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Enabled != "true" && req.Enabled != "false" {
		jsonError(w, "enabled must be 'true' or 'false'", http.StatusBadRequest)
		return
	}
	if err := s.Store.SetConfig("registry.enabled", req.Enabled); err != nil {
		jsonError(w, "save failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-8) + s[len(s)-4:]
}
