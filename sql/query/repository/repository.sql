-- name: RepositoryByID :one
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE id = ?;

-- name: RepositoryByCode :one
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE code = sqlc.arg(code)
  AND (CAST(sqlc.narg(project_id) AS CHAR) IS NULL OR project_id = sqlc.narg(project_id));

-- name: CountRepositories :one
SELECT COUNT(*)
FROM repository
WHERE (CAST(sqlc.narg(project_id) AS CHAR) IS NULL OR project_id = sqlc.narg(project_id))
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR code LIKE sqlc.narg(search_pattern)
    OR repository_url LIKE sqlc.narg(search_pattern)
  );

-- name: ListRepositories :many
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE (CAST(sqlc.narg(project_id) AS CHAR) IS NULL OR project_id = sqlc.narg(project_id))
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR code LIKE sqlc.narg(search_pattern)
    OR repository_url LIKE sqlc.narg(search_pattern)
  )
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateRepository :exec
INSERT INTO repository (id, project_id, name, code, repository_type, repository_url, git_credential_id, variable_overrides, default_branch, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRepository :exec
UPDATE repository
SET name = ?, repository_type = ?, repository_url = ?, git_credential_id = ?, variable_overrides = ?, default_branch = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteRepository :exec
DELETE FROM repository
WHERE id = ?;

-- name: RepositoryHasRunningPipelines :one
SELECT COUNT(*)
FROM pipeline_run
WHERE repository_id = ? AND status IN (?, ?);
