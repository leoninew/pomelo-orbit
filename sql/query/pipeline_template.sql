-- name: PipelineTemplateByID :one
SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
FROM pipeline_template
WHERE id = ?;

-- name: PipelineTemplateStages :many
SELECT id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
FROM pipeline_template_stage
WHERE template_id = ?
ORDER BY sort_order, id;

-- name: PipelineTemplateReferencedByWebhooks :one
SELECT COUNT(*)
FROM repository_webhook
WHERE template_id = ?;

-- name: DeletePipelineTemplate :exec
DELETE FROM pipeline_template
WHERE id = ?;
