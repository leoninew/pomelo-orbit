PRAGMA foreign_keys = OFF;

BEGIN IMMEDIATE;

CREATE TABLE artifact_new (
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
    FOREIGN KEY (source_artifact_id) REFERENCES artifact_new(id) ON DELETE SET NULL,
    CHECK (
        (collector = 'file' AND location IS NOT NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'command' AND location IS NULL AND value IS NOT NULL AND value_format IN ('text', 'git_object_id') AND image_ref IS NULL AND local_image_sha256 IS NULL AND source_artifact_id IS NULL) OR
        (collector = 'docker_image' AND location IS NULL AND value IS NULL AND value_format IS NULL AND image_ref IS NOT NULL AND local_image_sha256 IS NOT NULL AND source_artifact_id IS NOT NULL)
    )
);

INSERT INTO artifact_new (
    id, pipeline_run_id, pipeline_stage_id, stage_name, collector, name, location,
    value, value_format, image_ref, local_image_sha256, source_artifact_id, created_at,
    repository_id, repository_name, template_id, template_name, project_id
)
SELECT
    artifact.id, artifact.pipeline_run_id, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location,
    artifact_value.value, artifact_value.format, container_image_artifact.image_ref, container_image_artifact.local_image_sha256, artifact_input.input_artifact_id, artifact.created_at,
    artifact.repository_id, artifact.repository_name, artifact.template_id, artifact.template_name, artifact.project_id
FROM artifact
LEFT JOIN artifact_value ON artifact_value.artifact_id = artifact.id
LEFT JOIN container_image_artifact ON container_image_artifact.artifact_id = artifact.id
LEFT JOIN artifact_input ON artifact_input.output_artifact_id = artifact.id
WHERE
    (artifact.collector = 'file' AND artifact.location IS NOT NULL) OR
    (artifact.collector = 'command' AND artifact_value.value IS NOT NULL AND artifact_value.format IN ('text', 'git_object_id')) OR
    (artifact.collector = 'docker_image' AND container_image_artifact.image_ref IS NOT NULL AND container_image_artifact.local_image_sha256 IS NOT NULL AND artifact_input.input_artifact_id IS NOT NULL);

UPDATE pipeline_run_build_version_binding
SET artifact_id = NULL
WHERE artifact_id IS NOT NULL
  AND artifact_id NOT IN (SELECT id FROM artifact_new);

CREATE TABLE version_component_new (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    artifact_id TEXT,
    command_json TEXT NOT NULL DEFAULT '[]',
    pull_policy TEXT NOT NULL CHECK (pull_policy IN ('always', 'missing', 'never')),
    restart_policy TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    FOREIGN KEY (artifact_id) REFERENCES artifact_new(id) ON DELETE SET NULL,
    UNIQUE(version_id, name)
);

INSERT INTO version_component_new (
    id, version_id, name, image, artifact_id, command_json, pull_policy, restart_policy, created_at, updated_at
)
SELECT
    version_component.id, version_component.version_id, version_component.name, version_component.image,
    CASE WHEN version_component_artifact.artifact_id IN (SELECT id FROM artifact_new) THEN version_component_artifact.artifact_id END,
    version_component.command_json, version_component.pull_policy, version_component.restart_policy,
    version_component.created_at, version_component.updated_at
FROM version_component
LEFT JOIN version_component_artifact ON version_component_artifact.component_id = version_component.id;

DROP TABLE version_component_artifact;
DROP TABLE artifact_input;
DROP TABLE container_image_artifact;
DROP TABLE artifact_value;
DROP TABLE version_component;
DROP TABLE artifact;

ALTER TABLE artifact_new RENAME TO artifact;
ALTER TABLE version_component_new RENAME TO version_component;

CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX ix_artifact_repository_id ON artifact(repository_id);
CREATE INDEX idx_artifact_project ON artifact(project_id);
CREATE INDEX idx_version_component_version ON version_component(version_id);
CREATE INDEX idx_version_component_artifact_id ON version_component(artifact_id);

COMMIT;

PRAGMA foreign_keys = ON;
PRAGMA foreign_key_check;
