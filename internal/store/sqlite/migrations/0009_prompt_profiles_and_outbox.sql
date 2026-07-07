-- +goose Up

CREATE TABLE IF NOT EXISTS prompt_profiles (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    bot_id            TEXT NOT NULL,
    sender_user_id    TEXT NOT NULL,
    binding_id        TEXT NOT NULL,
    system_prompt     TEXT NOT NULL DEFAULT '',
    user_prompt       TEXT NOT NULL DEFAULT '',
    full_prompt       TEXT NOT NULL DEFAULT '',
    full_prompt_hash  TEXT NOT NULL DEFAULT '',
    prompt_version    INTEGER NOT NULL DEFAULT 0,
    source_updated_at INTEGER NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at        INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_prompt_profiles_bot_sender_active
    ON prompt_profiles (bot_id, sender_user_id)
    WHERE status = 'active';
CREATE UNIQUE INDEX IF NOT EXISTS uq_prompt_profiles_bot_sender_binding
    ON prompt_profiles (bot_id, sender_user_id, binding_id);
CREATE INDEX IF NOT EXISTS idx_prompt_profiles_bot_sender_status
    ON prompt_profiles (bot_id, sender_user_id, status);
CREATE INDEX IF NOT EXISTS idx_prompt_profiles_binding_status
    ON prompt_profiles (binding_id, status);

CREATE TABLE IF NOT EXISTS sync_outbox (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id      TEXT NOT NULL UNIQUE,
    event_type    TEXT NOT NULL,
    partition_key TEXT NOT NULL DEFAULT '',
    payload       TEXT NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'pending',
    retry_count   INTEGER NOT NULL DEFAULT 0,
    next_retry_at INTEGER NOT NULL DEFAULT (unixepoch()),
    last_error    TEXT NOT NULL DEFAULT '',
    sent_at       INTEGER,
    created_at    INTEGER NOT NULL DEFAULT (unixepoch()),
    updated_at    INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE INDEX IF NOT EXISTS idx_sync_outbox_status_next_retry_id
    ON sync_outbox (status, next_retry_at, id);
CREATE INDEX IF NOT EXISTS idx_sync_outbox_partition_id
    ON sync_outbox (partition_key, id);

CREATE TABLE IF NOT EXISTS admin_sync_inbox (
    event_id   TEXT PRIMARY KEY,
    created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

-- +goose Down

DROP TABLE IF EXISTS admin_sync_inbox;
DROP TABLE IF EXISTS sync_outbox;
DROP TABLE IF EXISTS prompt_profiles;
