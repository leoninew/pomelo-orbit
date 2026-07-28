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
  (SELECT COUNT(*) FROM service WHERE service.version_id = sqlc.arg(version_id)) +
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
SELECT id, version_id, name, image, pull_policy, restart_policy, created_at, updated_at
FROM version_component
WHERE version_id = ?
ORDER BY name;

-- name: VersionComponentByID :one
SELECT id, version_id, name, image, pull_policy, restart_policy, created_at, updated_at
FROM version_component
WHERE id = ?;

-- name: VersionExposesByVersion :many
SELECT id, version_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
FROM version_expose
WHERE version_id = ?
ORDER BY component_name, protocol, container_port;

-- name: InsertVersionComponent :exec
INSERT INTO version_component (
  id, version_id, name, image, pull_policy, restart_policy, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateVersionComponent :exec
UPDATE version_component
SET name = ?, image = ?, pull_policy = ?, restart_policy = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteVersionComponent :exec
DELETE FROM version_component
WHERE id = ?;

-- name: InsertVersionComponentArgument :exec
INSERT INTO version_component_argument (component_id, kind, position, value)
VALUES (?, ?, ?, ?);

-- name: VersionComponentArgumentsByComponent :many
SELECT component_id, kind, position, value
FROM version_component_argument
WHERE component_id = ?
ORDER BY kind, position;

-- name: DeleteVersionComponentArguments :exec
DELETE FROM version_component_argument
WHERE component_id = ?;

-- name: InsertVersionComponentEnv :exec
INSERT INTO version_component_env (component_id, env_key, value, position)
VALUES (?, ?, ?, ?);

-- name: VersionComponentEnvByComponent :many
SELECT component_id, env_key, value, position
FROM version_component_env
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentEnv :exec
DELETE FROM version_component_env
WHERE component_id = ?;

-- name: InsertVersionComponentPort :exec
INSERT INTO version_component_port (component_id, host_port, container_port, position)
VALUES (?, ?, ?, ?);

-- name: VersionComponentPortsByComponent :many
SELECT component_id, host_port, container_port, position
FROM version_component_port
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentPorts :exec
DELETE FROM version_component_port
WHERE component_id = ?;

-- name: InsertVersionComponentMount :exec
INSERT INTO version_component_mount (
  component_id, source_type, source, target, read_only, content, content_mode, position
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: VersionComponentMountsByComponent :many
SELECT component_id, source_type, source, target, read_only, content, content_mode, position
FROM version_component_mount
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentMounts :exec
DELETE FROM version_component_mount
WHERE component_id = ?;

-- name: InsertVersionComponentNetwork :exec
INSERT INTO version_component_network (component_id, name, position)
VALUES (?, ?, ?);

-- name: VersionComponentNetworksByComponent :many
SELECT component_id, name, position
FROM version_component_network
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentNetworks :exec
DELETE FROM version_component_network
WHERE component_id = ?;

-- name: InsertVersionComponentDependency :exec
INSERT INTO version_component_dependency (component_id, depends_on_name, condition, position)
VALUES (?, ?, ?, ?);

-- name: VersionComponentDependenciesByComponent :many
SELECT component_id, depends_on_name, condition, position
FROM version_component_dependency
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentDependencies :exec
DELETE FROM version_component_dependency
WHERE component_id = ?;

-- name: InsertVersionComponentHealthcheck :exec
INSERT INTO version_component_healthcheck (
  component_id, test_mode, interval, timeout, retries, start_period, start_interval, disabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: VersionComponentHealthcheckByComponent :one
SELECT component_id, test_mode, interval, timeout, retries, start_period, start_interval, disabled
FROM version_component_healthcheck
WHERE component_id = ?;

-- name: DeleteVersionComponentHealthcheck :exec
DELETE FROM version_component_healthcheck
WHERE component_id = ?;

-- name: InsertVersionComponentHealthcheckArg :exec
INSERT INTO version_component_healthcheck_arg (component_id, position, value)
VALUES (?, ?, ?);

-- name: VersionComponentHealthcheckArgsByComponent :many
SELECT component_id, position, value
FROM version_component_healthcheck_arg
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentHealthcheckArgs :exec
DELETE FROM version_component_healthcheck_arg
WHERE component_id = ?;

-- name: InsertVersionComponentResource :exec
INSERT INTO version_component_resource (
  component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory
) VALUES (?, ?, ?, ?, ?);

-- name: VersionComponentResourceByComponent :one
SELECT component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory
FROM version_component_resource
WHERE component_id = ?;

-- name: DeleteVersionComponentResource :exec
DELETE FROM version_component_resource
WHERE component_id = ?;

-- name: InsertVersionComponentTmpfs :exec
INSERT INTO version_component_tmpfs (component_id, target, size_bytes, mode, position)
VALUES (?, ?, ?, ?, ?);

-- name: VersionComponentTmpfsByComponent :many
SELECT component_id, target, size_bytes, mode, position
FROM version_component_tmpfs
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentTmpfs :exec
DELETE FROM version_component_tmpfs
WHERE component_id = ?;

-- name: InsertVersionComponentUlimit :exec
INSERT INTO version_component_ulimit (component_id, name, soft, hard, position)
VALUES (?, ?, ?, ?, ?);

-- name: VersionComponentUlimitsByComponent :many
SELECT component_id, name, soft, hard, position
FROM version_component_ulimit
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentUlimits :exec
DELETE FROM version_component_ulimit
WHERE component_id = ?;

-- name: RenameVersionComponentExposes :exec
UPDATE version_expose
SET component_name = sqlc.arg(new_name), updated_at = sqlc.arg(updated_at)
WHERE version_id = sqlc.arg(version_id)
  AND component_name = sqlc.arg(old_name);

-- name: RenameVersionComponentDependencies :exec
UPDATE version_component_dependency
SET depends_on_name = sqlc.arg(new_name)
WHERE component_id IN (
  SELECT id FROM version_component WHERE version_id = sqlc.arg(version_id)
)
  AND depends_on_name = sqlc.arg(old_name);

-- name: InsertVersionExpose :exec
INSERT INTO version_expose (
  id, version_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
