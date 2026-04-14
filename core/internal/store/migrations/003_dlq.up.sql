CREATE TABLE IF NOT EXISTS dlq (
    id TEXT PRIMARY KEY,
    data JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_dlq_created ON dlq ((data->>'created_at'));
