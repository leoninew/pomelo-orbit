-- name: ListVersions :many
SELECT id, application_id, label, status, env_json, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE application_id = ?
ORDER BY created_at DESC, id;

-- name: CountVersions :one
SELECT COUNT(*)
FROM version
WHERE application_id = ?
  AND (? = '' OR label LIKE ? OR note LIKE ?);

-- name: ListVersionsPage :many
SELECT id, application_id, label, status, env_json, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE application_id = ?
  AND (? = '' OR label LIKE ? OR note LIKE ?)
ORDER BY created_at DESC, id
LIMIT ? OFFSET ?;

-- name: VersionByID :one
SELECT id, application_id, label, status, env_json, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE id = ?;

-- name: CreateVersion :exec
INSERT INTO version (id, application_id, label, status, env_json, created_from_version_id, note, component_summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateVersion :exec
UPDATE version
SET label = ?, status = ?, env_json = ?, note = ?, component_summary = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateVersionComponentSummary :exec
UPDATE version
SET component_summary = ?, updated_at = ?
WHERE id = ?;

-- name: CountVersionRuntimeRefs :one
SELECT (
  (SELECT COUNT(*) FROM service WHERE service.version_id = sqlc.arg(version_id) OR service.last_successful_version_id = sqlc.arg(version_id)) +
  (SELECT COUNT(*) FROM deployment WHERE deployment.version_id = sqlc.arg(version_id))
);

-- name: ClearVersionForkRefs :exec
UPDATE version
SET created_from_version_id = NULL
WHERE created_from_version_id = ?;

-- name: DeleteVersionComponents :exec
DELETE FROM version_component
WHERE version_id = ?;

-- name: DeleteVersionExposes :exec
DELETE FROM version_expose
WHERE version_id = ?;

-- name: DeleteVersion :exec
DELETE FROM version
WHERE id = ?;

-- name: VersionComponentsByVersion :many
SELECT id, version_id, name, image, command_json, args_json, env_json, ports_json, mounts_json, networks_json,
       depends_on_json, healthcheck_json, resources_json, pull_policy, restart_policy, tmpfs_json, ulimits_json, created_at, updated_at
FROM version_component
WHERE version_id = ?
ORDER BY name;

-- name: VersionComponentSecretEnvRefsByVersion :many
SELECT ref.component_id, ref.env_key, ref.credential_id, ref.data_key
FROM version_component_secret_env_ref AS ref
JOIN version_component AS component ON component.id = ref.component_id
WHERE component.version_id = ?
ORDER BY component.name, ref.env_key;

-- name: VersionExposesByVersion :many
SELECT id, version_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
FROM version_expose
WHERE version_id = ?
ORDER BY component_name, protocol, container_port;

-- name: InsertVersionComponent :exec
INSERT INTO version_component (
  id, version_id, name, image, command_json, args_json, env_json, ports_json, mounts_json, networks_json,
  depends_on_json, healthcheck_json, resources_json, pull_policy, restart_policy, tmpfs_json, ulimits_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertVersionComponentSecretEnvRef :exec
INSERT INTO version_component_secret_env_ref (component_id, env_key, credential_id, data_key)
VALUES (?, ?, ?, ?);

-- name: InsertVersionExpose :exec
INSERT INTO version_expose (
  id, version_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
