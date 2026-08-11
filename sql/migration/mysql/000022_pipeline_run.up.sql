-- Domain: pipeline_run
CREATE TABLE IF NOT EXISTS pipeline_run (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26),
    repository_id VARCHAR(26) NOT NULL,
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    snapshot_id VARCHAR(26) NOT NULL,
    pipeline_id VARCHAR(26) NOT NULL,
    pipeline_name VARCHAR(255) NOT NULL,
    pipeline_version INT NOT NULL,
    `trigger` VARCHAR(64) NOT NULL,
    trigger_ref VARCHAR(255) NOT NULL,
    variables_snapshot LONGTEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    retry_of VARCHAR(26),
    started_at DATETIME(3),
    finished_at DATETIME(3),
    error_message TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_pipeline_run_pipeline_created ON pipeline_run(pipeline_id, created_at DESC);
CREATE INDEX idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX idx_pipeline_run_project ON pipeline_run(project_id);
CREATE INDEX idx_pipeline_run_repository_status ON pipeline_run(repository_id, status);

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
    UNIQUE KEY uq_pipeline_stage_run_run_stage (pipeline_run_id, stage_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX idx_pipeline_stage_run_status ON pipeline_stage_run(status);

CREATE TABLE IF NOT EXISTS pipeline_run_version_binding (
    pipeline_run_id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    application_name VARCHAR(255) NOT NULL,
    source_version_id VARCHAR(26) NOT NULL,
    source_version_label VARCHAR(255) NOT NULL,
    generated_version_id VARCHAR(26),
    generated_version_label VARCHAR(255)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_pipeline_run_version_binding_source_version
    ON pipeline_run_version_binding(source_version_id);
CREATE INDEX idx_pipeline_run_version_binding_generated_version
    ON pipeline_run_version_binding(generated_version_id);

CREATE TABLE IF NOT EXISTS artifact (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26),
    pipeline_run_id VARCHAR(26) NOT NULL,
    repository_id VARCHAR(26) NOT NULL DEFAULT '',
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    pipeline_id VARCHAR(26) NOT NULL DEFAULT '',
    pipeline_name VARCHAR(255) NOT NULL DEFAULT '',
    pipeline_stage_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    collector VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    location VARCHAR(1024),
    value TEXT,
    value_format VARCHAR(64),
    image_ref VARCHAR(512),
    local_image_sha256 VARCHAR(128),
    source_artifact_id VARCHAR(26),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (project_id) REFERENCES project(id),
    CONSTRAINT fk_artifact_source FOREIGN KEY (source_artifact_id) REFERENCES artifact(id) ON DELETE SET NULL,
    CONSTRAINT chk_artifact_collector_payload CHECK (
        (collector = 'file' AND location IS NOT NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'command' AND location IS NULL AND value IS NOT NULL AND value_format IN ('text', 'git_object_id') AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'docker_image' AND location IS NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NOT NULL AND local_image_sha256 IS NOT NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX idx_artifact_repository ON artifact(repository_id);
CREATE INDEX idx_artifact_project ON artifact(project_id);
CREATE INDEX idx_artifact_pipeline ON artifact(pipeline_id);
