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
	if app.ClientSecret == "" {
		app.ClientSecret = generateToken(32)
	}
	if app.SigningSecret == "" {
		app.SigningSecret = generateToken(32)
	}
	_, err := db.Exec(`INSERT INTO apps (id, owner_id, name, slug, description, icon, icon_url, homepage, tools, events, scopes, setup_url, redirect_url, client_secret, signing_secret, listed, listing_status, listing_reject_reason)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		app.ID, app.OwnerID, app.Name, app.Slug, app.Description, app.Icon, app.IconURL, app.Homepage,
		app.Tools, app.Events, app.Scopes, app.SetupURL, app.RedirectURL, app.ClientSecret, app.SigningSecret, app.Listed, "", "",
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
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, a.client_secret,
		a.request_url, a.signing_secret, a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.id = ?`, id).Scan(
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

func (db *DB) GetAppBySlug(slug string) (*store.App, error) {
	a := &store.App{}
	err := db.QueryRow(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, setup_url, redirect_url, client_secret,
		request_url, signing_secret, url_verified,
		listed, listing_status, listing_reject_reason, status,
		created_at, updated_at
		FROM apps WHERE slug = ?`, slug).Scan(
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

func (db *DB) ListAppsByOwner(ownerID string) ([]store.App, error) {
	rows, err := db.Query(`SELECT id, owner_id, name, slug, description, icon, icon_url, homepage,
		tools, events, scopes, setup_url, redirect_url, client_secret,
		request_url, signing_secret, url_verified,
		listed, listing_status, listing_reject_reason, status,
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

func (db *DB) ListListedApps() ([]store.App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, '',
		a.request_url, '', a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
		a.created_at, a.updated_at,
		COALESCE(u.username, '')
		FROM apps a LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.listed = 1 AND a.status = 'active' ORDER BY a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var apps []store.App
	for rows.Next() {
		var a store.App
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

func (db *DB) ListAllApps() ([]store.App, error) {
	rows, err := db.Query(`SELECT a.id, a.owner_id, a.name, a.slug, a.description, a.icon, a.icon_url, a.homepage,
		a.tools, a.events, a.scopes, a.setup_url, a.redirect_url, '',
		a.request_url, '', a.url_verified,
		a.listed, a.listing_status, a.listing_reject_reason, a.status,
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

func (db *DB) SetAppListed(id string, listed bool) error {
	_, err := db.Exec("UPDATE apps SET listed=?, updated_at=unixepoch() WHERE id=?", listed, id)
	return err
}

func (db *DB) UpdateApp(id string, name, description, icon, iconURL, homepage, setupURL, redirectURL string, tools, events, scopes json.RawMessage) error {
	_, err := db.Exec(`UPDATE apps SET name=?, description=?, icon=?, icon_url=?, homepage=?,
		tools=?, events=?, scopes=?, setup_url=?, redirect_url=?, updated_at=unixepoch() WHERE id=?`,
		name, description, icon, iconURL, homepage, tools, events, scopes, setupURL, redirectURL, id)
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
		Enabled:  true,
	}
	_, err := db.Exec(`INSERT INTO app_installations (id, app_id, bot_id, app_token, config)
		VALUES (?,?,?,?,?)`,
		inst.ID, inst.AppID, inst.BotID, inst.AppToken, inst.Config,
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
		i.handle, i.config, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.id = ?`, id).Scan(
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

func (db *DB) GetInstallationByToken(token string) (*store.AppInstallation, error) {
	i := &store.AppInstallation{}
	err := db.QueryRow(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.app_token = ?`, token).Scan(
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

func (db *DB) ListInstallationsByApp(appID string) ([]store.AppInstallation, error) {
	return db.listInstallations("i.app_id = ?", appID)
}

func (db *DB) ListInstallationsByBot(botID string) ([]store.AppInstallation, error) {
	return db.listInstallations("i.bot_id = ?", botID)
}

func (db *DB) listInstallations(where string, arg any) ([]store.AppInstallation, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT i.id, i.app_id, i.bot_id, i.app_token,
		i.handle, i.config, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
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

func (db *DB) UpdateInstallation(id, handle string, config json.RawMessage, enabled bool) error {
	_, err := db.Exec(`UPDATE app_installations SET handle=?, config=?, enabled=?, updated_at=unixepoch() WHERE id=?`,
		handle, config, enabled, id)
	return err
}

func (db *DB) SetAppURLVerified(id string, verified bool) error {
	_, err := db.Exec("UPDATE apps SET url_verified=?, updated_at=unixepoch() WHERE id=?", verified, id)
	return err
}

func (db *DB) UpdateAppRequestURL(id, requestURL string) error {
	_, err := db.Exec("UPDATE apps SET request_url=?, url_verified=0, updated_at=unixepoch() WHERE id=?", requestURL, id)
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
		i.handle, i.config, i.enabled,
		i.created_at, i.updated_at,
		COALESCE(a.name,''), COALESCE(a.slug,''), COALESCE(a.icon,''), COALESCE(a.icon_url,''),
		a.request_url, a.signing_secret
		FROM app_installations i JOIN apps a ON a.id = i.app_id
		WHERE i.bot_id = ? AND i.handle = ?`, botID, handle).Scan(
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

func (db *DB) DeleteInstallation(id string) error {
	_, err := db.Exec("DELETE FROM app_installations WHERE id = ?", id)
	return err
}

func (db *DB) CreateOAuthCode(code, appID, botID, state string) error {
	_, err := db.Exec(`INSERT INTO app_oauth_codes (code, app_id, bot_id, state) VALUES (?,?,?,?)`,
		code, appID, botID, state)
	return err
}

func (db *DB) ExchangeOAuthCode(code string) (appID, botID string, err error) {
	tx, err := db.Begin()
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback()

	err = tx.QueryRow("SELECT app_id, bot_id FROM app_oauth_codes WHERE code = ? AND expires_at > unixepoch()", code).
		Scan(&appID, &botID)
	if err != nil {
		return "", "", err
	}
	tx.Exec("DELETE FROM app_oauth_codes WHERE code = ?", code)
	if err := tx.Commit(); err != nil {
		return "", "", err
	}
	return appID, botID, nil
}

func (db *DB) CleanExpiredOAuthCodes() {
	db.Exec("DELETE FROM app_oauth_codes WHERE expires_at < unixepoch()")
}

func (db *DB) RequestListing(id string) error {
	_, err := db.Exec("UPDATE apps SET listing_status='pending', updated_at=unixepoch() WHERE id=?", id)
	return err
}

func (db *DB) ReviewListing(id string, approve bool, reason string) error {
	if approve {
		_, err := db.Exec("UPDATE apps SET listed=1, listing_status='', listing_reject_reason='', updated_at=unixepoch() WHERE id=?", id)
		return err
	}
	_, err := db.Exec("UPDATE apps SET listing_status='rejected', listing_reject_reason=?, updated_at=unixepoch() WHERE id=?", reason, id)
	return err
}
