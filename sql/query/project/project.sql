-- name: ProjectByID :one
SELECT id, name, code, is_active, created_at, updated_at
FROM project
WHERE id = ?;

-- name: ProjectByCode :one
SELECT id, name, code, is_active, created_at, updated_at
FROM project
WHERE code = ?;

-- name: ListProjectsByMember :many
SELECT project.id, project.name, project.code, project.is_active, project.created_at, project.updated_at
FROM project
JOIN project_member ON project_member.project_id = project.id
WHERE project_member.user_id = ?
ORDER BY project.id DESC;

-- name: ListActiveProjectsByMember :many
SELECT project.id, project.name, project.code, project.is_active, project.created_at, project.updated_at
FROM project
JOIN project_member ON project_member.project_id = project.id
WHERE project_member.user_id = ? AND project.is_active = 1
ORDER BY project.id DESC;

-- name: IsProjectMember :one
SELECT COUNT(*)
FROM project_member
WHERE project_id = ? AND user_id = ?;

-- name: CreateProject :exec
INSERT INTO project (id, name, code, is_active, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateProject :exec
UPDATE project
SET name = ?, code = ?, updated_at = ?
WHERE id = ?;

-- name: DeprecateProject :exec
UPDATE project
SET is_active = 0, updated_at = ?
WHERE id = ?;

-- name: CountProjectRepositories :one
SELECT COUNT(*)
FROM repository
WHERE project_id = ?;

-- name: CountProjectApplications :one
SELECT COUNT(*)
FROM application
WHERE project_id = ?;

-- name: ProjectMembers :many
SELECT user.id, user.username, user.password_hash, user.status, user.oauth_provider, user.oauth_provider_id,
       user.email, user.auth_source, user.created_at, user.updated_at, user.last_login_at
FROM user
JOIN project_member ON project_member.user_id = user.id
WHERE project_member.project_id = ?
ORDER BY user.username;

-- name: AddProjectMember :exec
INSERT INTO project_member (project_id, user_id, created_at)
VALUES (?, ?, ?);

-- name: RemoveProjectMember :exec
DELETE FROM project_member
WHERE project_id = ? AND user_id = ?;
