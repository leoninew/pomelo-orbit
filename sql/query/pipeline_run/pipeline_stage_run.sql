-- name: ListPipelineStageRuns :many
SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message
FROM pipeline_stage_run
WHERE pipeline_run_id = ?
ORDER BY rowid;

-- name: PipelineStageRunByID :one
SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message
FROM pipeline_stage_run
WHERE id = ?;

-- name: InsertPipelineStageRun :exec
INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePipelineStageRun :exec
UPDATE pipeline_stage_run
SET status = ?, started_at = ?, finished_at = ?, exit_code = ?, error_message = ?
WHERE id = ?;
