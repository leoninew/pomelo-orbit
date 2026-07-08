-- v0.1.0: pomelo-orbit task queue schema

CREATE TABLE IF NOT EXISTS background_task (
    id VARCHAR(26) PRIMARY KEY,
    task_type VARCHAR(128) NOT NULL,
    payload_json LONGTEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    attempts INT NOT NULL,
    max_attempts INT NOT NULL,
    locked_by VARCHAR(255),
    locked_at DATETIME(3),
    started_at DATETIME(3),
    finished_at DATETIME(3),
    error_message TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE INDEX idx_background_task_status_created ON background_task(status, created_at);
CREATE INDEX idx_background_task_locked_at ON background_task(locked_at);
CREATE INDEX idx_background_task_type ON background_task(task_type);
