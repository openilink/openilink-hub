-- +goose Up

ALTER TABLE app_installations ADD COLUMN IF NOT EXISTS tools JSONB NOT NULL DEFAULT '[]';
