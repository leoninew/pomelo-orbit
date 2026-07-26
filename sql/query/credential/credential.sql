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
WHERE project_id = ?
  AND (? = '' OR name LIKE ? OR type LIKE ?);

-- name: ListCredentials :many
SELECT id, project_id, name, type, encrypted_data, created_at
FROM credential
WHERE project_id = ?
  AND (? = '' OR name LIKE ? OR type LIKE ?)
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

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

-- name: CredentialReferencedByVersionComponents :one
SELECT COUNT(*)
FROM version_component_secret_env_ref
WHERE credential_id = ?;

-- name: VersionComponentSecretEnvRefsByCredential :many
SELECT component_id, env_key, credential_id, data_key
FROM version_component_secret_env_ref
WHERE credential_id = ?
ORDER BY component_id, env_key;
