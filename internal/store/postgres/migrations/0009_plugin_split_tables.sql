-- Split plugins into two tables: plugins (identity) + plugin_versions (releases)
-- NOTE: drops old plugins table, existing plugin data will be lost

DROP TABLE IF EXISTS plugin_installs;
DROP TABLE IF EXISTS plugins;

CREATE TABLE plugins (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    namespace       TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    author          TEXT NOT NULL DEFAULT '',
    icon            TEXT NOT NULL DEFAULT '',
    license         TEXT NOT NULL DEFAULT '',
    homepage        TEXT NOT NULL DEFAULT '',
    owner_id        TEXT NOT NULL,
    latest_version_id TEXT NOT NULL DEFAULT '',
    install_count   INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE plugin_versions (
    id              TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL,
    version         TEXT NOT NULL DEFAULT '1.0.0',
    changelog       TEXT NOT NULL DEFAULT '',
    script          TEXT NOT NULL,
    config_schema   JSONB NOT NULL DEFAULT '[]',
    github_url      TEXT NOT NULL DEFAULT '',
    commit_hash     TEXT NOT NULL DEFAULT '',
    match_types     TEXT NOT NULL DEFAULT '*',
    connect_domains TEXT NOT NULL DEFAULT '*',
    grant_perms     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    reject_reason   TEXT NOT NULL DEFAULT '',
    reviewed_by     TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_plugin_versions_plugin ON plugin_versions(plugin_id);
CREATE INDEX idx_plugin_versions_status ON plugin_versions(status);

CREATE TABLE plugin_installs (
    plugin_id    TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin_id, user_id)
);
