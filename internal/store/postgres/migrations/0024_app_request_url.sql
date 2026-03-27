-- +goose Up

-- Move request_url, signing_secret, url_verified from app_installations to apps table.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Add columns to apps
ALTER TABLE apps ADD COLUMN request_url TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN signing_secret TEXT NOT NULL DEFAULT '';
ALTER TABLE apps ADD COLUMN url_verified BOOLEAN NOT NULL DEFAULT FALSE;

-- 2. Migrate data: take non-empty values from existing installations
UPDATE apps SET
  request_url = COALESCE((SELECT i.request_url FROM app_installations i WHERE i.app_id = apps.id AND i.request_url != '' ORDER BY i.updated_at DESC LIMIT 1), ''),
  signing_secret = COALESCE((SELECT i.signing_secret FROM app_installations i WHERE i.app_id = apps.id AND i.signing_secret != '' ORDER BY i.updated_at DESC LIMIT 1), ''),
  url_verified = COALESCE((SELECT bool_or(i.url_verified) FROM app_installations i WHERE i.app_id = apps.id), FALSE);

-- 3. Generate signing_secret for apps that still have none
UPDATE apps SET signing_secret = encode(gen_random_bytes(32), 'hex') WHERE signing_secret = '';

-- 4. Drop columns from app_installations
ALTER TABLE app_installations DROP COLUMN request_url;
ALTER TABLE app_installations DROP COLUMN signing_secret;
ALTER TABLE app_installations DROP COLUMN url_verified;
