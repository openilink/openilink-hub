-- +goose Up

-- Replace message_traces with OTel-style span storage
DROP TABLE IF EXISTS message_traces;

CREATE TABLE trace_spans (
    id              BIGSERIAL PRIMARY KEY,
    trace_id        TEXT NOT NULL,
    span_id         TEXT NOT NULL,
    parent_span_id  TEXT NOT NULL DEFAULT '',
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT 'internal',  -- internal, client, server
    status_code     TEXT NOT NULL DEFAULT 'unset',     -- unset, ok, error
    status_message  TEXT NOT NULL DEFAULT '',
    start_time      BIGINT NOT NULL,                   -- unix milliseconds
    end_time        BIGINT NOT NULL DEFAULT 0,         -- unix milliseconds
    attributes      JSONB NOT NULL DEFAULT '{}',
    events          JSONB NOT NULL DEFAULT '[]',

    -- Denormalized for fast listing
    bot_id          TEXT NOT NULL DEFAULT '',

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trace_spans_trace ON trace_spans(trace_id);
CREATE INDEX idx_trace_spans_bot ON trace_spans(bot_id, created_at DESC) WHERE parent_span_id = '';
