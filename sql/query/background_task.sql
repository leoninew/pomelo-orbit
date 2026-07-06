-- name: EnqueueTask :exec
INSERT INTO background_task (id, task_type, payload_json, status, attempts, max_attempts)
VALUES (?, ?, ?, ?, 0, ?);

-- name: TaskToClaim :one
SELECT id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
       started_at, finished_at, error_message, created_at, updated_at
FROM background_task
WHERE status = ?
   OR (status = ? AND locked_at IS NOT NULL AND locked_at < ?)
ORDER BY created_at ASC
LIMIT 1;

-- name: FindTaskByID :one
SELECT id, task_type, payload_json, status, attempts, max_attempts, locked_by, locked_at,
       started_at, finished_at, error_message, created_at, updated_at
FROM background_task
WHERE id = ?;
