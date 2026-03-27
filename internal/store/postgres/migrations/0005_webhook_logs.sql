-- +goose Up

-- Webhook request logs
CREATE TABLE IF NOT EXISTS webhook_logs (
    id              BIGSERIAL PRIMARY KEY,
    bot_id          TEXT NOT NULL,
    channel_id      TEXT NOT NULL,
    message_id      BIGINT,
    plugin_id       TEXT NOT NULL DEFAULT '',
    plugin_version  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending/requesting/success/failed/skipped/error

    -- Request (filled after onRequest)
    request_url     TEXT NOT NULL DEFAULT '',
    request_method  TEXT NOT NULL DEFAULT '',
    request_body    TEXT NOT NULL DEFAULT '',

    -- Response (filled after HTTP)
    response_status INT NOT NULL DEFAULT 0,
    response_body   TEXT NOT NULL DEFAULT '',

    -- Script result
    script_error    TEXT NOT NULL DEFAULT '',
    replies         JSONB NOT NULL DEFAULT '[]',

    duration_ms     INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_channel ON webhook_logs(channel_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_bot ON webhook_logs(bot_id, created_at DESC);
