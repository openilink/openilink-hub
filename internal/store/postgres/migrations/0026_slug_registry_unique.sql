-- Remove kind column and fix slug uniqueness to per-registry namespace
ALTER TABLE apps DROP COLUMN IF EXISTS kind;
DROP INDEX IF EXISTS apps_slug_key;
CREATE UNIQUE INDEX IF NOT EXISTS apps_slug_registry_key ON apps (slug, registry);
