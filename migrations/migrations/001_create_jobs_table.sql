CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    queue TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INT NOT NULL DEFAULT 0,
    payload JSONB,
    error TEXT,
    scheduled_for TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ai_analyzed BOOLEAN NOT NULL DEFAULT FALSE
);
