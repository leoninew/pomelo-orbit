PRAGMA foreign_keys = OFF;

CREATE TABLE artifact_old (
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
    FOREIGN KEY (source_artifact_id) REFERENCES artifact(id) ON DELETE SET NULL,
    CHECK (
        (collector = 'file' AND location IS NOT NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'command' AND location IS NULL AND value IS NOT NULL AND value_format IN ('text', 'git_object_id') AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'docker_image' AND location IS NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NOT NULL AND local_image_sha256 IS NOT NULL)
    )
);

INSERT INTO artifact_old (
    id, project_id, pipeline_run_id, repository_id, repository_name, pipeline_id, pipeline_name,
    pipeline_stage_id, stage_name, collector, name, location, value, value_format, image_ref,
    local_image_sha256, source_artifact_id, created_at
)
SELECT
    id, project_id, pipeline_run_id, repository_id, repository_name, pipeline_id, pipeline_name,
    pipeline_stage_id, stage_name, collector, name, location, value, value_format, image_ref,
    local_image_sha256, source_artifact_id, created_at
FROM artifact;

DROP TABLE artifact;
ALTER TABLE artifact_old RENAME TO artifact;

CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX idx_artifact_repository ON artifact(repository_id);
CREATE INDEX idx_artifact_project ON artifact(project_id);
CREATE INDEX idx_artifact_pipeline ON artifact(pipeline_id);

PRAGMA foreign_keys = ON;
