package postgres

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/openilink/openilink-hub/internal/store"
)

func (db *DB) CreatePlugin(p *store.Plugin) (*store.Plugin, error) {
	p.ID = uuid.New().String()
	_, err := db.Exec(`INSERT INTO plugins (id, name, namespace, description, author, icon, license, homepage, owner_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.Name, p.Namespace, p.Description, p.Author, p.Icon, p.License, p.Homepage, p.OwnerID)
	return p, err
}

func (db *DB) GetPlugin(id string) (*store.Plugin, error) {
	p := &store.Plugin{}
	err := db.QueryRow(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM plugins p LEFT JOIN users u ON u.id = p.owner_id WHERE p.id = $1`, id).
		Scan(&p.ID, &p.Name, &p.Namespace, &p.Description, &p.Author, &p.Icon, &p.License, &p.Homepage,
			&p.OwnerID, &p.LatestVersionID, &p.InstallCount, &p.CreatedAt, &p.UpdatedAt, &p.OwnerName)
	return p, err
}

func (db *DB) GetPluginByName(name string) (*store.Plugin, error) {
	p := &store.Plugin{}
	err := db.QueryRow(`SELECT p.id, p.name, p.namespace, p.description, p.author, p.icon, p.license, p.homepage,
		p.owner_id, p.latest_version_id, p.install_count,
		EXTRACT(EPOCH FROM p.created_at)::BIGINT, EXTRACT(EPOCH FROM p.updated_at)::BIGINT,
		COALESCE(u.username, '')
		FROM plugins p LEFT JOIN users u ON u.id = p.owner_id WHERE p.name = $1`, name).
		Scan(&p.ID, &p.Name, &p.Namespace, &p.Description, &p.Author, &p.Icon, &p.License, &p.Homepage,
			&p.OwnerID, &p.LatestVersionID, &p.InstallCount, &p.CreatedAt, &p.UpdatedAt, &p.OwnerName)
	return p, err
}

func (db *DB) ListPlugins() ([]store.PluginWithLatest, error) {
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
	var result []store.PluginWithLatest
	for rows.Next() {
		var pw store.PluginWithLatest
		if err := rows.Scan(&pw.ID, &pw.Name, &pw.Namespace, &pw.Description, &pw.Author, &pw.Icon, &pw.License, &pw.Homepage,
			&pw.OwnerID, &pw.LatestVersionID, &pw.InstallCount, &pw.CreatedAt, &pw.UpdatedAt, &pw.OwnerName,
			&pw.Version, &pw.Changelog, &pw.MatchTypes, &pw.ConnectDomains, &pw.GrantPerms, &pw.ConfigSchema); err != nil {
			return nil, err
		}
		result = append(result, pw)
	}
	return result, rows.Err()
}

func (db *DB) ListPluginsByOwner(ownerID string) ([]store.PluginWithLatest, error) {
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
	var result []store.PluginWithLatest
	for rows.Next() {
		var pw store.PluginWithLatest
		if err := rows.Scan(&pw.ID, &pw.Name, &pw.Namespace, &pw.Description, &pw.Author, &pw.Icon, &pw.License, &pw.Homepage,
			&pw.OwnerID, &pw.LatestVersionID, &pw.InstallCount, &pw.CreatedAt, &pw.UpdatedAt, &pw.OwnerName,
			&pw.Version, &pw.Changelog, &pw.MatchTypes, &pw.ConnectDomains, &pw.GrantPerms, &pw.ConfigSchema); err != nil {
			return nil, err
		}
		result = append(result, pw)
	}
	return result, rows.Err()
}

func (db *DB) UpdatePluginMeta(id string, p *store.Plugin) error {
	_, err := db.Exec(`UPDATE plugins SET description=$1, author=$2, icon=$3, license=$4, homepage=$5, namespace=$6, updated_at=$7 WHERE id=$8`,
		p.Description, p.Author, p.Icon, p.License, p.Homepage, p.Namespace, db.now(), id)
	return err
}

func (db *DB) DeletePlugin(id string) error {
	db.Exec("DELETE FROM plugin_installs WHERE plugin_id = $1", id)
	db.Exec("DELETE FROM plugin_versions WHERE plugin_id = $1", id)
	_, err := db.Exec("DELETE FROM plugins WHERE id = $1", id)
	return err
}

func (db *DB) CreatePluginVersion(v *store.PluginVersion) (*store.PluginVersion, error) {
	v.ID = uuid.New().String()
	if v.TimeoutSec <= 0 {
		v.TimeoutSec = 5
	}
	_, err := db.Exec(`INSERT INTO plugin_versions
		(id, plugin_id, version, changelog, script, config_schema, github_url, commit_hash,
		 match_types, connect_domains, grant_perms, timeout_sec, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'pending')`,
		v.ID, v.PluginID, v.Version, v.Changelog, v.Script, v.ConfigSchema,
		v.GithubURL, v.CommitHash, v.MatchTypes, v.ConnectDomains, v.GrantPerms, v.TimeoutSec)
	v.Status = "pending"
	return v, err
}

func (db *DB) GetPluginVersion(id string) (*store.PluginVersion, error) {
	v := &store.PluginVersion{}
	err := db.QueryRow(`SELECT v.id, v.plugin_id, v.version, v.changelog, v.script, v.config_schema,
		v.github_url, v.commit_hash, v.match_types, v.connect_domains, v.grant_perms, v.timeout_sec,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.id = $1`, id).
		Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script, &v.ConfigSchema,
			&v.GithubURL, &v.CommitHash, &v.MatchTypes, &v.ConnectDomains, &v.GrantPerms, &v.TimeoutSec,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName)
	return v, err
}

func (db *DB) ListPluginVersions(pluginID string) ([]store.PluginVersion, error) {
	rows, err := db.Query(`SELECT v.id, v.plugin_id, v.version, v.changelog, '',
		v.config_schema, v.github_url, v.commit_hash,
		v.match_types, v.connect_domains, v.grant_perms, v.timeout_sec,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.plugin_id = $1 ORDER BY v.created_at DESC`, pluginID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []store.PluginVersion
	for rows.Next() {
		var v store.PluginVersion
		if err := rows.Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script,
			&v.ConfigSchema, &v.GithubURL, &v.CommitHash,
			&v.MatchTypes, &v.ConnectDomains, &v.GrantPerms, &v.TimeoutSec,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *DB) ListPendingVersions() ([]store.PluginVersion, error) {
	rows, err := db.Query(`SELECT v.id, v.plugin_id, v.version, v.changelog, v.script,
		v.config_schema, v.github_url, v.commit_hash,
		v.match_types, v.connect_domains, v.grant_perms, v.timeout_sec,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(ru.username, ''),
		p.name, COALESCE(p.icon, ''), COALESCE(p.description, ''), COALESCE(p.author, ''),
		COALESCE(ou.username, '')
		FROM plugin_versions v
		LEFT JOIN users ru ON ru.id = v.reviewed_by
		JOIN plugins p ON p.id = v.plugin_id
		LEFT JOIN users ou ON ou.id = p.owner_id
		WHERE v.status = 'pending' ORDER BY v.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []store.PluginVersion
	for rows.Next() {
		var v store.PluginVersion
		if err := rows.Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script,
			&v.ConfigSchema, &v.GithubURL, &v.CommitHash,
			&v.MatchTypes, &v.ConnectDomains, &v.GrantPerms, &v.TimeoutSec,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName,
			&v.PluginName, &v.PluginIcon, &v.PluginDesc, &v.PluginAuthor, &v.SubmitterName); err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}
	return versions, rows.Err()
}

func (db *DB) SupersedeNonApprovedVersions(pluginID string) {
	db.Exec("UPDATE plugin_versions SET status = 'superseded' WHERE plugin_id = $1 AND status IN ('pending', 'rejected')", pluginID)
}

func (db *DB) CancelPluginVersion(id string) error {
	_, err := db.Exec("UPDATE plugin_versions SET status = 'cancelled' WHERE id = $1", id)
	return err
}

func (db *DB) FindPendingVersion(pluginID string) (*store.PluginVersion, error) {
	return db.getVersionByPluginAndStatus(pluginID, "pending")
}

func (db *DB) getVersionByPluginAndStatus(pluginID, status string) (*store.PluginVersion, error) {
	v := &store.PluginVersion{}
	err := db.QueryRow(`SELECT v.id, v.plugin_id, v.version, v.changelog, v.script, v.config_schema,
		v.github_url, v.commit_hash, v.match_types, v.connect_domains, v.grant_perms, v.timeout_sec,
		v.status, v.reject_reason, v.reviewed_by,
		EXTRACT(EPOCH FROM v.created_at)::BIGINT, COALESCE(u.username, '')
		FROM plugin_versions v LEFT JOIN users u ON u.id = v.reviewed_by
		WHERE v.plugin_id = $1 AND v.status = $2
		ORDER BY v.created_at DESC LIMIT 1`, pluginID, status).
		Scan(&v.ID, &v.PluginID, &v.Version, &v.Changelog, &v.Script, &v.ConfigSchema,
			&v.GithubURL, &v.CommitHash, &v.MatchTypes, &v.ConnectDomains, &v.GrantPerms, &v.TimeoutSec,
			&v.Status, &v.RejectReason, &v.ReviewedBy, &v.CreatedAt, &v.ReviewerName)
	return v, err
}

func (db *DB) UpdatePluginVersion(id string, v *store.PluginVersion) error {
	if v.TimeoutSec <= 0 {
		v.TimeoutSec = 5
	}
	_, err := db.Exec(`UPDATE plugin_versions SET version=$1, changelog=$2, script=$3, config_schema=$4,
		github_url=$5, commit_hash=$6, match_types=$7, connect_domains=$8, grant_perms=$9, timeout_sec=$10,
		status='pending', reject_reason='', reviewed_by=''
		WHERE id=$11`,
		v.Version, v.Changelog, v.Script, v.ConfigSchema,
		v.GithubURL, v.CommitHash, v.MatchTypes, v.ConnectDomains, v.GrantPerms, v.TimeoutSec, id)
	return err
}

func (db *DB) ReviewPluginVersion(id, status, reviewedBy, reason string) error {
	_, err := db.Exec("UPDATE plugin_versions SET status=$1, reviewed_by=$2, reject_reason=$3 WHERE id=$4",
		status, reviewedBy, reason, id)
	if err != nil {
		return err
	}
	if status == "approved" {
		var pluginID string
		db.QueryRow("SELECT plugin_id FROM plugin_versions WHERE id = $1", id).Scan(&pluginID)
		if pluginID != "" {
			db.Exec("UPDATE plugins SET latest_version_id = $1, updated_at = $2 WHERE id = $3", id, db.now(), pluginID)
		}
	}
	return nil
}

func (db *DB) DeletePluginVersion(id string) error {
	_, err := db.Exec("DELETE FROM plugin_versions WHERE id = $1", id)
	return err
}

func (db *DB) RecordPluginInstall(pluginID, userID string) error {
	_, err := db.Exec(`INSERT INTO plugin_installs (plugin_id, user_id) VALUES ($1, $2)
		ON CONFLICT (plugin_id, user_id) DO UPDATE SET installed_at = $3`, pluginID, userID, db.now())
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE plugins SET install_count = (SELECT COUNT(*) FROM plugin_installs WHERE plugin_id = $1) WHERE id = $1`, pluginID)
	return err
}

func (db *DB) FindPluginOwner(name string) (string, error) {
	var owner string
	err := db.QueryRow("SELECT owner_id FROM plugins WHERE name = $1", name).Scan(&owner)
	if err != nil {
		return "", err
	}
	return owner, nil
}

func (db *DB) ResolvePluginScript(versionID string) (script, version string, timeoutSec int, err error) {
	err = db.QueryRow("SELECT script, version, timeout_sec FROM plugin_versions WHERE id = $1 AND status = 'approved'", versionID).
		Scan(&script, &version, &timeoutSec)
	if err != nil {
		return "", "", 0, fmt.Errorf("plugin version not found or not approved: %w", err)
	}
	if timeoutSec <= 0 {
		timeoutSec = 5
	}
	return script, version, timeoutSec, nil
}
