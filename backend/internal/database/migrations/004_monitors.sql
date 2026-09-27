CREATE TABLE IF NOT EXISTS monitors (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_url TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds IN (5, 60, 3600)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS monitors_user_id_created_at_idx ON monitors(user_id, created_at DESC);
