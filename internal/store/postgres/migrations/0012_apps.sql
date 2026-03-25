-- App platform: apps, installations, event logs, API logs

CREATE TABLE IF NOT EXISTS apps (
    id              TEXT PRIMARY KEY,
    owner_id        TEXT NOT NULL,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    description     TEXT NOT NULL DEFAULT '',
    icon            TEXT NOT NULL DEFAULT '',
    icon_url        TEXT NOT NULL DEFAULT '',
    homepage        TEXT NOT NULL DEFAULT '',
    tools           JSONB NOT NULL DEFAULT '[]',
    events          JSONB NOT NULL DEFAULT '[]',
    scopes          JSONB NOT NULL DEFAULT '[]',
    status          TEXT NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS app_installations (
    id              TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    bot_id          TEXT NOT NULL,
    app_token       TEXT NOT NULL UNIQUE,
    signing_secret  TEXT NOT NULL,
    request_url     TEXT NOT NULL DEFAULT '',
    url_verified    BOOLEAN NOT NULL DEFAULT FALSE,
    config          JSONB NOT NULL DEFAULT '{}',
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, bot_id)
);
CREATE INDEX IF NOT EXISTS idx_app_installations_bot ON app_installations(bot_id);
CREATE INDEX IF NOT EXISTS idx_app_installations_token ON app_installations(app_token);

CREATE TABLE IF NOT EXISTS app_event_logs (
    id              BIGSERIAL PRIMARY KEY,
    installation_id TEXT NOT NULL,
    trace_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    event_id        TEXT NOT NULL,
    request_body    TEXT NOT NULL DEFAULT '',
    response_status INT NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',
    retry_count     INT NOT NULL DEFAULT 0,
    error           TEXT NOT NULL DEFAULT '',
    duration_ms     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_app_event_logs_inst ON app_event_logs(installation_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_app_event_logs_trace ON app_event_logs(trace_id);

CREATE TABLE IF NOT EXISTS app_api_logs (
    id              BIGSERIAL PRIMARY KEY,
    installation_id TEXT NOT NULL,
    trace_id        TEXT NOT NULL,
    method          TEXT NOT NULL,
    path            TEXT NOT NULL,
    request_body    TEXT NOT NULL DEFAULT '',
    status_code     INT NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',
    duration_ms     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_app_api_logs_inst ON app_api_logs(installation_id, created_at DESC);
