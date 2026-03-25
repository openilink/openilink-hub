-- Add OAuth fields to apps for one-click install flow
ALTER TABLE apps ADD COLUMN IF NOT EXISTS setup_url TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS redirect_url TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN IF NOT EXISTS client_secret TEXT NOT NULL DEFAULT '';

-- Temporary OAuth codes for install flow
CREATE TABLE IF NOT EXISTS app_oauth_codes (
    code            TEXT PRIMARY KEY,
    app_id          TEXT NOT NULL,
    bot_id          TEXT NOT NULL,
    state           TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '10 minutes'
);
