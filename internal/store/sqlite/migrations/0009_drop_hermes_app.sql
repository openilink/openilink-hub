-- +goose Up
-- Hermes integration was unsupported (see issue #229 — message delivery never
-- worked reliably and upstream PR hadn't merged). Remove the builtin app row;
-- ON DELETE CASCADE on app_installations.app_id cleans up any installations.
DELETE FROM apps WHERE slug = 'hermes' AND registry = 'builtin';

-- +goose Down
-- Re-seeding is handled by builtin.SeedApps at startup if the manifest is
-- restored; nothing to do here.
