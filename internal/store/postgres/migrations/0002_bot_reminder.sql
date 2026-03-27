-- +goose Up

-- Bot inactivity reminder
ALTER TABLE bots ADD COLUMN IF NOT EXISTS reminder_hours INT NOT NULL DEFAULT 0;
ALTER TABLE bots ADD COLUMN IF NOT EXISTS last_reminded_at TIMESTAMPTZ;
