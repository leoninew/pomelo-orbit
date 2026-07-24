-- name: CountArtifacts :one
SELECT COUNT(*)
FROM artifact
WHERE project_id = ?
  AND (? = '' OR repository_id = ?)
  AND (? = '' OR template_id = ?)
  AND (? = '' OR name LIKE ? OR path LIKE ?);

-- name: ListArtifacts :many
SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
FROM artifact
WHERE project_id = ?
  AND (? = '' OR repository_id = ?)
  AND (? = '' OR template_id = ?)
  AND (? = '' OR name LIKE ? OR path LIKE ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListArtifactsByRun :many
SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
FROM artifact
WHERE pipeline_run_id = ?
  AND (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
ORDER BY created_at, id;

-- name: InsertArtifact :exec
INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
