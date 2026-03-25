-- Deduplicate existing messages: keep only the latest row per (bot_id, message_id).
DELETE FROM messages a
  USING messages b
  WHERE a.bot_id = b.bot_id
    AND a.message_id = b.message_id
    AND a.message_id IS NOT NULL
    AND a.id < b.id;

-- Prevent future duplicates.
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_bot_msgid
  ON messages (bot_id, message_id)
  WHERE message_id IS NOT NULL;
