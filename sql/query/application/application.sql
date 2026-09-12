-- name: ApplicationByID :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE id = ?;

-- name: ApplicationByName :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE name = ?;

-- name: ApplicationByProjectAndName :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE project_id = ? AND name = ?;

-- name: ApplicationByCode :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE code = ?;

-- name: ApplicationByProjectAndCode :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE project_id = ? AND code = ?;

-- name: CountApplications :one
SELECT COUNT(*)
FROM application
WHERE (CAST(sqlc.narg(project_id) AS CHAR) IS NULL OR project_id = sqlc.narg(project_id))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind));

-- name: ListApplications :many
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE (CAST(sqlc.narg(project_id) AS CHAR) IS NULL OR project_id = sqlc.narg(project_id))
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateApplication :exec
INSERT INTO application (id, project_id, name, code, kind, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateApplication :exec
UPDATE application
SET name = ?, code = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteServicesByApplication :exec
DELETE FROM service
WHERE application_id = ?;

-- name: VersionIdsByApplication :many
SELECT id
FROM version
WHERE application_id = ?;

-- name: DeleteVersionsByApplication :exec
DELETE FROM version
WHERE application_id = ?;

-- name: DeleteGatewayConfigByApplication :exec
DELETE FROM gateway_config
WHERE application_id = ?;

-- name: DeleteApplication :exec
DELETE FROM application
WHERE id = ?;
