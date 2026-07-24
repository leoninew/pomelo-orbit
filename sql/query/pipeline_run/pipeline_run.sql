-- name: PipelineRunByID :one
SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
       template_name, template_version, "trigger", trigger_ref, variables_snapshot, status, retry_of,
       started_at, finished_at, error_message, created_at
FROM pipeline_run
WHERE id = ?;

-- name: CountPipelineRuns :one
SELECT COUNT(*)
FROM pipeline_run
WHERE project_id = sqlc.arg(project_id)
  AND (
    sqlc.arg(repository_filter) = ''
    OR repository_id = sqlc.arg(repository_id)
  )
  AND (
    sqlc.arg(template_filter) = ''
    OR template_id = sqlc.arg(template_id)
  )
  AND (sqlc.narg(date_from) IS NULL OR created_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to) IS NULL OR created_at < sqlc.narg(date_to));

-- name: ListPipelineRuns :many
SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
       template_name, template_version, "trigger", trigger_ref, variables_snapshot, status, retry_of,
       started_at, finished_at, error_message, created_at
FROM pipeline_run
WHERE project_id = sqlc.arg(project_id)
  AND (
    sqlc.arg(repository_filter) = ''
    OR repository_id = sqlc.arg(repository_id)
  )
  AND (
    sqlc.arg(template_filter) = ''
    OR template_id = sqlc.arg(template_id)
  )
  AND (sqlc.narg(date_from) IS NULL OR created_at >= sqlc.narg(date_from))
  AND (sqlc.narg(date_to) IS NULL OR created_at < sqlc.narg(date_to))
ORDER BY created_at DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CountPipelineRunsByRepository :one
SELECT COUNT(*)
FROM pipeline_run
WHERE repository_id = ?;

-- name: ListPipelineRunsByRepository :many
SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
       template_name, template_version, "trigger", trigger_ref, variables_snapshot, status, retry_of,
       started_at, finished_at, error_message, created_at
FROM pipeline_run
WHERE repository_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CreatePipelineRun :exec
INSERT INTO pipeline_run (
  id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version,
  "trigger", trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, NULL, NULL, ?);

-- name: CancelPipelineRun :exec
UPDATE pipeline_run
SET status = ?, finished_at = ?
WHERE id = ?;

-- name: MarkPipelineRunRunning :exec
UPDATE pipeline_run
SET status = ?, started_at = ?, error_message = NULL
WHERE id = ?;

-- name: CompletePipelineRun :exec
UPDATE pipeline_run
SET status = ?, finished_at = ?, error_message = NULLIF(?, '')
WHERE id = ?;
