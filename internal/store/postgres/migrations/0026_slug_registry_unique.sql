-- +goose Up

-- Remove kind column and fix slug uniqueness to per-registry namespace
ALTER TABLE apps DROP COLUMN IF EXISTS kind;

-- Drop slug uniqueness (could be index or constraint depending on how it was created)
ALTER TABLE apps DROP CONSTRAINT IF EXISTS apps_slug_key;
DROP INDEX IF EXISTS apps_slug_key;

-- Create per-registry namespace uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS apps_slug_registry_key ON apps (slug, registry);
