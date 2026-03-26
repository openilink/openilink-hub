-- Rename columns
ALTER TABLE apps RENAME COLUMN setup_url TO oauth_setup_url;
ALTER TABLE apps RENAME COLUMN redirect_url TO oauth_redirect_url;
ALTER TABLE apps RENAME COLUMN request_url TO webhook_url;
ALTER TABLE apps RENAME COLUMN signing_secret TO webhook_secret;
ALTER TABLE apps RENAME COLUMN url_verified TO webhook_verified;

-- Add new columns
ALTER TABLE apps ADD COLUMN kind TEXT NOT NULL DEFAULT 'app';
ALTER TABLE apps ADD COLUMN registry TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN version TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN readme TEXT NOT NULL DEFAULT '';

-- Replace listed + listing_status with single listing column
ALTER TABLE apps ADD COLUMN listing TEXT NOT NULL DEFAULT 'unlisted';
UPDATE apps SET listing = CASE
    WHEN listed = TRUE THEN 'listed'
    WHEN listing_status = 'pending' THEN 'pending'
    WHEN listing_status = 'rejected' THEN 'rejected'
    ELSE 'unlisted'
END;
ALTER TABLE apps DROP COLUMN listed;
ALTER TABLE apps DROP COLUMN listing_status;

-- Drop client_secret (PKCE replaces it)
ALTER TABLE apps DROP COLUMN client_secret;

-- Migrate scope values
UPDATE apps SET scopes = REPLACE(REPLACE(REPLACE(scopes::text,
    'messages.send', 'message:write'),
    'contacts.read', 'contact:read'),
    'bot.read', 'bot:read')::jsonb
WHERE scopes != '[]'::jsonb;

-- Installation scopes
ALTER TABLE app_installations ADD COLUMN scopes JSONB NOT NULL DEFAULT '[]';

-- PKCE
ALTER TABLE app_oauth_codes ADD COLUMN code_challenge TEXT NOT NULL DEFAULT '';

-- Registries table
CREATE TABLE IF NOT EXISTS registries (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL UNIQUE,
    enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
