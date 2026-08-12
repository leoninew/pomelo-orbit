-- name: CredentialByID :one
SELECT id, project_id, name, type, encrypted_data, created_at
FROM credential
WHERE id = ?;

-- name: CredentialByName :one
SELECT id, project_id, name, type, encrypted_data, created_at
FROM credential
WHERE project_id = ? AND name = ?;

-- name: CredentialExists :one
SELECT COUNT(*)
FROM credential
WHERE id = ?;

-- name: CredentialName :one
SELECT name
FROM credential
WHERE id = ?;

-- name: CountCredentials :one
SELECT COUNT(*)
FROM credential
WHERE project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR type LIKE CAST(sqlc.narg(search_pattern) AS TEXT));

-- name: ListCredentials :many
SELECT id, project_id, name, type, encrypted_data, created_at
FROM credential
WHERE project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR type LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
ORDER BY id DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CreateCredential :exec
INSERT INTO credential (id, project_id, name, type, encrypted_data, created_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateCredential :exec
UPDATE credential
SET name = ?, encrypted_data = ?
WHERE id = ?;

-- name: DeleteCredential :exec
DELETE FROM credential
WHERE id = ?;

-- name: CredentialReferencedByRepositories :one
SELECT COUNT(*)
FROM repository
WHERE project_id = ? AND git_credential_id = ?;
