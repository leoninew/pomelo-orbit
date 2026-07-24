-- name: PipelineSnapshotByID :one
SELECT id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot
WHERE id = ?;

-- name: LatestPipelineSnapshot :one
SELECT id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at
FROM pipeline_snapshot
WHERE template_id = ?
ORDER BY version DESC
LIMIT 1;

-- name: CreatePipelineSnapshot :exec
INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);
