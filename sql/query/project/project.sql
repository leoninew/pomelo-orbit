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
WHERE project_member.user_id = ? AND project.is_active = TRUE
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
SET name = ?, updated_at = ?
WHERE id = ?;

-- name: DeprecateProject :exec
UPDATE project
SET is_active = FALSE, updated_at = ?
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
SELECT u.id, u.username, u.password_hash, u.status, u.oauth_provider, u.oauth_provider_id,
       u.email, u.auth_source, u.created_at, u.updated_at, u.last_login_at
FROM `user` AS u
JOIN project_member ON project_member.user_id = u.id
WHERE project_member.project_id = ?
ORDER BY u.username;

-- name: AddProjectMember :exec
INSERT INTO project_member (project_id, user_id, created_at)
VALUES (?, ?, ?);

-- name: RemoveProjectMember :exec
DELETE FROM project_member
WHERE project_id = ? AND user_id = ?;

-- name: RemoveUserFromAllProjects :exec
DELETE FROM project_member
WHERE user_id = ?;
