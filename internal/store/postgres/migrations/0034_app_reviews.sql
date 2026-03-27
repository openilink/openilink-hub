CREATE TABLE IF NOT EXISTS app_reviews (
    id         TEXT PRIMARY KEY,
    app_id     TEXT NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    action     TEXT NOT NULL,
    actor_id   TEXT NOT NULL,
    reason     TEXT NOT NULL DEFAULT '',
    version    TEXT NOT NULL DEFAULT '',
    snapshot   TEXT NOT NULL DEFAULT '{}',
    created_at BIGINT NOT NULL DEFAULT (EXTRACT(EPOCH FROM NOW())::BIGINT)
);
CREATE INDEX IF NOT EXISTS idx_app_reviews_app_id ON app_reviews(app_id);
