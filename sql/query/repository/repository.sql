-- name: RepositoryByID :one
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE id = ?;

-- name: RepositoryByCode :one
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE code = CAST(sqlc.arg(code) AS TEXT)
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT));

-- name: CountRepositories :one
SELECT COUNT(*)
FROM repository
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (
    CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR repository_url LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
  );

-- name: ListRepositories :many
SELECT id, project_id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (
    CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR code LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR repository_url LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
  )
ORDER BY created_at DESC, id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

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
