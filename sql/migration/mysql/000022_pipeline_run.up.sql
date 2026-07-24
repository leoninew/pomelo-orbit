-- Domain: pipeline_run
-- Tables: pipeline_run, pipeline_stage_run, artifact
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS pipeline_run (
    id VARCHAR(26) PRIMARY KEY,
    repository_id VARCHAR(26) NOT NULL,
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    snapshot_id VARCHAR(26) NOT NULL,
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    template_version INT NOT NULL,
    `trigger` VARCHAR(64) NOT NULL,
    trigger_ref VARCHAR(255) NOT NULL,
    variables_snapshot VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    status VARCHAR(32) NOT NULL,
    retry_of VARCHAR(26),
    started_at DATETIME(3),
    finished_at DATETIME(3),
    error_message TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
);

CREATE INDEX idx_pipeline_run_project_created ON pipeline_run(repository_id, created_at DESC);
CREATE INDEX idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX idx_pipeline_run_project ON pipeline_run(project_id);

CREATE TABLE IF NOT EXISTS pipeline_stage_run (
    id VARCHAR(26) PRIMARY KEY,
    pipeline_run_id VARCHAR(26) NOT NULL,
    stage_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at DATETIME(3),
    finished_at DATETIME(3),
    exit_code INT,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id)
);

CREATE INDEX idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX idx_pipeline_stage_run_status ON pipeline_stage_run(status);

CREATE TABLE IF NOT EXISTS artifact (
    id VARCHAR(26) PRIMARY KEY,
    pipeline_run_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(1024),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    repository_id VARCHAR(26) NOT NULL DEFAULT '',
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
);

CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX ix_artifact_repository_id ON artifact (repository_id);
CREATE INDEX idx_artifact_project ON artifact(project_id);
