-- Change slug uniqueness from global to per-registry namespace
DROP INDEX IF EXISTS apps_slug_key;
CREATE UNIQUE INDEX apps_slug_registry_key ON apps (slug, registry);
