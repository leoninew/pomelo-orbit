-- v0.1.0: pomelo-orbit task queue schema

CREATE TABLE IF NOT EXISTS background_task (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    max_attempts INTEGER NOT NULL,
    locked_by TEXT,
    locked_at DATETIME,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_background_task_status_created ON background_task(status, created_at);
CREATE INDEX IF NOT EXISTS idx_background_task_locked_at ON background_task(locked_at);
CREATE INDEX IF NOT EXISTS idx_background_task_type ON background_task(task_type);
