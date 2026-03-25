package database

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// App is the definition of an app.
type App struct {
	ID          string          `json:"id"`
	OwnerID     string          `json:"owner_id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Icon        string          `json:"icon,omitempty"`      // emoji icon
	IconURL     string          `json:"icon_url,omitempty"`  // image URL icon
	Homepage    string          `json:"homepage,omitempty"`
	Tools       json.RawMessage `json:"tools"`
	Events      json.RawMessage `json:"events"`
	Scopes      json.RawMessage `json:"scopes"`
	SetupURL      string `json:"setup_url,omitempty"`
	RedirectURL   string `json:"redirect_url,omitempty"`
	ClientSecret  string `json:"client_secret,omitempty"`
	RequestURL    string `json:"request_url,omitempty"`
	SigningSecret string `json:"signing_secret,omitempty"`
	URLVerified   bool   `json:"url_verified"`
	Listed             bool   `json:"listed"`
	ListingStatus      string `json:"listing_status,omitempty"`       // pending / rejected / ""
	ListingRejectReason string `json:"listing_reject_reason,omitempty"`
	Status             string `json:"status"`
	CreatedAt          int64  `json:"created_at"`
	UpdatedAt          int64  `json:"updated_at"`

	// Joined
	OwnerName string `json:"owner_name,omitempty"`
}

// AppTool is a tool (function) exposed by an app.
// It can optionally be triggered as a slash command via the Command field.
type AppTool struct {
	Name        string          `json:"name"`                  // tool identifier, e.g. "list_prs"
	Description string          `json:"description"`           // what it does (used by LLM)
	Command     string          `json:"command,omitempty"`     // optional slash trigger, e.g. "pr"
	Parameters  json.RawMessage `json:"parameters,omitempty"`  // JSON Schema for structured args
}

// AppInstallation is a per-bot installation of an app.
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
	AppRequestURL    string `json:"-"` // from apps.request_url
	AppSigningSecret string `json:"-"` // from apps.signing_secret
	BotName          string `json:"bot_name,omitempty"`
}

func generateToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateApp creates a new app.
func (db *DB) CreateApp(app *App) (*App, error) {
	app.ID = uuid.New().String()
	if app.Tools == nil {
		app.Tools = json.RawMessage("[]")
	}
	if app.Events == nil {
		app.Events = json.RawMessage("[]")
	}
	if app.Scopes == nil {
		app.Scopes = json.RawMessage("[]")
	}
	if app.ClientSecret == "" {
		app.ClientSecret = generateToken(32)
	}
	if app.SigningSecret == "" {
		app.SigningSecret = generateToken(32)
	}
	err := db.QueryRow(`INSERT INTO apps (id, owner_id, name, slug, description, icon, icon_url, homepage, tools, events, scopes, setup_url, redirect_url, client_secret, signing_secret, listed, listing_status, listing_reject_reason)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		RETURNING EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT`,
		app.ID, app.OwnerID, app.Name, app.Slug, app.Description, app.Icon, app.IconURL, app.Homepage,
		app.Tools, app.Events, app.Scopes, app.SetupURL, app.RedirectURL, app.ClientSecret, app.SigningSecret, app.Listed, "", "",
	).Scan(&app.CreatedAt, &app.UpdatedAt)
	app.Status = "active"
	return app, err
}

// GetApp returns an app by ID.
func (db *DB) GetApp(id string) (*App, error) {
	a := &App{}
	err := db.QueryRow(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, a.client_secret,
		a.request_url, a.signing_secret, a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
		EXTRACT(EPOCH FROM a.created_at)::BIGINT, EXTRACT(EPOCH FROM a.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.id = $1`, id).Scan(
		&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
		&a.Tools, &a.Events, &a.Scopes, &a.SetupURL, &a.RedirectURL, &a.ClientSecret,
		&a.RequestURL, &a.SigningSecret, &a.URLVerified,
		&a.Listed, &a.ListingStatus, &a.ListingRejectReason, &a.Status,
		&a.CreatedAt, &a.UpdatedAt, &a.OwnerName)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// GetAppBySlug returns an app by slug.
func (db *DB) GetAppBySlug(slug string) (*App, error) {
	a := &App{}
	err := db.QueryRow(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, setup_url, redirect_url, client_secret,
		request_url, signing_secret, url_verified,
		listed, listing_status, listing_reject_reason, status,
		EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT
		FROM apps WHERE slug = $1`, slug).Scan(
		&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
		&a.Tools, &a.Events, &a.Scopes, &a.SetupURL, &a.RedirectURL, &a.ClientSecret,
		&a.RequestURL, &a.SigningSecret, &a.URLVerified,
		&a.Listed, &a.ListingStatus, &a.ListingRejectReason, &a.Status,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

// ListAppsByOwner returns all apps owned by a user.
func (db *DB) ListAppsByOwner(ownerID string) ([]App, error) {
	rows, err := db.Query(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, setup_url, redirect_url, client_secret,
		request_url, signing_secret, url_verified,
		listed, listing_status, listing_reject_reason, status,
		EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT
		FROM apps WHERE owner_id = $1 ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []App
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.SetupURL, &a.RedirectURL, &a.ClientSecret,
			&a.RequestURL, &a.SigningSecret, &a.URLVerified,
			&a.Listed, &a.ListingStatus, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// ListListedApps returns all publicly listed apps.
func (db *DB) ListListedApps() ([]App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, '',
		a.request_url, '', a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
		EXTRACT(EPOCH FROM a.created_at)::BIGINT, EXTRACT(EPOCH FROM a.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.listed = TRUE AND a.status = 'active' ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []App
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.SetupURL, &a.RedirectURL, &a.ClientSecret,
			&a.RequestURL, &a.SigningSecret, &a.URLVerified,
			&a.Listed, &a.ListingStatus, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt, &a.OwnerName); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// ListAllApps returns all apps (admin only).
func (db *DB) ListAllApps() ([]App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, '',
		a.request_url, '', a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
		EXTRACT(EPOCH FROM a.created_at)::BIGINT, EXTRACT(EPOCH FROM a.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		ORDER BY a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []App
	for rows.Next() {
		var a App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.SetupURL, &a.RedirectURL, &a.ClientSecret,
			&a.RequestURL, &a.SigningSecret, &a.URLVerified,
			&a.Listed, &a.ListingStatus, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt, &a.OwnerName); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// SetAppListed sets the listed flag (admin only).
func (db *DB) SetAppListed(id string, listed bool) error {
	_, err := db.Exec("UPDATE apps SET listed=$1, updated_at=NOW() WHERE id=$2", listed, id)
	return err
}

// UpdateApp updates an app's fields.
func (db *DB) UpdateApp(id string, name, description, icon, iconURL, homepage, setupURL, redirectURL string, tools, events, scopes json.RawMessage) error {
	_, err := db.Exec(`UPDATE apps SET name=$1, description=$2, icon=$3, icon_url=$4, homepage=$5,
		tools=$6, events=$7, scopes=$8, setup_url=$9, redirect_url=$10, updated_at=NOW() WHERE id=$11`,
		name, description, icon, iconURL, homepage, tools, events, scopes, setupURL, redirectURL, id)
	return err
}

// DeleteApp deletes an app and all its installations (cascaded).
func (db *DB) DeleteApp(id string) error {
	_, err := db.Exec("DELETE FROM apps WHERE id = $1", id)
	return err
}

// InstallApp creates an installation of an app to a bot.
func (db *DB) InstallApp(appID, botID string) (*AppInstallation, error) {
	inst := &AppInstallation{
		ID:       uuid.New().String(),
		AppID:    appID,
		BotID:    botID,
		AppToken: "app_" + generateToken(32),
		Config:   json.RawMessage("{}"),
		Enabled:  true,
	}
	err := db.QueryRow(`INSERT INTO app_installations (id, app_id, bot_id, app_token, config)
		VALUES ($1,$2,$3,$4,$5)
		RETURNING EXTRACT(EPOCH FROM created_at)::BIGINT, EXTRACT(EPOCH FROM updated_at)::BIGINT`,
		inst.ID, inst.AppID, inst.BotID, inst.AppToken, inst.Config,
	).Scan(&inst.CreatedAt, &inst.UpdatedAt)
	return inst, err
}

// GetInstallation returns an installation by ID.
func (db *DB) GetInstallation(id string) (*AppInstallation, error) {
	i := &AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		EXTRACT(EPOCH FROM i.created_at)::BIGINT, EXTRACT(EPOCH FROM i.updated_at)::BIGINT,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.id = $1`, id).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppRequestURL, &i.AppSigningSecret)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// GetInstallationByToken returns an installation by app_token.
func (db *DB) GetInstallationByToken(token string) (*AppInstallation, error) {
	i := &AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		EXTRACT(EPOCH FROM i.created_at)::BIGINT, EXTRACT(EPOCH FROM i.updated_at)::BIGINT,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.app_token = $1`, token).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppRequestURL, &i.AppSigningSecret)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// ListInstallationsByApp returns all installations of an app.
func (db *DB) ListInstallationsByApp(appID string) ([]AppInstallation, error) {
	return db.listInstallations("i.app_id = $1", appID)
}

// ListInstallationsByBot returns all installations on a bot.
func (db *DB) ListInstallationsByBot(botID string) ([]AppInstallation, error) {
	return db.listInstallations("i.bot_id = $1", botID)
}

func (db *DB) listInstallations(where string, arg any) ([]AppInstallation, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		EXTRACT(EPOCH FROM i.created_at)::BIGINT, EXTRACT(EPOCH FROM i.updated_at)::BIGINT,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE %s ORDER BY i.created_at DESC`, where), arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AppInstallation
	for rows.Next() {
		var i AppInstallation
		if err := rows.Scan(&i.ID, &i.AppID, &i.BotID, &i.AppToken,
			&i.Handle, &i.Config, &i.Enabled,
			&i.CreatedAt, &i.UpdatedAt,
			&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
			&i.AppRequestURL, &i.AppSigningSecret); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// UpdateInstallation updates handle, config, enabled.
func (db *DB) UpdateInstallation(id, handle string, config json.RawMessage, enabled bool) error {
	_, err := db.Exec(`UPDATE app_installations SET handle=$1, config=$2, enabled=$3, updated_at=NOW() WHERE id=$4`,
		handle, config, enabled, id)
	return err
}

// SetAppURLVerified marks the app's request URL as verified.
func (db *DB) SetAppURLVerified(id string, verified bool) error {
	_, err := db.Exec("UPDATE apps SET url_verified=$1, updated_at=NOW() WHERE id=$2", verified, id)
	return err
}

// UpdateAppRequestURL updates the app's request URL and resets verification.
func (db *DB) UpdateAppRequestURL(id, requestURL string) error {
	_, err := db.Exec("UPDATE apps SET request_url=$1, url_verified=FALSE, updated_at=NOW() WHERE id=$2", requestURL, id)
	return err
}

// RegenerateInstallationToken generates a new app_token for an installation.
func (db *DB) RegenerateInstallationToken(id string) (string, error) {
	token := "app_" + generateToken(32)
	_, err := db.Exec("UPDATE app_installations SET app_token=$1, updated_at=NOW() WHERE id=$2", token, id)
	return token, err
}

// GetInstallationByHandle returns the installation matching a handle on a bot.
func (db *DB) GetInstallationByHandle(botID, handle string) (*AppInstallation, error) {
	i := &AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		EXTRACT(EPOCH FROM i.created_at)::BIGINT, EXTRACT(EPOCH FROM i.updated_at)::BIGINT,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.bot_id = $1 AND i.handle = $2`, botID, handle).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppRequestURL, &i.AppSigningSecret)
	if err != nil {
		return nil, err
	}
	return i, nil
}

// DeleteInstallation removes an installation.
func (db *DB) DeleteInstallation(id string) error {
	_, err := db.Exec("DELETE FROM app_installations WHERE id = $1", id)
	return err
}

// CreateOAuthCode creates a temporary OAuth code for the install flow.
func (db *DB) CreateOAuthCode(code, appID, botID, state string) error {
	_, err := db.Exec(`INSERT INTO app_oauth_codes (code, app_id, bot_id, state) VALUES ($1,$2,$3,$4)`,
		code, appID, botID, state)
	return err
}

// ExchangeOAuthCode consumes a code and returns the app_id and bot_id. Deletes the code.
func (db *DB) ExchangeOAuthCode(code string) (appID, botID string, err error) {
	err = db.QueryRow(`DELETE FROM app_oauth_codes WHERE code = $1 AND expires_at > NOW() RETURNING app_id, bot_id`,
		code).Scan(&appID, &botID)
	return
}

// CleanExpiredOAuthCodes removes expired codes.
func (db *DB) CleanExpiredOAuthCodes() {
	db.Exec("DELETE FROM app_oauth_codes WHERE expires_at < NOW()")
}

// RequestListing sets listing_status to "pending".
func (db *DB) RequestListing(id string) error {
	_, err := db.Exec("UPDATE apps SET listing_status='pending', updated_at=NOW() WHERE id=$1", id)
	return err
}

// ReviewListing approves or rejects a listing request.
func (db *DB) ReviewListing(id string, approve bool, reason string) error {
	if approve {
		_, err := db.Exec("UPDATE apps SET listed=TRUE, listing_status='', listing_reject_reason='', updated_at=NOW() WHERE id=$1", id)
		return err
	}
	_, err := db.Exec("UPDATE apps SET listing_status='rejected', listing_reject_reason=$1, updated_at=NOW() WHERE id=$2", reason, id)
	return err
}
