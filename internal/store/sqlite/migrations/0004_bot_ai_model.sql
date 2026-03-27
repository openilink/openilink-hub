-- +goose Up
ALTER TABLE bots ADD COLUMN ai_model TEXT NOT NULL DEFAULT '';
