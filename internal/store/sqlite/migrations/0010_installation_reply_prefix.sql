-- +goose Up
-- Per-installation flag: when an installation sends a reply through Bot API,
-- prepend "@{handle} " to text messages so users in multi-channel setups can
-- tell which channel responded. Default off — opt-in to avoid noise in
-- single-channel deployments. See issue #248.
ALTER TABLE app_installations ADD COLUMN reply_prefix_handle BOOLEAN NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite < 3.35 cannot DROP COLUMN; rollback skipped. The column defaults to 0
-- so leaving it in place after a downgrade is safe.
