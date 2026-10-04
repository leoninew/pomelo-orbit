-- name: RepositoryCredentialById :one
SELECT id, name, type, encrypted_data, revision, created_at, updated_at
FROM repository_credential
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');

-- name: RepositoryCredentialByName :one
SELECT id, name, type, encrypted_data, revision, created_at, updated_at
FROM repository_credential
WHERE project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token')
  AND name = sqlc.arg(name);

-- name: RepositoryCredentialExists :one
SELECT COUNT(*)
FROM repository_credential
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');

-- name: RepositoryCredentialName :one
SELECT name
FROM repository_credential
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');

-- name: CountRepositoryCredentials :one
SELECT COUNT(*)
FROM repository_credential
WHERE project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token')
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR type LIKE sqlc.narg(search_pattern));

-- name: ListRepositoryCredentials :many
SELECT id, name, type, encrypted_data, revision, created_at, updated_at
FROM repository_credential
WHERE project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token')
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR type LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateRepositoryCredential :exec
INSERT INTO repository_credential (id, name, type, encrypted_data, revision, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRepositoryCredential :exec
UPDATE repository_credential
SET name = sqlc.arg(name), encrypted_data = sqlc.arg(encrypted_data), revision = sqlc.arg(revision),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');

-- name: DeleteRepositoryCredential :exec
DELETE FROM repository_credential
WHERE id = sqlc.arg(id)
  AND project_id IS NULL
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');
