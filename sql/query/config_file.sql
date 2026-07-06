-- name: ConfigFilesByApplication :many
SELECT id, application_id, path, content, created_at, updated_at
FROM application_config_file
WHERE application_id = ?
ORDER BY path;

-- name: ConfigFileByID :one
SELECT id, application_id, path, content, created_at, updated_at
FROM application_config_file
WHERE id = ?;

-- name: DeleteConfigFile :exec
DELETE FROM application_config_file
WHERE id = ?;
