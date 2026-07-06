-- name: CredentialByID :one
SELECT id, project_id, name, type, encrypted_data, created_at
FROM credential
WHERE id = ?;

-- name: CredentialExists :one
SELECT COUNT(*)
FROM credential
WHERE id = ?;

-- name: CredentialName :one
SELECT name
FROM credential
WHERE id = ?;

-- name: DeleteCredential :exec
DELETE FROM credential
WHERE id = ?;
