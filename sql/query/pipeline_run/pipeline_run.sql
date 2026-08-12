-- name: ListPipelineRuns :many
SELECT id, project_id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name, pipeline_version, trigger, trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at
FROM pipeline_run
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(repository_id) AS TEXT) IS NULL OR repository_id = CAST(sqlc.narg(repository_id) AS TEXT))
  AND (CAST(sqlc.narg(pipeline_id) AS TEXT) IS NULL OR pipeline_id = CAST(sqlc.narg(pipeline_id) AS TEXT))
  AND (CAST(sqlc.narg(from_at) AS DATETIME) IS NULL OR created_at >= sqlc.narg(from_at))
  AND (CAST(sqlc.narg(to_at) AS DATETIME) IS NULL OR created_at <= sqlc.narg(to_at))
ORDER BY id DESC LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CountPipelineRuns :one
SELECT COUNT(*) FROM pipeline_run
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(repository_id) AS TEXT) IS NULL OR repository_id = CAST(sqlc.narg(repository_id) AS TEXT))
  AND (CAST(sqlc.narg(pipeline_id) AS TEXT) IS NULL OR pipeline_id = CAST(sqlc.narg(pipeline_id) AS TEXT))
  AND (CAST(sqlc.narg(from_at) AS DATETIME) IS NULL OR created_at >= sqlc.narg(from_at))
  AND (CAST(sqlc.narg(to_at) AS DATETIME) IS NULL OR created_at <= sqlc.narg(to_at));

-- name: PipelineRunByID :one
SELECT id, project_id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name, pipeline_version, trigger, trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at
FROM pipeline_run WHERE id = ?;

-- name: ListPipelineStageRuns :many
SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message
FROM pipeline_stage_run WHERE pipeline_run_id = ? ORDER BY started_at, id;

-- name: PipelineStageRunByID :one
SELECT id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message
FROM pipeline_stage_run WHERE id = ?;

-- name: InsertPipelineRun :exec
INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name, pipeline_version, trigger, trigger_ref, variables_snapshot, status, retry_of, started_at, finished_at, error_message, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertPipelineRunVersionBinding :exec
INSERT INTO pipeline_run_version_binding (pipeline_run_id, application_id, application_name, source_version_id, source_version_label, generated_version_id, generated_version_label)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: PipelineRunVersionBindingByRunID :one
SELECT pipeline_run_id, application_id, application_name, source_version_id, source_version_label, generated_version_id, generated_version_label
FROM pipeline_run_version_binding WHERE pipeline_run_id = ?;

-- name: CountActivePipelineRunsByRepository :one
SELECT COUNT(*) FROM pipeline_run
WHERE repository_id = ? AND status IN (?, ?);

-- name: CancelPipelineRun :execrows
UPDATE pipeline_run
SET status = ?, finished_at = ?, error_message = ?
WHERE id = ? AND status IN (?, ?);

-- name: BeginPipelineRun :execrows
UPDATE pipeline_run SET status = ?, started_at = ?
WHERE id = ? AND status = ?;

-- name: CompletePipelineRun :execrows
UPDATE pipeline_run SET status = ?, error_message = ?, finished_at = ?
WHERE id = ? AND status = ?;

-- name: InsertPipelineStageRun :exec
INSERT INTO pipeline_stage_run (id, pipeline_run_id, stage_id, stage_name, status, started_at, finished_at, exit_code, error_message)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: BeginPipelineStageRun :execrows
UPDATE pipeline_stage_run
SET status = ?, started_at = ?, error_message = NULL
WHERE pipeline_stage_run.id = ?
  AND pipeline_stage_run.status = ?
  AND EXISTS (
    SELECT 1
    FROM pipeline_run
    WHERE pipeline_run.id = pipeline_stage_run.pipeline_run_id
      AND pipeline_run.status = ?
  );

-- name: CompletePipelineStageRun :execrows
UPDATE pipeline_stage_run
SET status = ?, finished_at = ?, exit_code = ?, error_message = ?
WHERE id = ? AND status = ?;

-- name: CancelRunningPipelineStageRuns :execrows
UPDATE pipeline_stage_run
SET status = ?, finished_at = ?, error_message = ?
WHERE pipeline_run_id = ? AND status = ?;

-- name: InsertArtifact :exec
INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, pipeline_id, pipeline_name, pipeline_stage_id, stage_name, collector, name, location, value, value_format, image_ref, local_image_sha256, source_artifact_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CommandArtifactByRunStageAndName :one
SELECT id, value, value_format FROM artifact
WHERE pipeline_run_id = ? AND pipeline_stage_id = ? AND name = ? AND collector = 'command';

-- name: CompletePipelineRunVersionBinding :exec
UPDATE pipeline_run_version_binding SET generated_version_id = ?, generated_version_label = ? WHERE pipeline_run_id = ?;

-- name: ListArtifacts :many
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name, artifact.pipeline_id, artifact.pipeline_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location, artifact.value, artifact.value_format, artifact.image_ref, artifact.local_image_sha256, artifact.source_artifact_id, source_artifact.value AS source_commit_sha, binding.application_id, binding.application_name, binding.source_version_id, binding.source_version_label, binding.generated_version_id, binding.generated_version_label, version_component.id AS version_component_id, version_component.name AS version_component_name, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact ON source_artifact.id = artifact.source_artifact_id AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_version_binding AS binding ON binding.pipeline_run_id = artifact.pipeline_run_id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR artifact.project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(repository_id) AS TEXT) IS NULL OR artifact.repository_id = CAST(sqlc.narg(repository_id) AS TEXT))
  AND (CAST(sqlc.narg(pipeline_id) AS TEXT) IS NULL OR artifact.pipeline_id = CAST(sqlc.narg(pipeline_id) AS TEXT))
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL OR artifact.name LIKE CAST(sqlc.narg(search_pattern) AS TEXT) OR artifact.stage_name LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
ORDER BY artifact.id DESC LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: CountArtifacts :one
SELECT COUNT(*) FROM artifact
WHERE (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR project_id = CAST(sqlc.narg(project_id) AS TEXT))
  AND (CAST(sqlc.narg(repository_id) AS TEXT) IS NULL OR repository_id = CAST(sqlc.narg(repository_id) AS TEXT))
  AND (CAST(sqlc.narg(pipeline_id) AS TEXT) IS NULL OR pipeline_id = CAST(sqlc.narg(pipeline_id) AS TEXT))
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL OR name LIKE CAST(sqlc.narg(search_pattern) AS TEXT) OR stage_name LIKE CAST(sqlc.narg(search_pattern) AS TEXT));

-- name: ListArtifactsByRun :many
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name, artifact.pipeline_id, artifact.pipeline_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location, artifact.value, artifact.value_format, artifact.image_ref, artifact.local_image_sha256, artifact.source_artifact_id, source_artifact.value AS source_commit_sha, binding.application_id, binding.application_name, binding.source_version_id, binding.source_version_label, binding.generated_version_id, binding.generated_version_label, version_component.id AS version_component_id, version_component.name AS version_component_name, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact ON source_artifact.id = artifact.source_artifact_id AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_version_binding AS binding ON binding.pipeline_run_id = artifact.pipeline_run_id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE artifact.pipeline_run_id = CAST(sqlc.arg(pipeline_run_id) AS TEXT)
  AND (CAST(sqlc.narg(project_id) AS TEXT) IS NULL OR artifact.project_id = CAST(sqlc.narg(project_id) AS TEXT))
ORDER BY artifact.created_at, artifact.id;

-- name: ArtifactByID :one
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name, artifact.pipeline_id, artifact.pipeline_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location, artifact.value, artifact.value_format, artifact.image_ref, artifact.local_image_sha256, artifact.source_artifact_id, source_artifact.value AS source_commit_sha, binding.application_id, binding.application_name, binding.source_version_id, binding.source_version_label, binding.generated_version_id, binding.generated_version_label, version_component.id AS version_component_id, version_component.name AS version_component_name, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact ON source_artifact.id = artifact.source_artifact_id AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_version_binding AS binding ON binding.pipeline_run_id = artifact.pipeline_run_id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE artifact.id = ?;
