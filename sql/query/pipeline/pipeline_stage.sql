-- name: CountPipelineStages :one
SELECT COUNT(*)
FROM pipeline_stage
WHERE project_id = ?
  AND (? = '' OR name LIKE ?);

-- name: ListPipelineStages :many
SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
FROM pipeline_stage
WHERE project_id = ?
  AND (? = '' OR name LIKE ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: PipelineStageByID :one
SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
FROM pipeline_stage
WHERE id = ?;

-- name: PipelineStageByName :one
SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
FROM pipeline_stage
WHERE project_id = ? AND name = ?;

-- name: PipelineStagesByIds :many
SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
FROM pipeline_stage
WHERE project_id = ? AND id IN (sqlc.slice('ids'));

-- name: CreatePipelineStage :exec
INSERT INTO pipeline_stage (id, project_id, name, image, script, artifacts, description, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipelineStage :exec
UPDATE pipeline_stage
SET name = ?, image = ?, script = ?, artifacts = ?, description = ?, version = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePipelineStage :exec
DELETE FROM pipeline_stage
WHERE id = ?;

-- name: PipelineStageReferencedByTemplates :one
SELECT COUNT(*)
FROM pipeline_template_stage
JOIN pipeline_template ON pipeline_template.id = pipeline_template_stage.template_id
WHERE pipeline_template.project_id = ? AND pipeline_template_stage.stage_id = ?;
