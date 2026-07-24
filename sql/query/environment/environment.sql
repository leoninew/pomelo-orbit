-- name: CountEnvironments :one
SELECT COUNT(*)
FROM environment
WHERE project_id = ?
  AND (? = '' OR code LIKE ? OR name LIKE ?);

-- name: ListEnvironments :many
SELECT id, project_id, code, name, description, created_at, updated_at
FROM environment
WHERE project_id = ?
  AND (? = '' OR code LIKE ? OR name LIKE ?)
ORDER BY code, id
LIMIT ? OFFSET ?;

-- name: EnvironmentByID :one
SELECT id, project_id, code, name, description, created_at, updated_at
FROM environment
WHERE id = ?;

-- name: EnvironmentByProjectCode :one
SELECT id, project_id, code, name, description, created_at, updated_at
FROM environment
WHERE project_id = ? AND code = ?;

-- name: CreateEnvironment :exec
INSERT INTO environment (id, project_id, code, name, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEnvironment :exec
UPDATE environment
SET name = ?, description = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteEnvironment :exec
DELETE FROM environment
WHERE id = ?;

-- name: CountServicesByEnvironment :one
SELECT COUNT(*)
FROM service
WHERE environment_id = ?;
