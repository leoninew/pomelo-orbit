-- name: PipelineTemplateByID :one
SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline_template
WHERE id = ?;

-- name: PipelineTemplateByName :one
SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline_template
WHERE project_id = ? AND name = ?;

-- name: CountPipelineTemplates :one
SELECT COUNT(*)
FROM pipeline_template
WHERE project_id = ?
  AND (? = '' OR name LIKE ?);

-- name: ListPipelineTemplates :many
SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline_template
WHERE project_id = ?
  AND (? = '' OR name LIKE ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: PipelineTemplateStages :many
SELECT id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
FROM pipeline_template_stage
WHERE template_id = ?
ORDER BY sort_order, id;

-- name: CreatePipelineTemplate :exec
INSERT INTO pipeline_template (id, project_id, name, description, variable_declarations, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipelineTemplate :exec
UPDATE pipeline_template
SET name = ?, description = ?, variable_declarations = ?, version = ?, updated_at = ?
WHERE id = ?;

-- name: DeletePipelineTemplateStages :exec
DELETE FROM pipeline_template_stage
WHERE template_id = ?;

-- name: InsertPipelineTemplateStage :exec
INSERT INTO pipeline_template_stage (id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: PipelineTemplateReferencedByWebhooks :one
SELECT COUNT(*)
FROM repository_webhook
WHERE template_id = ?;

-- name: DeletePipelineTemplate :exec
DELETE FROM pipeline_template
WHERE id = ?;
