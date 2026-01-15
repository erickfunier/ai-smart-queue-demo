CREATE TABLE IF NOT EXISTS insights (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id),
    diagnosis TEXT NOT NULL,
    recommendation TEXT NOT NULL,
    suggested_fix JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
