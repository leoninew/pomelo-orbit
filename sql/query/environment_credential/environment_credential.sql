-- name: EnvironmentCredentialByEnvironment :one
SELECT id, environment_id, public_key, encrypted_private_key, revision, created_at
FROM environment_credential
WHERE environment_id = ?;

-- name: CreateEnvironmentCredential :exec
INSERT INTO environment_credential (
  id, environment_id, public_key, encrypted_private_key, revision, created_at
) VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteEnvironmentCredential :exec
DELETE FROM environment_credential
WHERE environment_id = ?;
