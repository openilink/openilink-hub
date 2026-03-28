package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCProviderConfig is the stored configuration for a custom OIDC provider.
type OIDCProviderConfig struct {
	Slug         string `json:"slug"`
	DisplayName  string `json:"display_name"`
	IssuerURL    string `json:"issuer_url"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Scopes       string `json:"scopes"`
	// Discovered endpoints (cached at save time)
	AuthURL     string `json:"auth_url"`
	TokenURL    string `json:"token_url"`
	UserInfoURL string `json:"userinfo_url"`
}

// slugRe is declared in app_handler.go

// oidcProviders loads all OIDC provider configs from the DB.
func (s *Server) oidcProviders() map[string]*OIDCProviderConfig {
	dbConf, err := s.Store.ListConfigByPrefix("oidc.")
	if err != nil {
		return nil
	}
	providers := map[string]*OIDCProviderConfig{}
	for key, val := range dbConf {
		if !strings.HasSuffix(key, ".config") {
			continue
		}
		var cfg OIDCProviderConfig
		if err := json.Unmarshal([]byte(val), &cfg); err != nil {
			continue
		}
		if cfg.ClientID == "" || cfg.IssuerURL == "" {
			continue
		}
		providers[cfg.Slug] = &cfg
	}
	return providers
}

// discoverOIDC fetches OIDC Discovery metadata from the issuer URL.
func discoverOIDC(ctx context.Context, issuerURL string) (authURL, tokenURL, userinfoURL string, err error) {
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return "", "", "", fmt.Errorf("OIDC discovery failed: %w", err)
	}
	ep := provider.Endpoint()
	authURL = ep.AuthURL
	tokenURL = ep.TokenURL

	// Extract userinfo_endpoint from provider claims
	var claims struct {
		UserInfoURL string `json:"userinfo_endpoint"`
	}
	if err := provider.Claims(&claims); err == nil {
		userinfoURL = claims.UserInfoURL
	}
	// Validate discovered endpoints use HTTPS to prevent SSRF
	for _, u := range []string{authURL, tokenURL, userinfoURL} {
		if u != "" && !strings.HasPrefix(u, "https://") {
			return "", "", "", fmt.Errorf("discovered endpoint is not HTTPS: %s", u)
		}
	}
	return authURL, tokenURL, userinfoURL, nil
}

// oidcExchangeAndIdentify exchanges the authorization code and extracts user identity.
func (s *Server) oidcExchangeAndIdentify(ctx context.Context, cfg *OIDCProviderConfig, redirectURI, code string) (providerID, username, email, avatarURL string, err error) {

	scopes := strings.Fields(cfg.Scopes)
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	oauth2Cfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
		RedirectURL: redirectURI,
		Scopes:      scopes,
	}

	token, err := oauth2Cfg.Exchange(ctx, code)
	if err != nil {
		return "", "", "", "", fmt.Errorf("token exchange failed: %w", err)
	}

	// Try to verify ID token first
	rawIDToken, ok := token.Extra("id_token").(string)
	if ok && rawIDToken != "" {
		provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
		if err != nil {
			return "", "", "", "", fmt.Errorf("OIDC provider init failed: %w", err)
		}
		verifier := provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
		idToken, err := verifier.Verify(ctx, rawIDToken)
		if err != nil {
			return "", "", "", "", fmt.Errorf("id_token verification failed: %w", err)
		}
		var claims struct {
			Sub               string `json:"sub"`
			PreferredUsername string `json:"preferred_username"`
			Name              string `json:"name"`
			Email             string `json:"email"`
			Picture           string `json:"picture"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return "", "", "", "", fmt.Errorf("id_token claims decode failed: %w", err)
		}
		if claims.Sub == "" {
			return "", "", "", "", fmt.Errorf("id_token missing sub claim")
		}
		uname := claims.PreferredUsername
		if uname == "" {
			uname = claims.Name
		}
		if uname == "" {
			uname = claims.Email
		}
		if uname == "" {
			uname = claims.Sub
		}
		return claims.Sub, uname, claims.Email, claims.Picture, nil
	}

	// Fallback: use userinfo endpoint (only when no id_token present)
	if cfg.UserInfoURL != "" {
		client := oauth2Cfg.Client(ctx, token)
		resp, err := client.Get(cfg.UserInfoURL)
		if err != nil {
			return "", "", "", "", fmt.Errorf("userinfo request failed: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", "", "", "", fmt.Errorf("userinfo endpoint returned %d", resp.StatusCode)
		}
		var claims map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
			return "", "", "", "", fmt.Errorf("userinfo decode failed: %w", err)
		}
		subVal, _ := claims["sub"]
		if subVal == nil {
			return "", "", "", "", fmt.Errorf("userinfo missing sub claim")
		}
		sub := fmt.Sprintf("%v", subVal)
		uname, _ := claims["preferred_username"].(string)
		if uname == "" {
			uname, _ = claims["name"].(string)
		}
		if uname == "" {
			uname, _ = claims["email"].(string)
		}
		if uname == "" {
			uname = sub
		}
		emailVal, _ := claims["email"].(string)
		picture, _ := claims["picture"].(string)
		return sub, uname, emailVal, picture, nil
	}

	return "", "", "", "", fmt.Errorf("no id_token or userinfo_endpoint available")
}

// --- OIDC redirect / callback helpers (called from oauth_handler.go) ---

func (s *Server) handleOIDCRedirect(w http.ResponseWriter, r *http.Request, slug, bindUID string) {
	providers := s.oidcProviders()
	cfg, ok := providers[slug]
	if !ok {
		jsonError(w, "unknown OIDC provider", http.StatusBadRequest)
		return
	}

	state := s.OAuthStates.Generate(bindUID)
	providerName := "oidc_" + slug

	params := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {s.Config.RPOrigin + "/api/auth/oauth/" + providerName + "/callback"},
		"state":         {state},
		"response_type": {"code"},
	}
	scopes := cfg.Scopes
	if scopes == "" {
		scopes = "openid profile email"
	} else {
		hasOpenID := false
		for _, s := range strings.Fields(scopes) {
			if s == "openid" {
				hasOpenID = true
				break
			}
		}
		if !hasOpenID {
			scopes = "openid " + scopes
		}
	}
	params.Set("scope", scopes)

	http.Redirect(w, r, cfg.AuthURL+"?"+params.Encode(), http.StatusFound)
}

func (s *Server) handleOIDCCallback(w http.ResponseWriter, r *http.Request, providerName, slug string) {
	providers := s.oidcProviders()
	cfg, ok := providers[slug]
	if !ok {
		jsonError(w, "unknown OIDC provider", http.StatusBadRequest)
		return
	}

	state := r.URL.Query().Get("state")
	entry, valid := s.OAuthStates.Validate(state)
	if !valid {
		jsonError(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	// Handle provider-returned OAuth errors (e.g. access_denied)
	if oauthErr := r.URL.Query().Get("error"); oauthErr != "" {
		desc := r.URL.Query().Get("error_description")
		slog.Warn("oidc provider returned error", "provider", providerName, "error", oauthErr, "description", desc)
		http.Redirect(w, r, "/login?error=oauth_"+oauthErr, http.StatusFound)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		jsonError(w, "no code provided", http.StatusBadRequest)
		return
	}

	redirectURI := s.Config.RPOrigin + "/api/auth/oauth/" + providerName + "/callback"
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	providerID, username, email, avatarURL, err := s.oidcExchangeAndIdentify(ctx, cfg, redirectURI, code)
	if err != nil {
		slog.Error("oidc exchange failed", "provider", providerName, "err", err)
		jsonError(w, "OIDC login failed", http.StatusBadGateway)
		return
	}

	s.completeOAuthFlow(w, r, entry, providerName, providerID, username, email, avatarURL)
}

// --- Admin CRUD handlers ---

// GET /api/admin/config/oidc — list all OIDC providers (secrets masked)
func (s *Server) handleGetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	providers := s.oidcProviders()
	// Sort by slug for stable ordering
	sorted := make([]*OIDCProviderConfig, 0, len(providers))
	for _, cfg := range providers {
		sorted = append(sorted, cfg)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Slug < sorted[j].Slug })

	result := make([]map[string]any, 0, len(sorted))
	for _, cfg := range sorted {
		result = append(result, map[string]any{
			"slug":          cfg.Slug,
			"display_name":  cfg.DisplayName,
			"issuer_url":    cfg.IssuerURL,
			"client_id":     cfg.ClientID,
			"client_secret": maskSecret(cfg.ClientSecret),
			"scopes":        cfg.Scopes,
			"auth_url":      cfg.AuthURL,
			"token_url":     cfg.TokenURL,
			"userinfo_url":  cfg.UserInfoURL,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// PUT /api/admin/config/oidc/{slug} — create or update an OIDC provider
func (s *Server) handleSetOIDCConfig(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !slugRe.MatchString(slug) || len(slug) > 32 {
		jsonError(w, "invalid slug (lowercase alphanumeric and hyphens only)", http.StatusBadRequest)
		return
	}
	if knownOAuthProviders[slug] {
		jsonError(w, "reserved slug", http.StatusBadRequest)
		return
	}

	var req struct {
		DisplayName  string `json:"display_name"`
		IssuerURL    string `json:"issuer_url"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		Scopes       string `json:"scopes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.IssuerURL == "" || req.ClientID == "" {
		jsonError(w, "issuer_url and client_id are required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(req.IssuerURL, "https://") {
		jsonError(w, "issuer_url must use HTTPS", http.StatusBadRequest)
		return
	}
	if req.DisplayName == "" {
		req.DisplayName = slug
	}

	// Check existing config for issuer pinning and secret preservation
	existingRaw, _ := s.Store.GetConfig("oidc." + slug + ".config")
	if existingRaw != "" {
		var existing OIDCProviderConfig
		if json.Unmarshal([]byte(existingRaw), &existing) == nil {
			// Prevent issuer change — sub is only unique within an issuer,
			// so changing it would break existing user identity links.
			if existing.IssuerURL != "" && existing.IssuerURL != req.IssuerURL {
				jsonError(w, "cannot change issuer_url for existing provider; delete and recreate instead", http.StatusBadRequest)
				return
			}
			// Preserve existing secret if omitted or masked value is sent back
			if req.ClientSecret == "" || req.ClientSecret == maskSecret(existing.ClientSecret) {
				req.ClientSecret = existing.ClientSecret
			}
		}
	}

	// Run OIDC Discovery
	authURL, tokenURL, userinfoURL, err := discoverOIDC(r.Context(), req.IssuerURL)
	if err != nil {
		slog.Error("OIDC discovery failed", "issuer", req.IssuerURL, "err", err)
		jsonError(w, "OIDC Discovery failed, check issuer URL", http.StatusBadRequest)
		return
	}

	cfg := &OIDCProviderConfig{
		Slug:         slug,
		DisplayName:  req.DisplayName,
		IssuerURL:    req.IssuerURL,
		ClientID:     req.ClientID,
		ClientSecret: req.ClientSecret,
		Scopes:       req.Scopes,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
		UserInfoURL:  userinfoURL,
	}

	data, _ := json.Marshal(cfg)
	if err := s.Store.SetConfig("oidc."+slug+".config", string(data)); err != nil {
		jsonError(w, "save failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// DELETE /api/admin/config/oidc/{slug} — remove an OIDC provider
func (s *Server) handleDeleteOIDCConfig(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !slugRe.MatchString(slug) {
		jsonError(w, "invalid slug", http.StatusBadRequest)
		return
	}
	if err := s.Store.DeleteConfig("oidc." + slug + ".config"); err != nil {
		jsonError(w, "delete failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}
