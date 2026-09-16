-- name: ApplicationById :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: ApplicationByName :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE name = sqlc.arg(name)
  AND project_id = sqlc.arg(project_id);

-- name: ApplicationByProjectAndName :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE project_id = sqlc.arg(project_id)
  AND name = sqlc.arg(name);

-- name: ApplicationByCode :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE code = sqlc.arg(code)
  AND project_id = sqlc.arg(project_id);

-- name: ApplicationByProjectAndCode :one
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE project_id = sqlc.arg(project_id)
  AND code = sqlc.arg(code);

-- name: CountApplications :one
SELECT COUNT(*)
FROM application
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind));

-- name: ListApplications :many
SELECT id, project_id, name, code, kind, created_at, updated_at
FROM application
WHERE project_id = sqlc.arg(project_id)
  AND (CAST(sqlc.narg(search_pattern) AS CHAR) IS NULL OR name LIKE sqlc.narg(search_pattern) OR code LIKE sqlc.narg(search_pattern))
  AND (CAST(sqlc.narg(kind) AS CHAR) IS NULL OR kind = sqlc.narg(kind))
ORDER BY id DESC
LIMIT ? OFFSET ?;

-- name: CreateApplication :exec
INSERT INTO application (id, project_id, name, code, kind, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateApplication :exec
UPDATE application
SET name = sqlc.arg(name), code = sqlc.arg(code), updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);

-- name: DeleteServicesByApplication :exec
DELETE FROM service
WHERE application_id = sqlc.arg(application_id)
  AND EXISTS (
    SELECT 1 FROM application
    WHERE application.id = service.application_id
      AND application.project_id = sqlc.arg(project_id)
  );

-- name: VersionIdsByApplication :many
SELECT id
FROM version
WHERE application_id = sqlc.arg(application_id)
  AND EXISTS (
    SELECT 1 FROM application
    WHERE application.id = version.application_id
      AND application.project_id = sqlc.arg(project_id)
  );

-- name: DeleteVersionsByApplication :exec
DELETE FROM version
WHERE application_id = sqlc.arg(application_id)
  AND EXISTS (
    SELECT 1 FROM application
    WHERE application.id = version.application_id
      AND application.project_id = sqlc.arg(project_id)
  );

-- name: DeleteGatewayConfigByApplication :exec
DELETE FROM gateway_config
WHERE application_id = sqlc.arg(application_id)
  AND EXISTS (
    SELECT 1 FROM application
    WHERE application.id = gateway_config.application_id
      AND application.project_id = sqlc.arg(project_id)
  );

-- name: DeleteApplication :exec
DELETE FROM application
WHERE id = sqlc.arg(id)
  AND project_id = sqlc.arg(project_id);
