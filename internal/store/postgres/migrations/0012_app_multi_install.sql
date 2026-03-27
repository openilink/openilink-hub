-- +goose Up

-- Allow multiple installations of the same app on the same bot (differentiated by handle)
ALTER TABLE app_installations DROP CONSTRAINT IF EXISTS app_installations_app_id_bot_id_key;
