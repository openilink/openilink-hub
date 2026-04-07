-- +goose Up
CREATE TABLE IF NOT EXISTS cron_jobs (
    id          TEXT PRIMARY KEY,
    bot_id      TEXT NOT NULL REFERENCES bots(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    cron_expr   TEXT NOT NULL,
    message     TEXT NOT NULL,
    recipient   TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at BIGINT,
    next_run_at BIGINT,
    created_at  BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM NOW())::BIGINT
);

CREATE INDEX idx_cron_jobs_bot ON cron_jobs(bot_id);
CREATE INDEX idx_cron_jobs_next ON cron_jobs(enabled, next_run_at);

-- +goose Down
DROP TABLE IF EXISTS cron_jobs;
