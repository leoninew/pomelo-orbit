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
  AND (sqlc.arg(search) = '' OR name LIKE sqlc.arg(name_pattern) OR code LIKE sqlc.arg(name_pattern))
  AND (sqlc.arg(kind_filter) = '' OR kind = sqlc.arg(kind));

-- name: ListApplications :many
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE (sqlc.narg(project_id) IS NULL OR project_id = sqlc.narg(project_id))
  AND (sqlc.arg(search) = '' OR name LIKE sqlc.arg(name_pattern) OR code LIKE sqlc.arg(name_pattern))
  AND (sqlc.arg(kind_filter) = '' OR kind = sqlc.arg(kind))
ORDER BY created_at DESC, id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CreateApplication :exec
INSERT INTO application (id, project_id, name, code, kind, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateApplication :exec
UPDATE application
SET name = ?, code = ?, updated_at = ?
WHERE id = ?;

-- name: DetachDeploymentServiceRefsByApplication :exec
UPDATE deployment
SET service_id = NULL
WHERE service_id IN (SELECT id FROM service WHERE service.application_id = ?);

-- name: DetachDeploymentVersionRefsByApplication :exec
UPDATE deployment
SET version_id = NULL
WHERE version_id IN (SELECT id FROM version WHERE version.application_id = ?);

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

-- name: DeleteApplication :exec
DELETE FROM application
WHERE id = ?;
