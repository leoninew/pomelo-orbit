-- Domain: task
-- Tables: background_task
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS background_task (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    max_attempts INTEGER NOT NULL,
    locked_by TEXT,
    locked_at TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX IF NOT EXISTS idx_background_task_status_created ON background_task(status, created_at);
CREATE INDEX IF NOT EXISTS idx_background_task_locked_at ON background_task(locked_at);
CREATE INDEX IF NOT EXISTS idx_background_task_type ON background_task(task_type);
