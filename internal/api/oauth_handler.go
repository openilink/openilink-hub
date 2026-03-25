package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openilink/openilink-hub/internal/auth"
	"github.com/openilink/openilink-hub/internal/config"
	"github.com/openilink/openilink-hub/internal/store"
)

// --- OAuth provider definitions ---

type oauthProvider struct {
	Name         string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	ClientID     string
	ClientSecret string
	Scopes       string
}

// oauthProviderDefs defines the static parts of each provider.
var oauthProviderDefs = map[string]struct {
	AuthURL, TokenURL, UserInfoURL, Scopes string
}{
	"github": {
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
		Scopes:      "read:user user:email",
	},
	"linuxdo": {
		AuthURL:     "https://connect.linux.do/oauth2/authorize",
		TokenURL:    "https://connect.linux.do/oauth2/token",
		UserInfoURL: "https://connect.linux.do/api/user",
		Scopes:      "",
	},
}

// oauthProviders returns enabled OAuth providers.
// DB config takes precedence over env vars.
func (s *Server) oauthProviders() map[string]*oauthProvider {
	dbConf, _ := s.Store.ListConfigByPrefix("oauth.")
	providers := map[string]*oauthProvider{}

	for name, def := range oauthProviderDefs {
		clientID := dbConf["oauth."+name+".client_id"]
		clientSecret := dbConf["oauth."+name+".client_secret"]

		// Fallback to env config
		if clientID == "" {
			switch name {
			case "github":
				clientID = s.Config.GitHubClientID
				clientSecret = s.Config.GitHubClientSecret
			case "linuxdo":
				clientID = s.Config.LinuxDoClientID
				clientSecret = s.Config.LinuxDoClientSecret
			}
		}

		if clientID == "" {
			continue
		}
		providers[name] = &oauthProvider{
			Name:         name,
			AuthURL:      def.AuthURL,
			TokenURL:     def.TokenURL,
			UserInfoURL:  def.UserInfoURL,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Scopes:       def.Scopes,
		}
	}
	return providers
}

// --- OAuth state store (in-memory, short-lived) ---

type oauthStateEntry struct {
	CreatedAt time.Time
	BindUID   string // non-empty = bind mode (link to existing user)
}

type oauthStateStore struct {
	mu    sync.Mutex
	store map[string]*oauthStateEntry
}

func newOAuthStateStore() *oauthStateStore {
	return &oauthStateStore{store: make(map[string]*oauthStateEntry)}
}

func (s *oauthStateStore) Generate(bindUID string) string {
	b := make([]byte, 16)
	rand.Read(b)
	state := hex.EncodeToString(b)
	s.mu.Lock()
	s.store[state] = &oauthStateEntry{CreatedAt: time.Now(), BindUID: bindUID}
	s.mu.Unlock()
	return state
}

func (s *oauthStateStore) Validate(state string) (*oauthStateEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.store[state]
	if !ok {
		return nil, false
	}
	delete(s.store, state)
	if time.Since(entry.CreatedAt) > 10*time.Minute {
		return nil, false
	}
	return entry, true
}

// --- Handlers ---

// GET /api/auth/oauth/providers — list enabled providers
func (s *Server) handleOAuthProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.oauthProviders()
	names := make([]string, 0, len(providers))
	for name := range providers {
		names = append(names, name)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"providers": names})
}

// GET /api/auth/oauth/{provider} — redirect to OAuth provider (login flow)
func (s *Server) handleOAuthRedirect(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	providers := s.oauthProviders()
	p, ok := providers[name]
	if !ok {
		jsonError(w, "unknown provider", http.StatusBadRequest)
		return
	}

	state := s.OAuthStates.Generate("")

	params := url.Values{
		"client_id":     {p.ClientID},
		"redirect_uri":  {s.Config.RPOrigin + "/api/auth/oauth/" + name + "/callback"},
		"state":         {state},
		"response_type": {"code"},
	}
	if p.Scopes != "" {
		params.Set("scope", p.Scopes)
	}

	http.Redirect(w, r, p.AuthURL+"?"+params.Encode(), http.StatusFound)
}

// GET /api/auth/oauth/{provider}/callback — handle OAuth callback (login or bind)
func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("provider")
	providers := s.oauthProviders()
	p, ok := providers[name]
	if !ok {
		jsonError(w, "unknown provider", http.StatusBadRequest)
		return
	}

	// Validate state
	state := r.URL.Query().Get("state")
	entry, valid := s.OAuthStates.Validate(state)
	if !valid {
		jsonError(w, "invalid oauth state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		jsonError(w, "no code provided", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	accessToken, err := exchangeCode(p, s.Config.RPOrigin+"/api/auth/oauth/"+name+"/callback", code)
	if err != nil {
		slog.Error("oauth token exchange failed", "provider", name, "err", err)
		jsonError(w, "token exchange failed", http.StatusBadGateway)
		return
	}

	// Get user info
	providerID, username, email, avatarURL, err := fetchUserInfo(p, accessToken)
	if err != nil {
		slog.Error("oauth user info failed", "provider", name, "err", err)
		jsonError(w, "failed to get user info", http.StatusBadGateway)
		return
	}

	// Bind mode: link OAuth account to existing logged-in user
	if entry.BindUID != "" {
		existing, err := s.Store.GetOAuthAccount(name, providerID)
		if err == nil && existing.UserID != entry.BindUID {
			// Check if the linked user still exists — clean up stale records
			if _, userErr := s.Store.GetUserByID(existing.UserID); userErr != nil {
				slog.Info("oauth cleanup stale binding", "provider", name, "old_user", existing.UserID)
				s.Store.DeleteOAuthAccount(name, providerID)
			} else {
				http.Redirect(w, r, "/dashboard/settings?oauth_error=already_linked", http.StatusFound)
				return
			}
		}

		if err == sql.ErrNoRows || (err == nil && existing.UserID != entry.BindUID) {
			if err := s.Store.CreateOAuthAccount(&store.OAuthAccount{
				Provider:   name,
				ProviderID: providerID,
				UserID:     entry.BindUID,
				Username:   username,
				AvatarURL:  avatarURL,
			}); err != nil {
				slog.Error("oauth bind failed", "provider", name, "err", err)
				http.Redirect(w, r, "/dashboard/settings?oauth_error=bind_failed", http.StatusFound)
				return
			}
		}

		http.Redirect(w, r, "/dashboard/settings?oauth_bound="+name, http.StatusFound)
		return
	}

	// Login mode: find or create user
	user, err := s.findOrCreateOAuthUser(name, providerID, username, email, avatarURL)
	if err != nil {
		slog.Error("oauth user creation failed", "provider", name, "err", err)
		jsonError(w, "login failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if user.Status != store.StatusActive {
		jsonError(w, "account disabled", http.StatusForbidden)
		return
	}

	token, _ := auth.CreateSession(s.Store, user.ID)
	setSessionCookie(w, token)

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// GET /api/auth/oauth/{provider}/bind — start bind flow (protected)
func (s *Server) handleOAuthBind(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	name := r.PathValue("provider")
	providers := s.oauthProviders()
	p, ok := providers[name]
	if !ok {
		jsonError(w, "unknown provider", http.StatusBadRequest)
		return
	}

	state := s.OAuthStates.Generate(userID)

	params := url.Values{
		"client_id":     {p.ClientID},
		"redirect_uri":  {s.Config.RPOrigin + "/api/auth/oauth/" + name + "/callback"},
		"state":         {state},
		"response_type": {"code"},
	}
	if p.Scopes != "" {
		params.Set("scope", p.Scopes)
	}

	http.Redirect(w, r, p.AuthURL+"?"+params.Encode(), http.StatusFound)
}

// GET /api/auth/oauth/accounts — list linked OAuth accounts for current user
func (s *Server) handleOAuthAccounts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	accounts, err := s.Store.ListOAuthAccountsByUser(userID)
	if err != nil {
		jsonError(w, "list failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

// DELETE /api/auth/oauth/accounts/{provider} — unlink an OAuth account
func (s *Server) handleOAuthUnbind(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	providerName := r.PathValue("provider")

	accounts, err := s.Store.ListOAuthAccountsByUser(userID)
	if err != nil {
		jsonError(w, "query failed", http.StatusInternalServerError)
		return
	}

	// Find the account to unlink
	var target *store.OAuthAccount
	for _, a := range accounts {
		if a.Provider == providerName {
			target = &a
			break
		}
	}
	if target == nil {
		jsonError(w, "not linked", http.StatusNotFound)
		return
	}

	// Ensure user has another login method (password or other OAuth)
	user, err := s.Store.GetUserByID(userID)
	if err != nil {
		jsonError(w, "user not found", http.StatusInternalServerError)
		return
	}
	otherOAuth := 0
	for _, a := range accounts {
		if a.Provider != providerName {
			otherOAuth++
		}
	}
	if user.PasswordHash == "" && otherOAuth == 0 {
		jsonError(w, "cannot unlink last login method", http.StatusBadRequest)
		return
	}

	if err := s.Store.DeleteOAuthAccount(target.Provider, target.ProviderID); err != nil {
		jsonError(w, "unlink failed", http.StatusInternalServerError)
		return
	}
	jsonOK(w)
}

// findOrCreateOAuthUser links an OAuth account to an existing user or creates a new one.
func (s *Server) findOrCreateOAuthUser(provider, providerID, username, email, avatarURL string) (*store.User, error) {
	// Check if OAuth account already linked
	oa, err := s.Store.GetOAuthAccount(provider, providerID)
	if err == nil {
		return s.Store.GetUserByID(oa.UserID)
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Try to find existing user by email
	var user *store.User
	if email != "" {
		user, err = s.Store.GetUserByEmail(email)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
	}

	// Create new user if not found
	if user == nil {
		displayName := username
		role := store.RoleMember
		count, _ := s.Store.UserCount()
		if count == 0 {
			role = store.RoleSuperAdmin
		}
		uname := provider + "_" + username
		if _, err := s.Store.GetUserByUsername(uname); err == nil {
			uname = provider + "_" + username + "_" + providerID
		}
		user, err = s.Store.CreateUserFull(uname, email, displayName, "", role)
		if err != nil {
			return nil, fmt.Errorf("create user: %w", err)
		}
	}

	// Link OAuth account
	if err := s.Store.CreateOAuthAccount(&store.OAuthAccount{
		Provider:   provider,
		ProviderID: providerID,
		UserID:     user.ID,
		Username:   username,
		AvatarURL:  avatarURL,
	}); err != nil {
		return nil, fmt.Errorf("link oauth account: %w", err)
	}

	return user, nil
}

// --- OAuth HTTP helpers ---

func exchangeCode(p *oauthProvider, redirectURI, code string) (string, error) {
	data := url.Values{
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"grant_type":    {"authorization_code"},
	}

	req, _ := http.NewRequest("POST", p.TokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		vals, _ := url.ParseQuery(string(body))
		result.AccessToken = vals.Get("access_token")
		result.Error = vals.Get("error")
	}
	if result.Error != "" {
		return "", fmt.Errorf("oauth error: %s", result.Error)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("no access_token in response")
	}
	return result.AccessToken, nil
}

func fetchUserInfo(p *oauthProvider, accessToken string) (providerID, username, email, avatarURL string, err error) {
	req, _ := http.NewRequest("GET", p.UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	switch p.Name {
	case "github":
		var u struct {
			ID        int    `json:"id"`
			Login     string `json:"login"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar_url"`
		}
		json.Unmarshal(body, &u)
		return strconv.Itoa(u.ID), u.Login, u.Email, u.AvatarURL, nil

	case "linuxdo":
		var u struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Avatar   string `json:"avatar_url"`
			Name     string `json:"name"`
		}
		json.Unmarshal(body, &u)
		name := u.Username
		if name == "" {
			name = u.Name
		}
		return strconv.Itoa(u.ID), name, u.Email, u.Avatar, nil

	default:
		return "", "", "", "", fmt.Errorf("unknown provider: %s", p.Name)
	}
}

// SetupOAuth initializes the OAuth state store. Call from main.
func SetupOAuth(cfg *config.Config) *oauthStateStore {
	return newOAuthStateStore()
}
