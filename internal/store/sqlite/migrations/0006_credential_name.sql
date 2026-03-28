-- +goose Up
ALTER TABLE credentials ADD COLUMN name TEXT NOT NULL DEFAULT '';
