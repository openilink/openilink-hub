ALTER TABLE messages ADD COLUMN IF NOT EXISTS processed_at TIMESTAMPTZ;

-- Backfill: treat all existing inbound messages as already processed.
UPDATE messages SET processed_at = created_at WHERE direction = 'inbound';

CREATE INDEX IF NOT EXISTS idx_messages_unprocessed
  ON messages (bot_id, id)
  WHERE direction = 'inbound' AND processed_at IS NULL;
