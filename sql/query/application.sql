-- name: ApplicationByID :one
SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at
FROM application
WHERE id = ?;

-- name: ApplicationByName :one
SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at
FROM application
WHERE name = ?;

-- name: ApplicationByCode :one
SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at
FROM application
WHERE code = ?;
