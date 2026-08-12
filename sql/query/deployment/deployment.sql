-- name: CreateDeployment :exec
INSERT INTO deployment (
  id, project_id, application_id, application_name, version_id, service_id, options_json, effective_plan_hash,
  operation_type, trigger_type, command_text, status, started_at, is_rollback, rollback_from_deployment_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeploymentByID :one
SELECT d.id, d.project_id, d.application_id, d.application_name, d.version_id, d.service_id,
       s.instance_key AS service_instance_key, d.options_json, d.effective_plan_hash,
       d.operation_type, d.trigger_type, d.command_text, d.status, d.started_at, d.finished_at, d.duration_ms,
       d.log_text, d.error_message, d.is_rollback, d.rollback_from_deployment_id
FROM deployment d
LEFT JOIN service s ON s.id = d.service_id
WHERE d.id = ?;

-- name: CountDeployments :one
SELECT COUNT(*)
FROM deployment
WHERE project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (
    CAST(sqlc.narg(application_id) AS TEXT) IS NULL
    OR application_id = CAST(sqlc.narg(application_id) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(status) AS TEXT) IS NULL
    OR status = CAST(sqlc.narg(status) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(application_name_pattern) AS TEXT) IS NULL
    OR application_name LIKE CAST(sqlc.narg(application_name_pattern) AS TEXT)
  )
  AND (CAST(sqlc.narg(date_from) AS DATETIME) IS NULL OR started_at >= sqlc.narg(date_from))
  AND (CAST(sqlc.narg(date_to) AS DATETIME) IS NULL OR started_at < sqlc.narg(date_to));

-- name: ListDeployments :many
SELECT d.id, d.project_id, d.application_id, d.application_name, d.version_id, d.service_id,
       s.instance_key AS service_instance_key, d.options_json, d.effective_plan_hash,
       d.operation_type, d.trigger_type, d.command_text, d.status, d.started_at, d.finished_at, d.duration_ms,
       d.log_text, d.error_message, d.is_rollback, d.rollback_from_deployment_id
FROM deployment d
LEFT JOIN service s ON s.id = d.service_id
WHERE d.project_id = CAST(sqlc.arg(project_id) AS TEXT)
  AND (
    CAST(sqlc.narg(application_id) AS TEXT) IS NULL
    OR d.application_id = CAST(sqlc.narg(application_id) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(status) AS TEXT) IS NULL
    OR d.status = CAST(sqlc.narg(status) AS TEXT)
  )
  AND (
    CAST(sqlc.narg(application_name_pattern) AS TEXT) IS NULL
    OR d.application_name LIKE CAST(sqlc.narg(application_name_pattern) AS TEXT)
  )
  AND (CAST(sqlc.narg(date_from) AS DATETIME) IS NULL OR d.started_at >= sqlc.narg(date_from))
  AND (CAST(sqlc.narg(date_to) AS DATETIME) IS NULL OR d.started_at < sqlc.narg(date_to))
ORDER BY d.id DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CancelDeployment :execrows
UPDATE deployment
SET status = ?, finished_at = ?, duration_ms = ?, error_message = ?
WHERE id = ? AND status IN (?, ?);

-- name: LatestSuccessfulDeploymentPlanHash :one
SELECT effective_plan_hash
FROM deployment
WHERE service_id = ?
  AND status = 'ran_to_completion'
  AND effective_plan_hash IS NOT NULL
ORDER BY finished_at DESC, id DESC
LIMIT 1;

-- name: BeginDeployment :execrows
UPDATE deployment
SET status = ?, started_at = ?, error_message = NULL
WHERE id = ? AND status = ?;

-- name: CompleteDeployment :execrows
UPDATE deployment
SET status = sqlc.arg(status),
    finished_at = sqlc.arg(finished_at),
    duration_ms = sqlc.arg(duration_ms),
    error_message = NULLIF(CAST(sqlc.arg(error_message) AS TEXT), '')
WHERE id = sqlc.arg(id) AND status = sqlc.arg(current_status);

-- name: CountActiveDeploymentsByService :one
SELECT COUNT(*)
FROM deployment
WHERE service_id = ? AND status IN (?, ?);

-- name: DeploymentStartedAt :one
SELECT started_at
FROM deployment
WHERE id = ?;
