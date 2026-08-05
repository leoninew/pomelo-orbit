-- name: CountArtifacts :one
SELECT COUNT(*)
FROM artifact
WHERE project_id = sqlc.arg(project_id)
  AND (sqlc.arg(repository_filter) = '' OR repository_id = sqlc.arg(repository_id))
  AND (sqlc.arg(template_filter) = '' OR template_id = sqlc.arg(template_id))
  AND (sqlc.arg(search_filter) = '' OR name LIKE sqlc.arg(search_pattern) OR location LIKE sqlc.arg(search_pattern));

-- name: ListArtifacts :many
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name,
       artifact.template_id, artifact.template_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location,
       artifact.value, artifact.value_format,
       artifact.image_ref, artifact.local_image_sha256,
       source_artifact.value AS source_commit_sha,
       pipeline_run_build_version_binding.application_id, pipeline_run_build_version_binding.application_name,
       pipeline_run_build_version_binding.source_version_id, pipeline_run_build_version_binding.source_version_label,
       pipeline_run_build_version_binding.generated_version_id, pipeline_run_build_version_binding.generated_version_label,
       pipeline_run_build_version_binding.component_name, version_component.id AS component_id, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact
  ON source_artifact.id = artifact.source_artifact_id
 AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_build_version_binding ON pipeline_run_build_version_binding.artifact_id = artifact.id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE artifact.project_id = sqlc.arg(project_id)
  AND (sqlc.arg(repository_filter) = '' OR artifact.repository_id = sqlc.arg(repository_id))
  AND (sqlc.arg(template_filter) = '' OR artifact.template_id = sqlc.arg(template_id))
  AND (sqlc.arg(search_filter) = '' OR artifact.name LIKE sqlc.arg(search_pattern) OR artifact.location LIKE sqlc.arg(search_pattern))
ORDER BY artifact.created_at DESC
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: ListArtifactsByRun :many
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name,
       artifact.template_id, artifact.template_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location,
       artifact.value, artifact.value_format,
       artifact.image_ref, artifact.local_image_sha256,
       source_artifact.value AS source_commit_sha,
       pipeline_run_build_version_binding.application_id, pipeline_run_build_version_binding.application_name,
       pipeline_run_build_version_binding.source_version_id, pipeline_run_build_version_binding.source_version_label,
       pipeline_run_build_version_binding.generated_version_id, pipeline_run_build_version_binding.generated_version_label,
       pipeline_run_build_version_binding.component_name, version_component.id AS component_id, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact
  ON source_artifact.id = artifact.source_artifact_id
 AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_build_version_binding ON pipeline_run_build_version_binding.artifact_id = artifact.id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE artifact.pipeline_run_id = sqlc.arg(pipeline_run_id)
  AND (sqlc.narg(project_id) IS NULL OR artifact.project_id = sqlc.narg(project_id))
ORDER BY artifact.created_at, artifact.id;

-- name: ArtifactByID :one
SELECT artifact.id, artifact.project_id, artifact.pipeline_run_id, artifact.repository_id, artifact.repository_name,
       artifact.template_id, artifact.template_name, artifact.pipeline_stage_id, artifact.stage_name, artifact.collector, artifact.name, artifact.location,
       artifact.value, artifact.value_format,
       artifact.image_ref, artifact.local_image_sha256,
       source_artifact.value AS source_commit_sha,
       pipeline_run_build_version_binding.application_id, pipeline_run_build_version_binding.application_name,
       pipeline_run_build_version_binding.source_version_id, pipeline_run_build_version_binding.source_version_label,
       pipeline_run_build_version_binding.generated_version_id, pipeline_run_build_version_binding.generated_version_label,
       pipeline_run_build_version_binding.component_name, version_component.id AS component_id, artifact.created_at
FROM artifact
LEFT JOIN artifact AS source_artifact
  ON source_artifact.id = artifact.source_artifact_id
 AND source_artifact.value_format = 'git_object_id'
LEFT JOIN pipeline_run_build_version_binding ON pipeline_run_build_version_binding.artifact_id = artifact.id
LEFT JOIN version_component ON version_component.artifact_id = artifact.id
WHERE artifact.id = sqlc.arg(artifact_id);

-- name: InsertArtifact :exec
INSERT INTO artifact (
  id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, pipeline_stage_id, stage_name,
  collector, name, location, value, value_format, image_ref, local_image_sha256, source_artifact_id, created_at
)
VALUES (
  sqlc.arg(id), sqlc.narg(project_id), sqlc.arg(pipeline_run_id), sqlc.arg(repository_id), sqlc.arg(repository_name),
  sqlc.arg(template_id), sqlc.arg(template_name), sqlc.arg(pipeline_stage_id), sqlc.arg(stage_name), sqlc.arg(collector),
  sqlc.arg(name), sqlc.narg(location), sqlc.narg(value), sqlc.narg(value_format), sqlc.narg(image_ref),
  sqlc.narg(local_image_sha256), sqlc.narg(source_artifact_id), sqlc.arg(created_at)
);

-- name: CommandArtifactByRunStageAndName :one
SELECT artifact.id, artifact.value, artifact.value_format
FROM artifact
WHERE artifact.pipeline_run_id = sqlc.arg(pipeline_run_id)
  AND artifact.pipeline_stage_id = sqlc.arg(pipeline_stage_id)
  AND artifact.name = sqlc.arg(name)
  AND artifact.collector = 'command';
