-- +goose Up
-- Hermes integration was unsupported (see issue #229 — message delivery never
-- worked reliably and upstream PR hadn't merged). Remove the builtin app row.
--
-- app_installations.app_id has ON DELETE CASCADE so installations are cleaned
-- up automatically. app_oauth_codes.app_id has no FK, so clear it explicitly
-- to avoid orphan rows (they would expire in ~10 min anyway, but a removal
-- migration is the right place to be tidy).
DELETE FROM app_oauth_codes WHERE app_id IN (
    SELECT id FROM apps WHERE slug = 'hermes' AND registry = 'builtin'
);
DELETE FROM apps WHERE slug = 'hermes' AND registry = 'builtin';

-- +goose Down
-- Re-seeding is handled by builtin.SeedApps at startup if the manifest is
-- restored; nothing to do here.
