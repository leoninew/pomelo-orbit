-- name: RoutesByApplication :many
SELECT id, application_id, service_name, domain, port, created_at, updated_at
FROM application_route
WHERE application_id = ?;

-- name: ApplicationRouteByID :one
SELECT id, application_id, service_name, domain, port, created_at, updated_at
FROM application_route
WHERE id = ?;

-- name: DeleteApplicationRoute :exec
DELETE FROM application_route
WHERE id = ?;
