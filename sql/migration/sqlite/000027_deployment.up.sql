-- Domain: deployment
-- Tables: deployment
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS deployment (
    id TEXT PRIMARY KEY,
    application_id TEXT,
    application_name TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    env_file TEXT,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    finished_at DATETIME,
    duration_ms INTEGER,
    log_text TEXT,
    error_message TEXT,
    is_rollback INTEGER NOT NULL,
    rollback_from_deployment_id TEXT,
    project_id TEXT REFERENCES project(id),
    version_id TEXT REFERENCES version(id),
    service_id TEXT REFERENCES service(id),
    options_json TEXT,
    effective_plan_hash TEXT,
    command_text TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX IF NOT EXISTS idx_deployment_app ON deployment(application_id);
CREATE INDEX IF NOT EXISTS idx_deployment_app_name ON deployment(application_name);
CREATE INDEX IF NOT EXISTS idx_deployment_status ON deployment(status);
CREATE INDEX IF NOT EXISTS idx_deployment_started ON deployment(started_at);
CREATE INDEX IF NOT EXISTS idx_deployment_project ON deployment(project_id);
CREATE INDEX IF NOT EXISTS idx_deployment_service_success_hash ON deployment(service_id, status, effective_plan_hash, finished_at);
