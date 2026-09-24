-- name: PipelineById :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline
WHERE id = sqlc.arg(id)
  AND (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  );

-- name: PipelineByName :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline
WHERE kind = sqlc.arg(kind)
  AND (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  )
  AND name = sqlc.arg(name);

-- name: CountPipelines :one
SELECT COUNT(*) FROM pipeline
WHERE (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  )
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern));

-- name: ListPipelines :many
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline
WHERE (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  )
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern))
ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: CreatePipeline :exec
INSERT INTO pipeline (id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipeline :exec
UPDATE pipeline
SET name = sqlc.arg(name),
    description = sqlc.arg(description),
    variable_declarations = sqlc.arg(variable_declarations),
    version = sqlc.arg(version),
    application_id = sqlc.arg(application_id),
    application_name = sqlc.arg(application_name),
    version_fork_strategy = sqlc.arg(version_fork_strategy),
    fixed_version_id = sqlc.arg(fixed_version_id),
    fixed_version_label = sqlc.arg(fixed_version_label),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  );

-- name: DeletePipeline :exec
DELETE FROM pipeline
WHERE id = sqlc.arg(id)
  AND (
    (kind = 'template' AND project_id IS NULL)
    OR (kind = 'application' AND project_id = sqlc.arg(project_id))
  );

-- name: UpdateApplicationPipelineIfVersion :execrows
UPDATE pipeline
SET source_template_name = sqlc.arg(source_template_name),
    source_template_version = sqlc.arg(source_template_version),
    name = sqlc.arg(name),
    description = sqlc.arg(description),
    variable_declarations = sqlc.arg(variable_declarations),
    version = sqlc.arg(version),
    application_id = sqlc.arg(application_id),
    application_name = sqlc.arg(application_name),
    version_fork_strategy = sqlc.arg(version_fork_strategy),
    fixed_version_id = sqlc.arg(fixed_version_id),
    fixed_version_label = sqlc.arg(fixed_version_label),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id)
  AND kind = 'application'
  AND version = sqlc.arg(expected_version);

-- name: CountPipelineStageTemplates :one
SELECT COUNT(*) FROM pipeline_stage
WHERE project_id IS NULL
  AND kind = 'template'
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern));

-- name: ListPipelineStageTemplates :many
SELECT id, COALESCE(project_id, '') AS project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage
WHERE project_id IS NULL
  AND kind = 'template'
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern))
ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: PipelineStageTemplateById :one
SELECT id, COALESCE(project_id, '') AS project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND kind = 'template';

-- name: PipelineStageTemplateByName :one
SELECT id, COALESCE(project_id, '') AS project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage
WHERE project_id IS NULL
  AND name = sqlc.arg(name)
  AND kind = 'template';

-- name: InsertPipelineStageTemplate :exec
INSERT INTO pipeline_stage (id, project_id, kind, pipeline_id, name, image, script, description,
                            version, source_template_stage_id, source_template_stage_name,
                            source_template_stage_version, source_template_stage_description,
                            artifacts, depends_on, sort_order, created_at, updated_at)
VALUES (?, NULL, 'template', NULL, ?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, ?, NULL, NULL, ?, ?);

-- name: UpdatePipelineStageTemplate :exec
UPDATE pipeline_stage
SET name = sqlc.arg(name),
    image = sqlc.arg(image),
    script = sqlc.arg(script),
    description = sqlc.arg(description),
    artifacts = sqlc.arg(artifacts),
    version = sqlc.arg(version),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND kind = 'template';

-- name: DeletePipelineStageTemplate :exec
DELETE FROM pipeline_stage
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND kind = 'template';

-- name: TemplatePipelineStageReferences :many
SELECT id, pipeline_id, source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, name, image,
        script, description, artifacts, depends_on, sort_order, created_at, updated_at
FROM pipeline_stage_reference
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND EXISTS (
    SELECT 1
    FROM pipeline
    WHERE pipeline.id = pipeline_stage_reference.pipeline_id
      AND pipeline.kind = 'template'
      AND pipeline.project_id IS NULL
  )
ORDER BY sort_order, id;

-- name: DeleteTemplatePipelineStageReferences :exec
DELETE FROM pipeline_stage_reference
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND EXISTS (
    SELECT 1
    FROM pipeline
    WHERE pipeline.id = pipeline_stage_reference.pipeline_id
      AND pipeline.kind = 'template'
      AND pipeline.project_id IS NULL
  );

-- name: InsertTemplatePipelineStageReference :exec
INSERT INTO pipeline_stage_reference (id, pipeline_id, source_template_stage_id,
                                      source_template_stage_name, source_template_stage_version,
                                      source_template_stage_description, name, image, script,
                                       description, artifacts, depends_on, sort_order, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: ApplicationPipelineStages :many
SELECT id, project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND project_id = sqlc.arg(project_id)
  AND kind = 'application'
ORDER BY sort_order, id;

-- name: DeleteApplicationPipelineStages :exec
DELETE FROM pipeline_stage
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND project_id = sqlc.arg(project_id)
  AND kind = 'application';

-- name: InsertApplicationPipelineStage :exec
INSERT INTO pipeline_stage (id, project_id, kind, pipeline_id, name, image, script, description,
                            version, source_template_stage_id, source_template_stage_name,
                            source_template_stage_version, source_template_stage_description,
                            artifacts, depends_on, sort_order, created_at, updated_at)
VALUES (?, ?, 'application', ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: PipelineSnapshotById :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot
WHERE id = sqlc.arg(id)
  AND (project_id = sqlc.arg(project_id) OR project_id IS NULL);

-- name: LatestPipelineSnapshot :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND (project_id = sqlc.arg(project_id) OR project_id IS NULL)
ORDER BY pipeline_version DESC, created_at DESC
LIMIT 1;

-- name: PipelineSnapshotAtVersion :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot
WHERE pipeline_id = sqlc.arg(pipeline_id)
  AND pipeline_version = sqlc.arg(pipeline_version)
  AND (project_id = sqlc.arg(project_id) OR project_id IS NULL);

-- name: InsertPipelineSnapshot :exec
INSERT INTO pipeline_snapshot (id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CountCDConfigurationReferences :one
SELECT (
  SELECT COUNT(*)
  FROM pipeline
  WHERE pipeline.project_id = sqlc.arg(project_id)
    AND (application_id IS NOT NULL OR fixed_version_id IS NOT NULL)
) + (
  SELECT COUNT(*)
  FROM pipeline_snapshot
  WHERE pipeline_snapshot.project_id = sqlc.arg(project_id)
    AND (application_id IS NOT NULL OR fixed_version_id IS NOT NULL)
);
