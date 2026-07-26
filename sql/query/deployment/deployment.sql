-- name: CreateDeployment :exec
INSERT INTO deployment (
  id, project_id, application_id, application_name, version_id, service_id, options_json,
  operation_type, trigger_type, command_text, status, started_at, is_rollback, rollback_from_deployment_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeploymentByID :one
SELECT id, project_id, application_id, application_name, version_id, service_id, options_json,
       operation_type, trigger_type, command_text, status, started_at, finished_at, duration_ms, log_text,
       error_message, is_rollback, rollback_from_deployment_id
FROM deployment
WHERE id = ?;

-- name: CountDeployments :one
SELECT COUNT(*)
FROM deployment
WHERE project_id = sqlc.arg(project_id)
  AND (
    sqlc.arg(application_filter) = ''
    OR application_id = sqlc.arg(application_id)
  )
  AND (
    sqlc.arg(status_filter) = ''
    OR status = sqlc.arg(status)
  )
  AND (
    sqlc.arg(application_name_filter) = ''
    OR application_name LIKE sqlc.arg(application_name)
  )
  AND (sqlc.narg(date_from) IS NULL OR started_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to) IS NULL OR started_at < sqlc.narg(date_to));

-- name: ListDeployments :many
SELECT id, project_id, application_id, application_name, version_id, service_id, options_json,
       operation_type, trigger_type, command_text, status, started_at, finished_at, duration_ms, log_text,
       error_message, is_rollback, rollback_from_deployment_id
FROM deployment
WHERE project_id = sqlc.arg(project_id)
  AND (
    sqlc.arg(application_filter) = ''
    OR application_id = sqlc.arg(application_id)
  )
  AND (
    sqlc.arg(status_filter) = ''
    OR status = sqlc.arg(status)
  )
  AND (
    sqlc.arg(application_name_filter) = ''
    OR application_name LIKE sqlc.arg(application_name)
  )
  AND (sqlc.narg(date_from) IS NULL OR started_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to) IS NULL OR started_at < sqlc.narg(date_to))
ORDER BY started_at DESC, id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CancelDeployment :exec
UPDATE deployment
SET status = ?, finished_at = ?, duration_ms = ?, error_message = ?
WHERE id = ?;

-- name: MarkDeploymentRunning :exec
UPDATE deployment
SET status = ?, started_at = ?, error_message = NULL
WHERE id = ?;

-- name: CompleteDeployment :exec
UPDATE deployment
SET status = ?, finished_at = ?, duration_ms = ?, error_message = NULLIF(?, '')
WHERE id = ?;

-- name: DeploymentStartedAt :one
SELECT started_at
FROM deployment
WHERE id = ?;
