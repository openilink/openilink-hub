CREATE TABLE IF NOT EXISTS message_traces (
    id          BIGSERIAL PRIMARY KEY,
    bot_id      TEXT NOT NULL,
    trace_id    TEXT NOT NULL,
    message_id  BIGINT NOT NULL DEFAULT 0,
    sender      TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',
    msg_type    TEXT NOT NULL DEFAULT '',
    spans       JSONB NOT NULL DEFAULT '[]',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_message_traces_bot ON message_traces(bot_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_message_traces_trace ON message_traces(trace_id);
