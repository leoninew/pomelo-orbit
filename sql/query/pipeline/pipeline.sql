-- name: PipelineByID :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline WHERE id = ?;

-- name: PipelineByName :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline WHERE project_id = ? AND name = ?;

-- name: CountPipelines :one
SELECT COUNT(*) FROM pipeline
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern));

-- name: ListPipelines :many
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern))
ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: CreatePipeline :exec
INSERT INTO pipeline (id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipeline :exec
UPDATE pipeline SET name = ?, description = ?, variable_declarations = ?, version = ?, application_id = ?, application_name = ?, version_fork_strategy = ?, fixed_version_id = ?, fixed_version_label = ?, updated_at = ? WHERE id = ?;

-- name: DeletePipeline :exec
DELETE FROM pipeline WHERE id = ?;

-- name: CountPipelineStageTemplates :one
SELECT COUNT(*) FROM pipeline_stage
WHERE project_id = sqlc.arg(project_id)
  AND kind = 'template'
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern));

-- name: ListPipelineStageTemplates :many
SELECT id, project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage
WHERE project_id = sqlc.arg(project_id)
  AND kind = 'template'
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern))
ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: PipelineStageTemplateByID :one
SELECT id, project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage WHERE id = ? AND kind = 'template';

-- name: PipelineStageTemplateByName :one
SELECT id, project_id, kind, pipeline_id, name, image, script, description, version,
       source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, artifacts, depends_on, sort_order,
       created_at, updated_at
FROM pipeline_stage WHERE project_id = ? AND name = ? AND kind = 'template';

-- name: InsertPipelineStageTemplate :exec
INSERT INTO pipeline_stage (id, project_id, kind, pipeline_id, name, image, script, description,
                            version, source_template_stage_id, source_template_stage_name,
                            source_template_stage_version, source_template_stage_description,
                            artifacts, depends_on, sort_order, created_at, updated_at)
VALUES (?, ?, 'template', NULL, ?, ?, ?, ?, ?, NULL, NULL, NULL, NULL, ?, NULL, NULL, ?, ?);

-- name: UpdatePipelineStageTemplate :exec
UPDATE pipeline_stage
SET name = ?, image = ?, script = ?, description = ?, artifacts = ?, version = ?, updated_at = ?
WHERE id = ? AND kind = 'template';

-- name: DeletePipelineStageTemplate :exec
DELETE FROM pipeline_stage WHERE id = ? AND kind = 'template';

-- name: TemplatePipelineStageReferences :many
SELECT id, pipeline_id, source_template_stage_id, source_template_stage_name,
       source_template_stage_version, source_template_stage_description, name, image,
        script, description, artifacts, depends_on, sort_order, created_at, updated_at
FROM pipeline_stage_reference WHERE pipeline_id = ? ORDER BY sort_order, id;

-- name: DeleteTemplatePipelineStageReferences :exec
DELETE FROM pipeline_stage_reference WHERE pipeline_id = ?;

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
FROM pipeline_stage WHERE pipeline_id = ? AND kind = 'application' ORDER BY sort_order, id;

-- name: DeleteApplicationPipelineStages :exec
DELETE FROM pipeline_stage WHERE pipeline_id = ? AND kind = 'application';

-- name: InsertApplicationPipelineStage :exec
INSERT INTO pipeline_stage (id, project_id, kind, pipeline_id, name, image, script, description,
                            version, source_template_stage_id, source_template_stage_name,
                            source_template_stage_version, source_template_stage_description,
                            artifacts, depends_on, sort_order, created_at, updated_at)
VALUES (?, ?, 'application', ?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: PipelineSnapshotByID :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot WHERE id = ?;

-- name: LatestPipelineSnapshot :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot WHERE pipeline_id = ? ORDER BY pipeline_version DESC, created_at DESC LIMIT 1;

-- name: InsertPipelineSnapshot :exec
INSERT INTO pipeline_snapshot (id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
