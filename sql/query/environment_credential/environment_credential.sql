-- name: EnvironmentCredentialById :one
SELECT id, project_id, public_key, encrypted_private_key, revision, created_at
FROM environment_credential
WHERE id = ?;

-- name: EnvironmentCredentialByProjectLatest :one
SELECT id, project_id, public_key, encrypted_private_key, revision, created_at
FROM environment_credential
WHERE project_id = ?
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: CreateEnvironmentCredential :exec
INSERT INTO environment_credential (
  id, project_id, public_key, encrypted_private_key, revision, created_at
) VALUES (?, ?, ?, ?, ?, ?);
