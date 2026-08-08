-- name: ListVersions :many
SELECT id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE application_id = ?
ORDER BY created_at DESC, id;

-- name: CountVersions :one
SELECT COUNT(*)
FROM version
WHERE application_id = CAST(sqlc.arg(application_id) AS TEXT)
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR label LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR note LIKE CAST(sqlc.narg(search_pattern) AS TEXT));

-- name: ListVersionsPage :many
SELECT id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE application_id = CAST(sqlc.arg(application_id) AS TEXT)
  AND (CAST(sqlc.narg(search_pattern) AS TEXT) IS NULL
    OR label LIKE CAST(sqlc.narg(search_pattern) AS TEXT)
    OR note LIKE CAST(sqlc.narg(search_pattern) AS TEXT))
ORDER BY created_at DESC, id
LIMIT sqlc.arg(limit) OFFSET sqlc.arg(offset);

-- name: VersionByID :one
SELECT id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE id = ?;

-- name: LatestVersionByApplication :one
SELECT id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at
FROM version
WHERE application_id = ?
ORDER BY id DESC
LIMIT 1;

-- name: CreateVersion :exec
INSERT INTO version (id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateVersion :exec
UPDATE version
SET label = ?, status = ?, note = ?, component_summary = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateVersionComponentSummary :exec
UPDATE version
SET component_summary = ?, updated_at = ?
WHERE id = ?;

-- name: CountVersionRuntimeRefs :one
SELECT (
  SELECT COUNT(*) FROM service WHERE service.version_id = sqlc.arg(version_id)
);

-- name: ClearVersionForkRefs :exec
UPDATE version
SET created_from_version_id = NULL
WHERE created_from_version_id = sqlc.arg(version_id);

-- name: DeleteVersionComponents :exec
DELETE FROM version_component
WHERE version_id = ?;

-- name: DeleteVersion :exec
DELETE FROM version
WHERE id = ?;

-- name: VersionComponentsByVersion :many
SELECT id, version_id, name, image, artifact_id, artifact_name, artifact_image_ref, artifact_local_image_sha256, artifact_source_commit_sha, entrypoint_json, command_json, pull_policy, restart_policy, created_at, updated_at
FROM version_component
WHERE version_id = ?
ORDER BY name;

-- name: VersionComponentByID :one
SELECT id, version_id, name, image, artifact_id, artifact_name, artifact_image_ref, artifact_local_image_sha256, artifact_source_commit_sha, entrypoint_json, command_json, pull_policy, restart_policy, created_at, updated_at
FROM version_component
WHERE id = ?;

-- name: InsertVersionComponent :exec
INSERT INTO version_component (
  id, version_id, name, image, artifact_id, artifact_name, artifact_image_ref, artifact_local_image_sha256, artifact_source_commit_sha, entrypoint_json, command_json, pull_policy, restart_policy, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateVersionComponentBasic :exec
UPDATE version_component
SET name = ?, image = ?, pull_policy = ?, restart_policy = ?, updated_at = ?
WHERE id = ?;

-- name: TouchVersionComponent :exec
UPDATE version_component
SET updated_at = ?
WHERE id = ?;

-- name: DeleteVersionComponent :exec
DELETE FROM version_component
WHERE id = ?;

-- name: UpdateVersionComponentCommand :exec
UPDATE version_component
SET command_json = ?, updated_at = ?
WHERE id = ?;

-- name: UpdateVersionComponentEntrypoint :exec
UPDATE version_component
SET entrypoint_json = ?, updated_at = ?
WHERE id = ?;

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

-- name: InsertVersionComponentEndpoint :exec
INSERT INTO version_component_endpoint (
  component_id, name, protocol, container_port, mode, bind_address, listen_port, entrypoint, path_prefix, position
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: VersionComponentEndpointsByComponent :many
SELECT component_id, name, protocol, container_port, mode, bind_address, listen_port, entrypoint, path_prefix, position
FROM version_component_endpoint
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentEndpoints :exec
DELETE FROM version_component_endpoint
WHERE component_id = ?;

-- name: InsertVersionComponentMount :exec
INSERT INTO version_component_mount (
  component_id, source_type, source, target, read_only, source_is_host_path, content, mode, ignore_if_exists, position
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: VersionComponentMountsByComponent :many
SELECT component_id, source_type, source, target, read_only, source_is_host_path, content, mode, ignore_if_exists, position
FROM version_component_mount
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentMounts :exec
DELETE FROM version_component_mount
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
  component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: VersionComponentHealthcheckByComponent :one
SELECT component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled
FROM version_component_healthcheck
WHERE component_id = ?;

-- name: DeleteVersionComponentHealthcheck :exec
DELETE FROM version_component_healthcheck
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

-- name: InsertVersionComponentDevice :exec
INSERT INTO version_component_device (component_id, driver, device_count, capabilities_json, position)
VALUES (?, ?, ?, ?, ?);

-- name: VersionComponentDevicesByComponent :many
SELECT component_id, driver, device_count, capabilities_json, position
FROM version_component_device
WHERE component_id = ?
ORDER BY position;

-- name: DeleteVersionComponentDevices :exec
DELETE FROM version_component_device
WHERE component_id = ?;

-- name: RenameVersionComponentDependencies :exec
UPDATE version_component_dependency
SET depends_on_name = sqlc.arg(new_name)
WHERE component_id IN (
  SELECT id FROM version_component WHERE version_id = sqlc.arg(version_id)
)
  AND depends_on_name = sqlc.arg(old_name);
