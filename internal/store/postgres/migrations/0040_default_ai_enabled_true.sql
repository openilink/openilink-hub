-- +goose Up

ALTER TABLE bots
  ALTER COLUMN ai_enabled SET DEFAULT TRUE;

UPDATE bots
SET ai_enabled = TRUE
WHERE ai_enabled = FALSE;

-- +goose Down

ALTER TABLE bots
  ALTER COLUMN ai_enabled SET DEFAULT FALSE;
