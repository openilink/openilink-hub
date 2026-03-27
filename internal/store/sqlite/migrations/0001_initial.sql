-- +goose Up

-- OpeniLink Hub SQLite schema (consolidated from 24 PG migrations)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL DEFAULT '',
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'member',
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at    INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email != '';

CREATE TABLE IF NOT EXISTS credentials (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    public_key       BLOB NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    transport        TEXT NOT NULL DEFAULT '[]',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    backup_eligible  INTEGER NOT NULL DEFAULT 0,
    backup_state     INTEGER NOT NULL DEFAULT 0,
    created_at       INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_credentials_user ON credentials(user_id);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS oauth_accounts (
    provider    TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    username    TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (provider, provider_id)
);
CREATE INDEX IF NOT EXISTS idx_oauth_user ON oauth_accounts(user_id);

CREATE TABLE IF NOT EXISTS bots (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    name            TEXT NOT NULL DEFAULT '',
    provider        TEXT NOT NULL DEFAULT 'ilink',
    provider_id     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'disconnected',
    credentials     TEXT NOT NULL DEFAULT '{}',
    sync_state      TEXT NOT NULL DEFAULT '{}',
    msg_count       INTEGER NOT NULL DEFAULT 0,
    last_msg_at     INTEGER,
    reminder_hours  INTEGER NOT NULL DEFAULT 0,
    last_reminded_at INTEGER,
    ai_enabled      INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_bots_user ON bots(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_provider_id ON bots(provider, provider_id) WHERE provider_id != '';

CREATE TABLE IF NOT EXISTS channels (
    id             TEXT PRIMARY KEY,
    bot_id         TEXT NOT NULL,
    name           TEXT NOT NULL,
    handle         TEXT NOT NULL DEFAULT '',
    api_key        TEXT NOT NULL UNIQUE,
    filter_rule    TEXT NOT NULL DEFAULT '{}',
    ai_config      TEXT NOT NULL DEFAULT '{}',
    webhook_config TEXT NOT NULL DEFAULT '{}',
    enabled        INTEGER NOT NULL DEFAULT 1,
    last_seq       INTEGER NOT NULL DEFAULT 0,
    created_at     INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at     INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_channels_bot ON channels(bot_id);

CREATE TABLE IF NOT EXISTS messages (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id         TEXT NOT NULL,
    channel_id     TEXT DEFAULT NULL,
    direction      TEXT NOT NULL,
    seq            INTEGER,
    message_id     INTEGER,
    from_user_id   TEXT NOT NULL DEFAULT '',
    to_user_id     TEXT NOT NULL DEFAULT '',
    client_id      TEXT NOT NULL DEFAULT '',
    create_time_ms INTEGER,
    update_time_ms INTEGER,
    delete_time_ms INTEGER,
    session_id     TEXT NOT NULL DEFAULT '',
    group_id       TEXT NOT NULL DEFAULT '',
    message_type   INTEGER NOT NULL DEFAULT 0,
    message_state  INTEGER NOT NULL DEFAULT 0,
    item_list      TEXT NOT NULL DEFAULT '[]',
    context_token  TEXT NOT NULL DEFAULT '',
    media_status   TEXT NOT NULL DEFAULT '',
    media_keys     TEXT NOT NULL DEFAULT '{}',
    raw            TEXT,
    processed_at   INTEGER,
    created_at     INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_messages_bot ON messages(bot_id, id);
CREATE INDEX IF NOT EXISTS idx_messages_seq ON messages(bot_id, seq) WHERE seq IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_from ON messages(bot_id, from_user_id) WHERE from_user_id != '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_bot_msgid ON messages(bot_id, message_id) WHERE message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_unprocessed ON messages(bot_id, id) WHERE direction = 'inbound' AND processed_at IS NULL;

CREATE TABLE IF NOT EXISTS system_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS plugins (
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
    install_count   INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS plugin_versions (
    id              TEXT PRIMARY KEY,
    plugin_id       TEXT NOT NULL,
    version         TEXT NOT NULL DEFAULT '1.0.0',
    changelog       TEXT NOT NULL DEFAULT '',
    script          TEXT NOT NULL,
    config_schema   TEXT NOT NULL DEFAULT '[]',
    github_url      TEXT NOT NULL DEFAULT '',
    commit_hash     TEXT NOT NULL DEFAULT '',
    match_types     TEXT NOT NULL DEFAULT '*',
    connect_domains TEXT NOT NULL DEFAULT '*',
    grant_perms     TEXT NOT NULL DEFAULT '',
    timeout_sec     INTEGER NOT NULL DEFAULT 5,
    status          TEXT NOT NULL DEFAULT 'pending',
    reject_reason   TEXT NOT NULL DEFAULT '',
    reviewed_by     TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_plugin_versions_plugin ON plugin_versions(plugin_id);
CREATE INDEX IF NOT EXISTS idx_plugin_versions_status ON plugin_versions(status);

CREATE TABLE IF NOT EXISTS plugin_installs (
    plugin_id    TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    installed_at INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY (plugin_id, user_id)
);

CREATE TABLE IF NOT EXISTS webhook_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id          TEXT NOT NULL,
    channel_id      TEXT NOT NULL,
    message_id      INTEGER,
    plugin_id       TEXT NOT NULL DEFAULT '',
    plugin_version  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    request_url     TEXT NOT NULL DEFAULT '',
    request_method  TEXT NOT NULL DEFAULT '',
    request_body    TEXT NOT NULL DEFAULT '',
    response_status INTEGER NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    script_error    TEXT NOT NULL DEFAULT '',
    replies         TEXT NOT NULL DEFAULT '[]',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_channel ON webhook_logs(channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_bot ON webhook_logs(bot_id, created_at);

CREATE TABLE IF NOT EXISTS apps (
    id                   TEXT PRIMARY KEY,
    owner_id             TEXT NOT NULL,
    name                 TEXT NOT NULL,
    slug                 TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    icon                 TEXT NOT NULL DEFAULT '',
    icon_url             TEXT NOT NULL DEFAULT '',
    homepage             TEXT NOT NULL DEFAULT '',
    tools                TEXT NOT NULL DEFAULT '[]',
    events               TEXT NOT NULL DEFAULT '[]',
    scopes               TEXT NOT NULL DEFAULT '[]',
    oauth_setup_url      TEXT NOT NULL DEFAULT '',
    oauth_redirect_url   TEXT NOT NULL DEFAULT '',
    webhook_url          TEXT NOT NULL DEFAULT '',
    webhook_secret       TEXT NOT NULL DEFAULT '',
    webhook_verified     INTEGER NOT NULL DEFAULT 0,
    registry             TEXT NOT NULL DEFAULT '',
    version              TEXT NOT NULL DEFAULT '',
    readme               TEXT NOT NULL DEFAULT '',
    guide                TEXT NOT NULL DEFAULT '',
    config_schema        TEXT NOT NULL DEFAULT '',
    listing              TEXT NOT NULL DEFAULT 'unlisted',
    listing_reject_reason TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'active',
    created_at           INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at           INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE UNIQUE INDEX IF NOT EXISTS apps_slug_registry_key ON apps(slug, registry);

CREATE TABLE IF NOT EXISTS app_installations (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    bot_id          TEXT NOT NULL,
    app_token       TEXT NOT NULL UNIQUE,
    handle          TEXT NOT NULL DEFAULT '',
    config          TEXT NOT NULL DEFAULT '{}',
    scopes          TEXT NOT NULL DEFAULT '[]',
    tools           TEXT NOT NULL DEFAULT '[]',
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_app_installations_bot ON app_installations(bot_id);
CREATE INDEX IF NOT EXISTS idx_app_installations_token ON app_installations(app_token);
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_installations_bot_handle ON app_installations(bot_id, handle) WHERE handle != '';

CREATE TABLE IF NOT EXISTS app_event_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    installation_id TEXT NOT NULL,
    trace_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    event_id        TEXT NOT NULL,
    request_body    TEXT NOT NULL DEFAULT '',
    response_status INTEGER NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    retry_count     INTEGER NOT NULL DEFAULT 0,
    error           TEXT NOT NULL DEFAULT '',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_app_event_logs_inst ON app_event_logs(installation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_app_event_logs_trace ON app_event_logs(trace_id);

CREATE TABLE IF NOT EXISTS app_api_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    installation_id TEXT NOT NULL,
    trace_id        TEXT NOT NULL,
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    request_body    TEXT NOT NULL DEFAULT '',
    status_code     INTEGER NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    duration_ms     INTEGER NOT NULL DEFAULT 0,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_app_api_logs_inst ON app_api_logs(installation_id, created_at);

CREATE TABLE IF NOT EXISTS app_oauth_codes (
    code            TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL,
    bot_id          TEXT NOT NULL,
    state           TEXT NOT NULL,
    code_challenge  TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    expires_at      INTEGER NOT NULL DEFAULT (unixepoch() + 600)
);

CREATE TABLE IF NOT EXISTS registries (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    url             TEXT NOT NULL UNIQUE,
    enabled         INTEGER NOT NULL DEFAULT 1,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS trace_spans (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id        TEXT NOT NULL,
    span_id         TEXT NOT NULL,
    parent_span_id  TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'internal',
    status_code     TEXT NOT NULL DEFAULT 'unset',
    status_message  TEXT NOT NULL DEFAULT '',
    start_time      INTEGER NOT NULL,
    end_time        INTEGER NOT NULL DEFAULT 0,
    attributes      TEXT NOT NULL DEFAULT '{}',
    events          TEXT NOT NULL DEFAULT '[]',
    bot_id          TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_trace_spans_trace ON trace_spans(trace_id);
CREATE INDEX IF NOT EXISTS idx_trace_spans_bot ON trace_spans(bot_id, created_at) WHERE parent_span_id = '';
