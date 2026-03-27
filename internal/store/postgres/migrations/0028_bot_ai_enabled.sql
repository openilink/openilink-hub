-- +goose Up

ALTER TABLE bots ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- Migrate AI enabled from channel ai_config to bot level
UPDATE bots SET ai_enabled = TRUE
WHERE id IN (
  SELECT DISTINCT bot_id FROM channels
  WHERE ai_config IS NOT NULL
    AND ai_config::text != ''
    AND ai_config::text != 'null'
    AND (ai_config->>'enabled')::boolean = true
) AND ai_enabled = FALSE;
