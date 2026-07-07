-- +goose Up

CREATE TABLE IF NOT EXISTS wechat_pending_bindings (
    id                     BIGSERIAL PRIMARY KEY,
    event_id               TEXT NOT NULL UNIQUE,
    provider_bot_id        TEXT NOT NULL,
    bot_id                 TEXT NOT NULL,
    binding_id             TEXT NOT NULL,
    role_id                TEXT NOT NULL,
    session_id             TEXT NOT NULL,
    status                 TEXT NOT NULL DEFAULT 'pending_finalize',
    external_chat_id       TEXT NOT NULL DEFAULT '',
    last_finalize_event_id TEXT NOT NULL DEFAULT '',
    last_error             TEXT NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at             TIMESTAMPTZ NOT NULL,
    finalized_at           TIMESTAMPTZ,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_wechat_pending_bindings_lookup
    ON wechat_pending_bindings (bot_id, provider_bot_id, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_wechat_pending_bindings_expires
    ON wechat_pending_bindings (expires_at);

-- +goose Down

DROP TABLE IF EXISTS wechat_pending_bindings;

