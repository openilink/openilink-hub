-- +goose Up
ALTER TABLE bots ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
