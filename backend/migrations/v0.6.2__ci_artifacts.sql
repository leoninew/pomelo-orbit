-- CI Phase 3: 制品记录

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    job_name TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,  -- file 类型时的宿主机路径，docker_image 为空
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(pipeline_run_id);
