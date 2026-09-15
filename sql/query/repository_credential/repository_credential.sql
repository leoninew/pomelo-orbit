-- name: RepositoryCredentialById :one
SELECT id, project_id, name, type, encrypted_data, revision, created_at
FROM repository_credential
WHERE id = ?;

-- name: RepositoryCredentialByName :one
SELECT id, project_id, name, type, encrypted_data, revision, created_at
FROM repository_credential
WHERE project_id = ? AND name = ?;

-- name: RepositoryCredentialExists :one
SELECT COUNT(*)
FROM repository_credential
WHERE id = ?;

-- name: RepositoryCredentialName :one
SELECT name
FROM repository_credential
WHERE id = ?;

-- name: CountRepositoryCredentials :one
SELECT COUNT(*)
FROM repository_credential
WHERE project_id = sqlc.arg(project_id)
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token')
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR type LIKE sqlc.narg(search_pattern));

-- name: ListRepositoryCredentials :many
SELECT id, project_id, name, type, encrypted_data, revision, created_at
FROM repository_credential
WHERE project_id = sqlc.arg(project_id)
  AND type IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token')
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL
    OR name LIKE sqlc.narg(search_pattern)
    OR type LIKE sqlc.narg(search_pattern))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateRepositoryCredential :exec
INSERT INTO repository_credential (id, project_id, name, type, encrypted_data, revision, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRepositoryCredential :exec
UPDATE repository_credential
SET name = ?, encrypted_data = ?, revision = ?
WHERE id = ?;

-- name: DeleteRepositoryCredential :exec
DELETE FROM repository_credential
WHERE id = ?;

-- name: RepositoryCredentialReferencedByRepositories :one
SELECT COUNT(*)
FROM repository
WHERE project_id = ? AND git_credential_id = ?;
