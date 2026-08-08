-- name: EnqueueTask :exec
INSERT INTO background_task (id, task_type, payload_json, status, attempts, max_attempts, created_at, updated_at)
VALUES (?, ?, ?, ?, 0, ?, ?, ?);

-- name: TaskToClaim :one
SELECT id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
       started_at, finished_at, error_message, created_at, updated_at
FROM background_task
WHERE status = ?
   OR (status = ? AND locked_at IS NOT NULL AND locked_at < ?)
ORDER BY created_at ASC
LIMIT 1;

-- name: ClaimTask :execrows
UPDATE background_task
SET status = ?, attempts = attempts + 1, locked_by = ?, locked_at = ?,
    started_at = COALESCE(started_at, ?), updated_at = ?, error_message = NULL
WHERE id = ?
  AND (status = ? OR (status = ? AND locked_at IS NOT NULL AND locked_at < ?));

-- name: CompleteTask :exec
UPDATE background_task
SET status = ?, locked_by = NULL, locked_at = NULL, finished_at = ?,
    updated_at = ?, error_message = NULL
WHERE id = ? AND status = ?;

-- name: FailTask :exec
UPDATE background_task
SET status = CASE WHEN attempts >= max_attempts THEN ? ELSE ? END,
    locked_by = NULL, locked_at = NULL,
    finished_at = CASE WHEN attempts >= max_attempts THEN ? ELSE finished_at END,
    updated_at = ?, error_message = ?
WHERE id = ? AND status = ?;

-- name: FindTaskByID :one
SELECT id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
       started_at, finished_at, error_message, created_at, updated_at
FROM background_task
WHERE id = ?;
