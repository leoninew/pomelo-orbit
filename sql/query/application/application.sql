-- name: ApplicationByID :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE id = ?;

-- name: ApplicationByName :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE name = ?;

-- name: ApplicationByCode :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE code = ?;

-- name: CountApplications :one
SELECT COUNT(*)
FROM application
WHERE (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
  AND (sqlc.narg(search_pattern) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (sqlc.narg(kind) IS NULL OR kind = sqlc.narg(kind));

-- name: ListApplications :many
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
  AND (sqlc.narg(search_pattern) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (sqlc.narg(kind) IS NULL OR kind = sqlc.narg(kind))
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
