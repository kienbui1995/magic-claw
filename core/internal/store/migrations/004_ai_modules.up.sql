CREATE TABLE IF NOT EXISTS prompts (
    id TEXT PRIMARY KEY,
    data JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS memory_turns (
    id SERIAL PRIMARY KEY,
    session_id TEXT NOT NULL,
    data JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memory_turns_session ON memory_turns (session_id);
