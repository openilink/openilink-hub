package sqlite

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func generateToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (db *DB) CreateApp(app *store.App) (*store.App, error) {
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
	if app.WebhookSecret == "" {
		app.WebhookSecret = generateToken(32)
	}
	if app.Listing == "" {
		app.Listing = "unlisted"
	}
	if app.ConfigSchema == "" {
		app.ConfigSchema = "{}"
	}
	_, err := db.Exec(`INSERT INTO apps (id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, oauth_setup_url, oauth_redirect_url,
		webhook_url, webhook_secret, webhook_verified,
		registry, version, readme, guide, config_schema,
		listing, listing_reject_reason)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		app.ID, app.OwnerID, app.Name, app.Slug, app.Description, app.Icon, app.IconURL, app.Homepage,
		app.Tools, app.Events, app.Scopes, app.OAuthSetupURL, app.OAuthRedirectURL,
		app.WebhookURL, app.WebhookSecret, app.WebhookVerified,
		app.Registry, app.Version, app.Readme, app.Guide, app.ConfigSchema,
		app.Listing, app.ListingRejectReason,
	)
	if err != nil {
		return nil, err
	}
	err = db.QueryRow("SELECT created_at, updated_at FROM apps WHERE id = ?", app.ID).Scan(&app.CreatedAt, &app.UpdatedAt)
	app.Status = "active"
	return app, err
}

func (db *DB) GetApp(id string) (*store.App, error) {
	a := &store.App{}
	err := db.QueryRow(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.oauth_setup_url, a.oauth_redirect_url,
		a.webhook_url, a.webhook_secret, a.webhook_verified,
		a.registry, a.version, a.readme, a.guide, a.config_schema,
		a.listing, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.id = ?`, id).Scan(
		&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
		&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
		&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
		&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
		&a.Listing, &a.ListingRejectReason, &a.Status,
		&a.CreatedAt, &a.UpdatedAt, &a.OwnerName)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (db *DB) GetAppBySlug(slug, registry string) (*store.App, error) {
	a := &store.App{}
	err := db.QueryRow(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, oauth_setup_url, oauth_redirect_url,
		webhook_url, webhook_secret, webhook_verified,
		registry, version, readme, guide, config_schema,
		listing, listing_reject_reason, status,
		created_at, updated_at
		FROM apps WHERE slug = ? AND registry = ?`, slug, registry).Scan(
		&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
		&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
		&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
		&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
		&a.Listing, &a.ListingRejectReason, &a.Status,
		&a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (db *DB) ListAppsByOwner(ownerID string) ([]store.App, error) {
	rows, err := db.Query(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, oauth_setup_url, oauth_redirect_url,
		webhook_url, webhook_secret, webhook_verified,
		registry, version, readme, guide, config_schema,
		listing, listing_reject_reason, status,
		created_at, updated_at
		FROM apps WHERE owner_id = ? ORDER BY created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []store.App
	for rows.Next() {
		var a store.App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
			&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
			&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
			&a.Listing, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (db *DB) ListListedApps() ([]store.App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.oauth_setup_url, a.oauth_redirect_url,
		a.webhook_url, '', a.webhook_verified,
		a.registry, a.version, a.readme, a.guide, a.config_schema,
		a.listing, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.listing = 'listed' AND a.status = 'active' ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []store.App
	for rows.Next() {
		var a store.App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
			&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
			&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
			&a.Listing, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt, &a.OwnerName); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (db *DB) ListAllApps() ([]store.App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.oauth_setup_url, a.oauth_redirect_url,
		a.webhook_url, '', a.webhook_verified,
		a.registry, a.version, a.readme, a.guide, a.config_schema,
		a.listing, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		ORDER BY a.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []store.App
	for rows.Next() {
		var a store.App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
			&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
			&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
			&a.Listing, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt, &a.OwnerName); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (db *DB) ListMarketplaceApps() ([]store.App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.oauth_setup_url, a.oauth_redirect_url,
		a.webhook_url, '', a.webhook_verified,
		a.registry, a.version, a.readme, a.guide, a.config_schema,
		a.listing, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.registry != '' AND a.status = 'active' ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []store.App
	for rows.Next() {
		var a store.App
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Name, &a.Slug, &a.Description, &a.Icon, &a.IconURL, &a.Homepage,
			&a.Tools, &a.Events, &a.Scopes, &a.OAuthSetupURL, &a.OAuthRedirectURL,
			&a.WebhookURL, &a.WebhookSecret, &a.WebhookVerified,
			&a.Registry, &a.Version, &a.Readme, &a.Guide, &a.ConfigSchema,
			&a.Listing, &a.ListingRejectReason, &a.Status,
			&a.CreatedAt, &a.UpdatedAt, &a.OwnerName); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (db *DB) UpdateApp(id string, name, description, icon, iconURL, homepage, oauthSetupURL, oauthRedirectURL, configSchema, version, readme, guide string, tools, events, scopes json.RawMessage) error {
	_, err := db.Exec(`UPDATE apps SET name=?, description=?, icon=?, icon_url=?, homepage=?,
		tools=?, events=?, scopes=?, oauth_setup_url=?, oauth_redirect_url=?, config_schema=?,
		version=?, readme=?, guide=?, updated_at=unixepoch() WHERE id=?`,
		name, description, icon, iconURL, homepage, tools, events, scopes, oauthSetupURL, oauthRedirectURL, configSchema, version, readme, guide, id)
	return err
}

func (db *DB) UpdateMarketplaceApp(id, name, description, iconURL, homepage, webhookURL, oauthSetupURL, oauthRedirectURL, version, readme, guide string, tools, events, scopes json.RawMessage) error {
	_, err := db.Exec(`UPDATE apps SET name=?, description=?, icon_url=?, homepage=?,
		webhook_url=?, oauth_setup_url=?, oauth_redirect_url=?, version=?, readme=?, guide=?,
		tools=?, events=?, scopes=?, updated_at=unixepoch() WHERE id=?`,
		name, description, iconURL, homepage, webhookURL, oauthSetupURL, oauthRedirectURL, version, readme, guide, tools, events, scopes, id)
	return err
}

func (db *DB) DeleteApp(id string) error {
	_, err := db.Exec("DELETE FROM apps WHERE id = ?", id)
	return err
}

func (db *DB) InstallApp(appID, botID string) (*store.AppInstallation, error) {
	inst := &store.AppInstallation{
		ID:       uuid.New().String(),
		AppID:    appID,
		BotID:    botID,
		AppToken: "app_" + generateToken(32),
		Config:   json.RawMessage("{}"),
		Scopes:   json.RawMessage("[]"),
		Enabled:  true,
	}
	inst.Tools = json.RawMessage("[]")
	_, err := db.Exec(`INSERT INTO app_installations (id, app_id, bot_id, app_token, config, scopes, tools)
		VALUES (?,?,?,?,?,?,?)`,
		inst.ID, inst.AppID, inst.BotID, inst.AppToken, inst.Config, inst.Scopes, inst.Tools,
	)
	if err != nil {
		return nil, err
	}
	db.QueryRow("SELECT created_at, updated_at FROM app_installations WHERE id = ?", inst.ID).
		Scan(&inst.CreatedAt, &inst.UpdatedAt)
	return inst, nil
}

func (db *DB) GetInstallation(id string) (*store.AppInstallation, error) {
	i := &store.AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.scopes, i.tools, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.webhook_url, a.webhook_secret,
		a.registry, a.readme, a.guide
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.id = ?`, id).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Scopes, &i.Tools, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppWebhookURL, &i.AppWebhookSecret,
		&i.AppRegistry, &i.AppReadme, &i.AppGuide)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (db *DB) GetInstallationByToken(token string) (*store.AppInstallation, error) {
	i := &store.AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.scopes, i.tools, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.webhook_url, a.webhook_secret,
		a.registry, a.readme, a.guide
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.app_token = ?`, token).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Scopes, &i.Tools, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppWebhookURL, &i.AppWebhookSecret,
		&i.AppRegistry, &i.AppReadme, &i.AppGuide)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (db *DB) ListInstallationsByApp(appID string) ([]store.AppInstallation, error) {
	return db.listInstallations("i.app_id = ?", appID)
}

func (db *DB) ListInstallationsByBot(botID string) ([]store.AppInstallation, error) {
	return db.listInstallations("i.bot_id = ?", botID)
}

func (db *DB) listInstallations(where string, arg any) ([]store.AppInstallation, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.scopes, i.tools, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.webhook_url, a.webhook_secret,
		a.registry, a.readme, a.guide
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE %s ORDER BY i.created_at DESC`, where), arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []store.AppInstallation
	for rows.Next() {
		var i store.AppInstallation
		if err := rows.Scan(&i.ID, &i.AppID, &i.BotID, &i.AppToken,
			&i.Handle, &i.Config, &i.Scopes, &i.Tools, &i.Enabled,
			&i.CreatedAt, &i.UpdatedAt,
			&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
			&i.AppWebhookURL, &i.AppWebhookSecret,
			&i.AppRegistry, &i.AppReadme, &i.AppGuide); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

func (db *DB) UpdateInstallation(id, handle string, config json.RawMessage, scopes json.RawMessage, enabled bool) error {
	_, err := db.Exec(`UPDATE app_installations SET handle=?, config=?, scopes=?, enabled=?, updated_at=unixepoch() WHERE id=?`,
		handle, config, scopes, enabled, id)
	return err
}

func (db *DB) SetAppWebhookVerified(id string, verified bool) error {
	_, err := db.Exec("UPDATE apps SET webhook_verified=?, updated_at=unixepoch() WHERE id=?", verified, id)
	return err
}

func (db *DB) UpdateAppWebhookURL(id, webhookURL string) error {
	_, err := db.Exec("UPDATE apps SET webhook_url=?, webhook_verified=0, updated_at=unixepoch() WHERE id=?", webhookURL, id)
	return err
}

func (db *DB) RegenerateInstallationToken(id string) (string, error) {
	token := "app_" + generateToken(32)
	_, err := db.Exec("UPDATE app_installations SET app_token=?, updated_at=unixepoch() WHERE id=?", token, id)
	return token, err
}

func (db *DB) GetInstallationByHandle(botID, handle string) (*store.AppInstallation, error) {
	i := &store.AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.scopes, i.tools, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.webhook_url, a.webhook_secret,
		a.registry, a.readme, a.guide
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.bot_id = ? AND i.handle = ?`, botID, handle).Scan(
		&i.ID, &i.AppID, &i.BotID, &i.AppToken,
		&i.Handle, &i.Config, &i.Scopes, &i.Tools, &i.Enabled,
		&i.CreatedAt, &i.UpdatedAt,
		&i.AppName, &i.AppSlug, &i.AppIcon, &i.AppIconURL,
		&i.AppWebhookURL, &i.AppWebhookSecret,
		&i.AppRegistry, &i.AppReadme, &i.AppGuide)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (db *DB) DeleteInstallation(id string) error {
	_, err := db.Exec("DELETE FROM app_installations WHERE id = ?", id)
	return err
}

func (db *DB) CreateOAuthCode(code, appID, botID, state, codeChallenge string) error {
	_, err := db.Exec(`INSERT INTO app_oauth_codes (code, app_id, bot_id, state, code_challenge) VALUES (?,?,?,?,?)`,
		code, appID, botID, state, codeChallenge)
	return err
}

func (db *DB) ExchangeOAuthCode(code string) (appID, botID, codeChallenge string, err error) {
	tx, err := db.Begin()
	if err != nil {
		return "", "", "", err
	}
	defer tx.Rollback()

	err = tx.QueryRow("SELECT app_id, bot_id, code_challenge FROM app_oauth_codes WHERE code = ? AND expires_at > unixepoch()", code).
		Scan(&appID, &botID, &codeChallenge)
	if err != nil {
		return "", "", "", err
	}
	tx.Exec("DELETE FROM app_oauth_codes WHERE code = ?", code)
	if err := tx.Commit(); err != nil {
		return "", "", "", err
	}
	return appID, botID, codeChallenge, nil
}

func (db *DB) CleanExpiredOAuthCodes() {
	db.Exec("DELETE FROM app_oauth_codes WHERE expires_at < unixepoch()")
}

func (db *DB) RequestListing(id string) error {
	_, err := db.Exec("UPDATE apps SET listing='pending', updated_at=unixepoch() WHERE id=?", id)
	return err
}

func (db *DB) ReviewListing(id string, approve bool, reason string) error {
	if approve {
		_, err := db.Exec("UPDATE apps SET listing='listed', listing_reject_reason='', updated_at=unixepoch() WHERE id=?", id)
		return err
	}
	_, err := db.Exec("UPDATE apps SET listing='rejected', listing_reject_reason=?, updated_at=unixepoch() WHERE id=?", reason, id)
	return err
}

func (db *DB) WithdrawListing(id string) error {
	_, err := db.Exec("UPDATE apps SET listing='unlisted', updated_at=unixepoch() WHERE id=?", id)
	return err
}

func (db *DB) SetListing(id, listing string) error {
	_, err := db.Exec("UPDATE apps SET listing=?, updated_at=unixepoch() WHERE id=?", listing, id)
	return err
}

func (db *DB) UpdateAppTools(id string, tools json.RawMessage) error {
	_, err := db.Exec(`UPDATE apps SET tools = ?, updated_at = unixepoch() WHERE id = ?`, tools, id)
	return err
}

func (db *DB) UpdateInstallationTools(id string, tools json.RawMessage) error {
	_, err := db.Exec(`UPDATE app_installations SET tools = ?, updated_at = unixepoch() WHERE id = ?`, tools, id)
	return err
}

func (db *DB) CreateAppReview(review *store.AppReview) error {
	if review.ID == "" {
		review.ID = uuid.New().String()
	}
	_, err := db.Exec(`INSERT INTO app_reviews (id, app_id, action, actor_id, reason, version, snapshot, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, unixepoch())`,
		review.ID, review.AppID, review.Action, review.ActorID, review.Reason, review.Version, review.Snapshot)
	return err
}

func (db *DB) ListAppReviews(appID string) ([]store.AppReview, error) {
	rows, err := db.Query(`SELECT id, app_id, action, actor_id, reason, version, snapshot, created_at
		FROM app_reviews WHERE app_id = ? ORDER BY created_at DESC`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reviews []store.AppReview
	for rows.Next() {
		var r store.AppReview
		if err := rows.Scan(&r.ID, &r.AppID, &r.Action, &r.ActorID, &r.Reason, &r.Version, &r.Snapshot, &r.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}
