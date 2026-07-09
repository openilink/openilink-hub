-- +goose Up
-- Per-installation flag: when an installation sends a reply through Bot API,
-- prepend "@{handle} " to text messages so users in multi-channel setups can
-- tell which channel responded. Default off — opt-in to avoid noise in
-- single-channel deployments. See issue #248.
ALTER TABLE app_installations
    ADD COLUMN IF NOT EXISTS reply_prefix_handle BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE app_installations DROP COLUMN IF EXISTS reply_prefix_handle;
