-- OpeniLink Hub schema v1 (consolidated)

CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT NOT NULL DEFAULT '',
    display_name  TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT '',
    role          TEXT NOT NULL DEFAULT 'member',
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email != '';

CREATE TABLE IF NOT EXISTS credentials (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    public_key       BYTEA NOT NULL,
    attestation_type TEXT NOT NULL DEFAULT '',
    transport        TEXT NOT NULL DEFAULT '[]',
    sign_count       INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_credentials_user ON credentials(user_id);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS oauth_accounts (
    provider    TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    username    TEXT NOT NULL DEFAULT '',
    avatar_url  TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (provider, provider_id)
);
CREATE INDEX IF NOT EXISTS idx_oauth_user ON oauth_accounts(user_id);

CREATE TABLE IF NOT EXISTS bots (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    provider    TEXT NOT NULL DEFAULT 'ilink',
    status      TEXT NOT NULL DEFAULT 'disconnected',
    credentials JSONB NOT NULL DEFAULT '{}',
    sync_state  JSONB NOT NULL DEFAULT '{}',
    msg_count   BIGINT NOT NULL DEFAULT 0,
    last_msg_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bots_user ON bots(user_id);

CREATE TABLE IF NOT EXISTS channels (
    id             TEXT PRIMARY KEY,
    bot_id         TEXT NOT NULL,
    name           TEXT NOT NULL,
    handle         TEXT NOT NULL DEFAULT '',
    api_key        TEXT NOT NULL UNIQUE,
    filter_rule    JSONB NOT NULL DEFAULT '{}',
    ai_config      JSONB NOT NULL DEFAULT '{}',
    webhook_config JSONB NOT NULL DEFAULT '{}',
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    last_seq       BIGINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_channels_bot ON channels(bot_id);

-- Messages table mirrors WeChat WeixinMessage 1:1
CREATE TABLE IF NOT EXISTS messages (
    id             BIGSERIAL PRIMARY KEY,
    bot_id         TEXT NOT NULL,
    channel_id     TEXT DEFAULT NULL,
    direction      TEXT NOT NULL,
    -- WeChat WeixinMessage fields
    seq            BIGINT,
    message_id     BIGINT,
    from_user_id   TEXT NOT NULL DEFAULT '',
    to_user_id     TEXT NOT NULL DEFAULT '',
    client_id      TEXT NOT NULL DEFAULT '',
    create_time_ms BIGINT,
    update_time_ms BIGINT,
    delete_time_ms BIGINT,
    session_id     TEXT NOT NULL DEFAULT '',
    group_id       TEXT NOT NULL DEFAULT '',
    message_type   INT NOT NULL DEFAULT 0,
    message_state  INT NOT NULL DEFAULT 0,
    item_list      JSONB NOT NULL DEFAULT '[]',
    context_token  TEXT NOT NULL DEFAULT '',
    -- Operational
    media_status   TEXT NOT NULL DEFAULT '',
    media_keys     JSONB NOT NULL DEFAULT '{}',
    raw            JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_bot ON messages(bot_id, id);
CREATE INDEX IF NOT EXISTS idx_messages_seq ON messages(bot_id, seq) WHERE seq IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_from ON messages(bot_id, from_user_id) WHERE from_user_id != '';

CREATE TABLE IF NOT EXISTS system_config (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
