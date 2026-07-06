-- name: RepositoryByID :one
SELECT id, project_id, name, code, repository_url, git_credential_id,
       variable_overrides, default_branch, created_at, updated_at
FROM repository
WHERE id = ?;

-- name: RepositoryHasRunningPipelines :one
SELECT COUNT(*)
FROM pipeline_run
WHERE repository_id = ? AND status IN (?, ?);
