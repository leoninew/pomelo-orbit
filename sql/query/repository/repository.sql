-- name: RepositoryByID :one
SELECT id, project_id, name, code, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE id = ?;

-- name: RepositoryByCode :one
SELECT id, project_id, name, code, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE code = sqlc.arg(code)
  AND (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id));

-- name: CountRepositories :one
SELECT COUNT(*)
FROM repository
WHERE (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
  AND (
    sqlc.arg(search) = ''
    OR name LIKE sqlc.arg(pattern)
    OR code LIKE sqlc.arg(pattern)
    OR repository_url LIKE sqlc.arg(pattern)
  );

-- name: ListRepositories :many
SELECT id, project_id, name, code, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
  AND (
    sqlc.arg(search) = ''
    OR name LIKE sqlc.arg(pattern)
    OR code LIKE sqlc.arg(pattern)
    OR repository_url LIKE sqlc.arg(pattern)
  )
ORDER BY created_at DESC, id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CreateRepository :exec
INSERT INTO repository (id, project_id, name, code, repository_url, git_credential_id, variable_overrides, default_branch, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRepository :exec
UPDATE repository
SET name = ?, repository_url = ?, git_credential_id = ?, variable_overrides = ?, default_branch = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRepository :exec
DELETE FROM repository
WHERE id = ?;

-- name: RepositoryHasRunningPipelines :one
SELECT COUNT(*)
FROM pipeline_run
WHERE repository_id = ? AND status IN (?, ?);
