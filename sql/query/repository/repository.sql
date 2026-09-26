-- name: RepositoryById :one
SELECT id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE id = sqlc.arg(id)
  AND project_id IS NULL;

-- name: RepositoryByCode :one
SELECT id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE code = sqlc.arg(code)
  AND project_id IS NULL;

-- name: CountRepositories :one
SELECT COUNT(*)
FROM repository
WHERE project_id IS NULL
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR code LIKE sqlc.narg(search_pattern)
    OR repository_url LIKE sqlc.narg(search_pattern)
  );

-- name: ListRepositories :many
SELECT id, name, code, repository_type, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE project_id IS NULL
  AND (
    CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR code LIKE sqlc.narg(search_pattern)
    OR repository_url LIKE sqlc.narg(search_pattern)
  )
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateRepository :exec
INSERT INTO repository (id, name, code, repository_type, repository_url, git_credential_id, variable_overrides, default_branch, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRepository :exec
UPDATE repository
SET name = sqlc.arg(name),
    repository_type = sqlc.arg(repository_type),
    repository_url = sqlc.arg(repository_url),
    git_credential_id = sqlc.arg(git_credential_id),
    variable_overrides = sqlc.arg(variable_overrides),
    default_branch = sqlc.arg(default_branch),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id IS NULL;

-- name: CountPipelinesByRepository :one
SELECT COUNT(*)
FROM pipeline
WHERE kind = 'application'
  AND repository_id = sqlc.arg(repository_id);

-- name: DeleteRepository :exec
DELETE FROM repository
WHERE id = sqlc.arg(id)
  AND project_id IS NULL;

-- name: RepositoryReferencesCredential :one
SELECT COUNT(*)
FROM repository
WHERE project_id IS NULL
  AND git_credential_id = sqlc.arg(git_credential_id);
