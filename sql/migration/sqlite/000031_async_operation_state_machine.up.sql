-- Domain: async operation state machine (ordered after 000030 pipeline stage template library)
-- Rebuild preserves all task audit fields and its frozen retry budget.
CREATE TABLE background_task_new (
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

INSERT INTO background_task_new (
    id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
    started_at, finished_at, error_message, created_at, updated_at
)
SELECT
    id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
    started_at, finished_at, error_message, created_at, updated_at
FROM background_task;

DROP TABLE background_task;
ALTER TABLE background_task_new RENAME TO background_task;

CREATE INDEX idx_background_task_status_created ON background_task(status, created_at);
CREATE INDEX idx_background_task_locked_at ON background_task(locked_at);
CREATE INDEX idx_background_task_type ON background_task(task_type);

CREATE UNIQUE INDEX uq_pipeline_stage_run_run_stage
    ON pipeline_stage_run(pipeline_run_id, stage_id);

CREATE UNIQUE INDEX uq_deployment_active_service
    ON deployment(service_id)
    WHERE service_id IS NOT NULL AND status IN ('waiting_to_run', 'running');

CREATE UNIQUE INDEX uq_pipeline_run_active_repository
    ON pipeline_run(repository_id)
    WHERE status IN ('waiting_to_run', 'running');
