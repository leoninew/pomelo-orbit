-- name: PipelineByID :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline WHERE id = ?;

-- name: PipelineByName :one
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline WHERE project_id = ? AND name = ?;

-- name: CountPipelines :one
SELECT COUNT(*) FROM pipeline
WHERE project_id = sqlc.arg(project_id)
  AND (sqlc.arg(kind) = '' OR kind = sqlc.arg(kind))
  AND (sqlc.arg(search) = '' OR name LIKE sqlc.arg(search_pattern));

-- name: ListPipelines :many
SELECT id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline
WHERE project_id = sqlc.arg(project_id)
  AND (sqlc.arg(kind) = '' OR kind = sqlc.arg(kind))
  AND (sqlc.arg(search) = '' OR name LIKE sqlc.arg(search_pattern))
ORDER BY updated_at DESC, id LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CreatePipeline :exec
INSERT INTO pipeline (id, project_id, kind, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, name, description, variable_declarations, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipeline :exec
UPDATE pipeline SET name = ?, description = ?, variable_declarations = ?, version = ?, version_fork_strategy = ?, fixed_version_id = ?, fixed_version_label = ?, updated_at = ? WHERE id = ?;

-- name: DeletePipeline :exec
DELETE FROM pipeline WHERE id = ?;

-- name: PipelineStages :many
SELECT id, pipeline_id, name, image, script, artifacts, depends_on, sort_order, description, created_at, updated_at
FROM pipeline_stage WHERE pipeline_id = ? ORDER BY sort_order, id;

-- name: PipelineStageByID :one
SELECT id, pipeline_id, name, image, script, artifacts, depends_on, sort_order, description, created_at, updated_at
FROM pipeline_stage WHERE id = ?;

-- name: DeletePipelineStages :exec
DELETE FROM pipeline_stage WHERE pipeline_id = ?;

-- name: InsertPipelineStage :exec
INSERT INTO pipeline_stage (id, pipeline_id, name, image, script, artifacts, depends_on, sort_order, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipelineStage :exec
UPDATE pipeline_stage SET name = ?, image = ?, script = ?, artifacts = ?, depends_on = ?, sort_order = ?, description = ?, updated_at = ? WHERE id = ?;

-- name: DeletePipelineStage :exec
DELETE FROM pipeline_stage WHERE id = ?;

-- name: PipelineSnapshotByID :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot WHERE id = ?;

-- name: LatestPipelineSnapshot :one
SELECT id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot WHERE pipeline_id = ? ORDER BY pipeline_version DESC, created_at DESC LIMIT 1;

-- name: InsertPipelineSnapshot :exec
INSERT INTO pipeline_snapshot (id, project_id, pipeline_id, pipeline_name, pipeline_version, source_pipeline_id, source_template_name, source_template_version, application_id, application_name, repository_id, repository_name, version_fork_strategy, fixed_version_id, fixed_version_label, stages_snapshot, variables_snapshot, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
