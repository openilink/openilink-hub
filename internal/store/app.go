package store

import "encoding/json"

type App struct {
	ID                  string          `json:"id"`
	OwnerID             string          `json:"owner_id"`
	Name                string          `json:"name"`
	Slug                string          `json:"slug"`
	Description         string          `json:"description"`
	Icon                string          `json:"icon,omitempty"`
	IconURL             string          `json:"icon_url,omitempty"`
	Homepage            string          `json:"homepage,omitempty"`
	Tools               json.RawMessage `json:"tools"`
	Events              json.RawMessage `json:"events"`
	Scopes              json.RawMessage `json:"scopes"`
	SetupURL            string          `json:"setup_url,omitempty"`
	RedirectURL         string          `json:"redirect_url,omitempty"`
	ClientSecret        string          `json:"client_secret,omitempty"`
	RequestURL          string          `json:"request_url,omitempty"`
	SigningSecret       string          `json:"signing_secret,omitempty"`
	URLVerified         bool            `json:"url_verified"`
	Listed              bool            `json:"listed"`
	ListingStatus       string          `json:"listing_status,omitempty"`
	ListingRejectReason string          `json:"listing_reject_reason,omitempty"`
	Status              string          `json:"status"`
	CreatedAt           int64           `json:"created_at"`
	UpdatedAt           int64           `json:"updated_at"`

	// Joined
	OwnerName string `json:"owner_name,omitempty"`
}

type AppTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Command     string          `json:"command,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type AppInstallation struct {
	ID        string          `json:"id"`
	AppID     string          `json:"app_id"`
	BotID     string          `json:"bot_id"`
	AppToken  string          `json:"app_token"`
	Handle    string          `json:"handle,omitempty"`
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`

	// Joined from apps table
	AppName          string `json:"app_name,omitempty"`
	AppSlug          string `json:"app_slug,omitempty"`
	AppIcon          string `json:"app_icon,omitempty"`
	AppIconURL       string `json:"app_icon_url,omitempty"`
	AppRequestURL    string `json:"-"`
	AppSigningSecret string `json:"-"`
	BotName          string `json:"bot_name,omitempty"`
}

type AppStore interface {
	CreateApp(app *App) (*App, error)
	GetApp(id string) (*App, error)
	GetAppBySlug(slug string) (*App, error)
	ListAppsByOwner(ownerID string) ([]App, error)
	ListListedApps() ([]App, error)
	ListAllApps() ([]App, error)
	SetAppListed(id string, listed bool) error
	UpdateApp(id string, name, description, icon, iconURL, homepage, setupURL, redirectURL string, tools, events, scopes json.RawMessage) error
	DeleteApp(id string) error
	InstallApp(appID, botID string) (*AppInstallation, error)
	GetInstallation(id string) (*AppInstallation, error)
	GetInstallationByToken(token string) (*AppInstallation, error)
	ListInstallationsByApp(appID string) ([]AppInstallation, error)
	ListInstallationsByBot(botID string) ([]AppInstallation, error)
	UpdateInstallation(id, handle string, config json.RawMessage, enabled bool) error
	SetAppURLVerified(id string, verified bool) error
	UpdateAppRequestURL(id, requestURL string) error
	RegenerateInstallationToken(id string) (string, error)
	GetInstallationByHandle(botID, handle string) (*AppInstallation, error)
	DeleteInstallation(id string) error
	CreateOAuthCode(code, appID, botID, state string) error
	ExchangeOAuthCode(code string) (appID, botID string, err error)
	CleanExpiredOAuthCodes()
	RequestListing(id string) error
	ReviewListing(id string, approve bool, reason string) error
}
