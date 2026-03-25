-- Plugin marketplace
CREATE TABLE IF NOT EXISTS plugins (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    author        TEXT NOT NULL DEFAULT '',
    version       TEXT NOT NULL DEFAULT '1.0.0',
    github_url    TEXT NOT NULL DEFAULT '',
    commit_hash   TEXT NOT NULL DEFAULT '',
    script        TEXT NOT NULL,
    config_schema JSONB NOT NULL DEFAULT '[]',
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending/approved/rejected
    reject_reason TEXT NOT NULL DEFAULT '',
    submitted_by  TEXT NOT NULL,
    reviewed_by   TEXT NOT NULL DEFAULT '',
    install_count INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_plugins_status ON plugins(status);

-- Track who installed which plugin
CREATE TABLE IF NOT EXISTS plugin_installs (
    plugin_id    TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin_id, user_id)
);
