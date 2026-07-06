-- name: ServiceConfigsByApplication :many
SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at
FROM application_service
WHERE application_id = ?;

-- name: ApplicationServiceConfigByName :one
SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at
FROM application_service
WHERE application_id = ? AND service_name = ?;

-- name: ApplicationServiceConfigExists :one
SELECT COUNT(*)
FROM application_service
WHERE id = ?;
