-- Allow same app to be installed multiple times on same bot
ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_bot_id_key;

-- Each installation gets its own handle for @mention routing
ALTER TABLE app_installations ADD COLUMN IF NOT EXISTS handle TEXT NOT NULL DEFAULT '';

-- Handle must be unique per bot (but can be empty)
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_installations_bot_handle
    ON app_installations(bot_id, handle) WHERE handle != '';
