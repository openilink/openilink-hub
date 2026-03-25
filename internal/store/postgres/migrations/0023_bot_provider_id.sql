-- Extract provider_id from credentials JSON into a proper column.
ALTER TABLE bots ADD COLUMN IF NOT EXISTS provider_id TEXT NOT NULL DEFAULT '';

-- Backfill from existing credentials for iLink bots.
UPDATE bots SET provider_id = credentials->>'bot_id'
  WHERE provider = 'ilink' AND credentials->>'bot_id' IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_bots_provider_id
  ON bots (provider, provider_id)
  WHERE provider_id != '';
