package database

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Plugin is the stable identity of a plugin.
type Plugin struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace,omitempty"`
	Description     string `json:"description"`
	Author          string `json:"author"`
	Icon            string `json:"icon,omitempty"`
	License         string `json:"license,omitempty"`
	Homepage        string `json:"homepage,omitempty"`
	OwnerID         string `json:"owner_id"`
	LatestVersionID string `json:"latest_version_id,omitempty"`
	InstallCount    int    `json:"install_count"`
	CreatedAt       int64  `json:"created_at"`
	UpdatedAt       int64  `json:"updated_at"`

	// Joined
	OwnerName string `json:"owner_name,omitempty"`
}

// PluginVersion is a specific release of a plugin.
type PluginVersion struct {
	ID             string          `json:"id"`
	PluginID       string          `json:"plugin_id"`
	Version        string          `json:"version"`
	Changelog      string          `json:"changelog,omitempty"`
	Script         string          `json:"script,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema"`
	GithubURL      string          `json:"github_url,omitempty"`
	CommitHash     string          `json:"commit_hash,omitempty"`
	MatchTypes     string          `json:"match_types"`
	ConnectDomains string          `json:"connect_domains"`
	GrantPerms     string          `json:"grant_perms"`
	Status         string          `json:"status"`
	RejectReason   string          `json:"reject_reason,omitempty"`
	ReviewedBy     string          `json:"reviewed_by,omitempty"`
	CreatedAt      int64           `json:"created_at"`

	// Joined
	ReviewerName string `json:"reviewer_name,omitempty"`
}

// PluginWithLatest is a plugin joined with its latest version info for listing.
type PluginWithLatest struct {
	Plugin
	Version        string          `json:"version,omitempty"`
	Changelog      string          `json:"changelog,omitempty"`
	MatchTypes     string          `json:"match_types,omitempty"`
	ConnectDomains string          `json:"connect_domains,omitempty"`
	GrantPerms     string          `json:"grant_perms,omitempty"`
	ConfigSchema   json.RawMessage `json:"config_schema,omitempty"`
}

type ConfigField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// --- Plugin CRUD ---

func (db *DB) CreatePlugin(p *Plugin) (*Plugin, error) {
	p.ID = uuid.New().String()
	_, err := db.Exec(`INSERT INTO plugins (id, name, namespace, description, author, icon, license, homepage, owner_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.Name, p.Namespace, p.Description, p.Author, p.Icon, p.License, p.Homepage, p.OwnerID)
	return p, err
}

func (db *DB) GetPlugin(id string) (*Plugin, error) {
	p := &Plugin{}
	err := db.QueryRow(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM plugins p LEFT JOIN users u ON u.id = p.owner_id WHERE p.id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Namespace, &p.Description, &p.Author, &p.Icon, &p.License, &p.Homepage,
			&p.OwnerID, &p.LatestVersionID, &p.InstallCount, &p.CreatedAt, &p.UpdatedAt, &p.OwnerName)
	return p, err
}

func (db *DB) GetPluginByName(name string) (*Plugin, error) {
	p := &Plugin{}
	err := db.QueryRow(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM plugins p LEFT JOIN users u ON u.id = p.owner_id WHERE p.name = $1`, name).
		Scan(&p.ID, &p.Name, &p.Namespace, &p.Description, &p.Author, &p.Icon, &p.License, &p.Homepage,
			&p.OwnerID, &p.LatestVersionID, &p.InstallCount, &p.CreatedAt, &p.UpdatedAt, &p.OwnerName)
	return p, err
}

func (db *DB) ListPlugins() ([]PluginWithLatest, error) {
	rows, err := db.Query(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, ''),
		COALESCE(v.version, ''), COALESCE(v.changelog, ''),
		COALESCE(v.match_types, '*'), COALESCE(v.connect_domains, '*'), COALESCE(v.grant_perms, ''),
		COALESCE(v.config_schema, '[]'::jsonb)
		FROM plugins p
		LEFT JOIN users u ON u.id = p.owner_id
		LEFT JOIN plugin_versions v ON v.id = p.latest_version_id
		WHERE p.latest_version_id != ''
		ORDER BY p.install_count DESC, p.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PluginWithLatest
	for rows.Next() {
		var pw PluginWithLatest
		if err := rows.Scan(&pw.ID, &pw.Name, &pw.Namespace, &pw.Description, &pw.Author, &pw.Icon, &pw.License, &pw.Homepage,
			&pw.OwnerID, &pw.LatestVersionID, &pw.InstallCount, &pw.CreatedAt, &pw.UpdatedAt, &pw.OwnerName,
			&pw.Version, &pw.Changelog, &pw.MatchTypes, &pw.ConnectDomains, &pw.GrantPerms, &pw.ConfigSchema); err != nil {
			return nil, err
		}
		result = append(result, pw)
	}
	return result, rows.Err()
}

func (db *DB) ListPluginsByOwner(ownerID string) ([]PluginWithLatest, error) {
	rows, err := db.Query(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, ''),
		COALESCE(v.version, ''), COALESCE(v.changelog, ''),
		COALESCE(v.match_types, '*'), COALESCE(v.connect_domains, '*'), COALESCE(v.grant_perms, ''),
		COALESCE(v.config_schema, '[]'::jsonb)
		FROM plugins p
		LEFT JOIN users u ON u.id = p.owner_id
		LEFT JOIN plugin_versions v ON v.id = p.latest_version_id
		WHERE p.owner_id = $1
		ORDER BY p.created_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []PluginWithLatest
	for rows.Next() {
		var pw PluginWithLatest
		if err := rows.Scan(&pw.ID, &pw.Name, &pw.Namespace, &pw.Description, &pw.Author, &pw.Icon, &pw.License, &pw.Homepage,
			&pw.OwnerID, &pw.LatestVersionID, &pw.InstallCount, &pw.CreatedAt, &pw.UpdatedAt, &pw.OwnerName,
			&pw.Version, &pw.Changelog, &pw.MatchTypes, &pw.ConnectDomains, &pw.GrantPerms, &pw.ConfigSchema); err != nil {
			return nil, err
		}
		result = append(result, pw)
	}
	return result, rows.Err()
}

func (db *DB) UpdatePluginMeta(id string, p *Plugin) error {
	_, err := db.Exec(`UPDATE plugins SET description=$1, author=$2, icon=$3, license=$4, homepage=$5, namespace=$6, updated_at=NOW() WHERE id=$7`,
		p.Description, p.Author, p.Icon, p.License, p.Homepage, p.Namespace, id)
	return err
}

func (db *DB) DeletePlugin(id string) error {
	db.Exec("DELETE FROM plugin_installs WHERE plugin_id = $1", id)
	db.Exec("DELETE FROM plugin_versions WHERE plugin_id = $1", id)
	_, err := db.Exec("DELETE FROM plugins WHERE id = $1", id)
	return err
}

// --- Version CRUD ---

func (db *DB) CreatePluginVersion(v *PluginVersion) (*PluginVersion, error) {
	v.ID = uuid.New().String()
	_, err := db.Exec(`INSERT INTO plugin_versions
		(id, plugin_id, version, changelog, script, config_schema, github_url, commit_hash,
		 match_types, connect_domains, grant_perms, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'pending')`,
		v.ID, v.PluginID, v.Version, v.Changelog, v.Script, v.ConfigSchema,
		v.GithubURL, v.CommitHash, v.MatchTypes, v.ConnectDomains, v.GrantPerms)
	return v, err
}

func (db *DB) GetPluginVersion(id string) (*PluginVersion, error) {
	v := &PluginVersion{}
	err := db.QueryRow(`SELECT v.id, v.plugin_id, v.version, v.changelog, v.script, v.config_schema,
		v.github_url, v.commit_hash, v.match_types, v.connect_domains, v.grant_perms,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.id = $1`, id).
		Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script, &v.ConfigSchema,
			&v.GithubURL, &v.CommitHash, &v.MatchTypes, &v.ConnectDomains, &v.GrantPerms,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName)
	return v, err
}

func (db *DB) ListPluginVersions(pluginID string) ([]PluginVersion, error) {
	rows, err := db.Query(`SELECT v.id, v.plugin_id, v.version, v.changelog, '',
		v.config_schema, v.github_url, v.commit_hash,
		v.match_types, v.connect_domains, v.grant_perms,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.plugin_id = $1 ORDER BY v.created_at DESC`, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []PluginVersion
	for rows.Next() {
		var v PluginVersion
		if err := rows.Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script,
			&v.ConfigSchema, &v.GithubURL, &v.CommitHash,
			&v.MatchTypes, &v.ConnectDomains, &v.GrantPerms,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

// ListPendingVersions returns all pending versions for admin review.
func (db *DB) ListPendingVersions() ([]PluginVersion, error) {
	rows, err := db.Query(`SELECT v.id, v.plugin_id, v.version, v.changelog, '',
		v.config_schema, v.github_url, v.commit_hash,
		v.match_types, v.connect_domains, v.grant_perms,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.status = 'pending' ORDER BY v.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []PluginVersion
	for rows.Next() {
		var v PluginVersion
		if err := rows.Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script,
			&v.ConfigSchema, &v.GithubURL, &v.CommitHash,
			&v.MatchTypes, &v.ConnectDomains, &v.GrantPerms,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *DB) FindPendingVersion(pluginID string) (*PluginVersion, error) {
	return db.getVersionByPluginAndStatus(pluginID, "pending")
}

func (db *DB) getVersionByPluginAndStatus(pluginID, status string) (*PluginVersion, error) {
	v := &PluginVersion{}
	err := db.QueryRow(`SELECT v.id, v.plugin_id, v.version, v.changelog, v.script, v.config_schema,
		v.github_url, v.commit_hash, v.match_types, v.connect_domains, v.grant_perms,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.plugin_id = $1 AND v.status = $2
		ORDER BY v.created_at DESC LIMIT 1`, pluginID, status).
		Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script, &v.ConfigSchema,
			&v.GithubURL, &v.CommitHash, &v.MatchTypes, &v.ConnectDomains, &v.GrantPerms,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName)
	return v, err
}

func (db *DB) UpdatePluginVersion(id string, v *PluginVersion) error {
	_, err := db.Exec(`UPDATE plugin_versions SET version=$1, changelog=$2, script=$3, config_schema=$4,
		github_url=$5, commit_hash=$6, match_types=$7, connect_domains=$8, grant_perms=$9,
		status='pending', reject_reason='', reviewed_by=''
		WHERE id=$10`,
		v.Version, v.Changelog, v.Script, v.ConfigSchema,
		v.GithubURL, v.CommitHash, v.MatchTypes, v.ConnectDomains, v.GrantPerms, id)
	return err
}

func (db *DB) ReviewPluginVersion(id, status, reviewedBy, reason string) error {
	_, err := db.Exec("UPDATE plugin_versions SET status=$1, reviewed_by=$2, reject_reason=$3 WHERE id=$4",
		status, reviewedBy, reason, id)
	if err != nil {
		return err
	}
	// If approved, update plugin's latest_version_id
	if status == "approved" {
		var pluginID string
		db.QueryRow("SELECT plugin_id FROM plugin_versions WHERE id = $1", id).Scan(&pluginID)
		if pluginID != "" {
			db.Exec("UPDATE plugins SET latest_version_id = $1, updated_at = NOW() WHERE id = $2", id, pluginID)
		}
	}
	return nil
}

func (db *DB) DeletePluginVersion(id string) error {
	_, err := db.Exec("DELETE FROM plugin_versions WHERE id = $1", id)
	return err
}

// --- Install tracking ---

func (db *DB) RecordPluginInstall(pluginID, userID string) error {
	_, err := db.Exec(`INSERT INTO plugin_installs (plugin_id, user_id) VALUES ($1, $2)
		ON CONFLICT (plugin_id, user_id) DO UPDATE SET installed_at = NOW()`, pluginID, userID)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE plugins SET install_count = (SELECT COUNT(*) FROM plugin_installs WHERE plugin_id = $1) WHERE id = $1`, pluginID)
	return err
}

// --- Helpers ---

func (db *DB) FindPluginOwner(name string) (string, error) {
	var owner string
	err := db.QueryRow("SELECT owner_id FROM plugins WHERE name = $1", name).Scan(&owner)
	if err != nil {
		return "", err
	}
	return owner, nil
}

// ResolvePluginScript returns the script for a given version ID (used by webhook sink).
func (db *DB) ResolvePluginScript(versionID string) (script, version string, err error) {
	err = db.QueryRow("SELECT script, version FROM plugin_versions WHERE id = $1 AND status = 'approved'", versionID).
		Scan(&script, &version)
	if err != nil {
		return "", "", fmt.Errorf("plugin version not found or not approved: %w", err)
	}
	return script, version, nil
}
