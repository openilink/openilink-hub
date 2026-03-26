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
	OAuthSetupURL       string          `json:"oauth_setup_url,omitempty"`
	OAuthRedirectURL    string          `json:"oauth_redirect_url,omitempty"`
	WebhookURL          string          `json:"webhook_url,omitempty"`
	WebhookSecret       string          `json:"-"` // never serialize; exposed explicitly where needed
	WebhookVerified     bool            `json:"webhook_verified"`
	Registry            string          `json:"registry,omitempty"`
	Version             string          `json:"version,omitempty"`
	Readme              string          `json:"readme,omitempty"`
	Guide               string          `json:"guide,omitempty"`
	Listing             string          `json:"listing"`
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
	Scopes    json.RawMessage `json:"scopes"`
	Enabled   bool            `json:"enabled"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`

	// Joined from apps table
	AppName          string `json:"app_name,omitempty"`
	AppSlug          string `json:"app_slug,omitempty"`
	AppIcon          string `json:"app_icon,omitempty"`
	AppIconURL       string `json:"app_icon_url,omitempty"`
	AppWebhookURL    string `json:"-"`
	AppWebhookSecret string `json:"-"`
	AppRegistry      string `json:"app_registry,omitempty"`
	AppReadme        string `json:"app_readme,omitempty"`
	AppGuide         string `json:"app_guide,omitempty"`
	BotName          string `json:"bot_name,omitempty"`
}

type AppStore interface {
	CreateApp(app *App) (*App, error)
	GetApp(id string) (*App, error)
	GetAppBySlug(slug, registry string) (*App, error)
	ListAppsByOwner(ownerID string) ([]App, error)
	ListListedApps() ([]App, error)
	ListAllApps() ([]App, error)
	ListMarketplaceApps() ([]App, error)
	UpdateApp(id string, name, description, icon, iconURL, homepage, oauthSetupURL, oauthRedirectURL string, tools, events, scopes json.RawMessage) error
	UpdateMarketplaceApp(id, name, description, iconURL, homepage, webhookURL, oauthSetupURL, oauthRedirectURL, version, guide string, tools, events, scopes json.RawMessage) error
	DeleteApp(id string) error
	InstallApp(appID, botID string) (*AppInstallation, error)
	GetInstallation(id string) (*AppInstallation, error)
	GetInstallationByToken(token string) (*AppInstallation, error)
	ListInstallationsByApp(appID string) ([]AppInstallation, error)
	ListInstallationsByBot(botID string) ([]AppInstallation, error)
	UpdateInstallation(id, handle string, config json.RawMessage, scopes json.RawMessage, enabled bool) error
	SetAppWebhookVerified(id string, verified bool) error
	UpdateAppWebhookURL(id, webhookURL string) error
	RegenerateInstallationToken(id string) (string, error)
	GetInstallationByHandle(botID, handle string) (*AppInstallation, error)
	DeleteInstallation(id string) error
	CreateOAuthCode(code, appID, botID, state, codeChallenge string) error
	ExchangeOAuthCode(code string) (appID, botID, codeChallenge string, err error)
	CleanExpiredOAuthCodes()
	RequestListing(id string) error
	ReviewListing(id string, approve bool, reason string) error
}
