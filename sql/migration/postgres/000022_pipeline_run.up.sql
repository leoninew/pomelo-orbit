-- Domain: pipeline_run
CREATE TABLE IF NOT EXISTS pipeline_run (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL DEFAULT '',
    snapshot_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    repository_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_pipeline_created ON pipeline_run(pipeline_id, created_at DESC);
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
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    exit_code INTEGER,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_status ON pipeline_stage_run(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pipeline_stage_run_run_stage
    ON pipeline_stage_run(pipeline_run_id, stage_id);

CREATE TABLE IF NOT EXISTS pipeline_run_version_binding (
    pipeline_run_id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    source_version_label TEXT NOT NULL,
    generated_version_id TEXT,
    generated_version_label TEXT
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_version_binding_source_version
    ON pipeline_run_version_binding(source_version_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_version_binding_generated_version
    ON pipeline_run_version_binding(generated_version_id);

CREATE TABLE IF NOT EXISTS artifact (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES project(id),
    pipeline_run_id TEXT NOT NULL,
    repository_id TEXT NOT NULL DEFAULT '',
    repository_name TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    pipeline_name TEXT NOT NULL DEFAULT '',
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
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    FOREIGN KEY (source_artifact_id) REFERENCES artifact(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX IF NOT EXISTS idx_artifact_repository ON artifact(repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
CREATE INDEX IF NOT EXISTS idx_artifact_pipeline ON artifact(pipeline_id);
