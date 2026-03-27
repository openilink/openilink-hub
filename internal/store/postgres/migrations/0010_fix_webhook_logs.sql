-- +goose Up

-- Add missing columns to webhook_logs if they don't exist
ALTER TABLE webhook_logs ADD COLUMN IF NOT EXISTS plugin_version TEXT NOT NULL DEFAULT '';
ALTER TABLE webhook_logs ADD COLUMN IF NOT EXISTS plugin_id TEXT NOT NULL DEFAULT '';
