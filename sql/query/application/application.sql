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
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT) OR code LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
  AND (CAST(sqlc.narg(kind) AS TEXT) IS NULL OR kind = CAST(sqlc.narg(kind) AS TEXT));

-- name: ListApplications :many
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT) OR code LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
  AND (CAST(sqlc.narg(kind) AS TEXT) IS NULL OR kind = CAST(sqlc.narg(kind) AS TEXT))
ORDER BY id DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

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
