-- Domain: pipeline_run
-- Tables: pipeline_run, pipeline_stage_run, artifact
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS pipeline_run (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    snapshot_id TEXT NOT NULL,
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    template_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_project_created ON pipeline_run(repository_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_project ON pipeline_run(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_repository_status ON pipeline_run(repository_id, status);

CREATE TABLE IF NOT EXISTS pipeline_stage_run (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_status ON pipeline_stage_run(status);

CREATE TABLE IF NOT EXISTS artifact (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    pipeline_stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    collector TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT,
    value TEXT,
    value_format TEXT,
    image_ref TEXT,
    local_image_sha256 TEXT,
    source_artifact_id TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    repository_id TEXT NOT NULL DEFAULT '',
    repository_name TEXT NOT NULL DEFAULT '',
    template_id TEXT NOT NULL DEFAULT '',
    template_name TEXT NOT NULL DEFAULT '',
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE,
    FOREIGN KEY (source_artifact_id) REFERENCES artifact(id) ON DELETE SET NULL,
    CHECK (
        (collector = 'file' AND location IS NOT NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'command' AND location IS NULL AND value IS NOT NULL AND value_format IN ('text', 'git_object_id') AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'docker_image' AND location IS NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NOT NULL AND local_image_sha256 IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX IF NOT EXISTS ix_artifact_repository_id ON artifact (repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
