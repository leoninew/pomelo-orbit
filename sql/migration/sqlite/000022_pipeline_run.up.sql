-- Domain: pipeline_run
CREATE TABLE IF NOT EXISTS pipeline_run (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES project(id),
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL DEFAULT '',
    snapshot_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
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
    started_at DATETIME,
    finished_at DATETIME,
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_status ON pipeline_stage_run(status);

CREATE TABLE IF NOT EXISTS pipeline_run_version_binding (
    pipeline_run_id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    source_version_label TEXT NOT NULL,
    generated_version_id TEXT,
    generated_version_label TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
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
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
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
CREATE INDEX IF NOT EXISTS idx_artifact_repository ON artifact(repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
CREATE INDEX IF NOT EXISTS idx_artifact_pipeline ON artifact(pipeline_id);
