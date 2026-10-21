-- name: EnvironmentById :one
SELECT e.id, e.project_id, e.code, e.target_type, e.platform, e.host, e.port, e.username, e.workspace_root,
       c.id AS ssh_credential_id, c.revision AS ssh_credential_revision, e.host_key_fingerprint, e.target_revision,
       e.last_probe_revision, e.last_probe_status, e.last_probe_at, e.last_probe_diagnostic,
       e.gateway_application_id, e.created_at, e.updated_at
FROM environment e
LEFT JOIN environment_credential c ON c.environment_id = e.id AND e.target_type = 'ssh'
WHERE e.id = ?;

-- name: EnvironmentByProjectId :one
SELECT e.id, e.project_id, e.code, e.target_type, e.platform, e.host, e.port, e.username, e.workspace_root,
       c.id AS ssh_credential_id, c.revision AS ssh_credential_revision, e.host_key_fingerprint, e.target_revision,
       e.last_probe_revision, e.last_probe_status, e.last_probe_at, e.last_probe_diagnostic,
       e.gateway_application_id, e.created_at, e.updated_at
FROM environment e
LEFT JOIN environment_credential c ON c.environment_id = e.id AND e.target_type = 'ssh'
WHERE e.project_id = ?;

-- name: EnvironmentByTarget :one
SELECT e.id, e.project_id, e.code, e.target_type, e.platform, e.host, e.port, e.username, e.workspace_root,
       c.id AS ssh_credential_id, c.revision AS ssh_credential_revision, e.host_key_fingerprint, e.target_revision,
       e.last_probe_revision, e.last_probe_status, e.last_probe_at, e.last_probe_diagnostic,
       e.gateway_application_id, e.created_at, e.updated_at
FROM environment e
LEFT JOIN environment_credential c ON c.environment_id = e.id AND e.target_type = 'ssh'
JOIN project p ON p.id = e.project_id
WHERE e.project_id <> sqlc.arg(project_id)
  AND p.is_active = TRUE
  AND e.target_type = sqlc.arg(target_type)
  AND (sqlc.arg(target_type) = 'local' OR (e.host = sqlc.arg(host) AND e.port = sqlc.arg(port)));

-- name: CreateEnvironment :exec
INSERT INTO environment (
  id, project_id, code, target_type, platform, host, port, username, workspace_root,
  host_key_fingerprint, target_revision,
  last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
  gateway_application_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEnvironment :exec
UPDATE environment
SET code = ?, target_type = ?, platform = ?, host = ?, port = ?, username = ?, workspace_root = ?,
    host_key_fingerprint = ?,
    target_revision = ?, last_probe_revision = ?, last_probe_status = ?, last_probe_at = ?, last_probe_diagnostic = ?,
    gateway_application_id = ?, updated_at = ?
WHERE id = ?;

-- name: BindGatewayApplication :execrows
UPDATE environment
SET gateway_application_id = ?, updated_at = ?
WHERE id = ?
  AND gateway_application_id IS NULL;

-- name: UnbindGatewayApplication :execrows
UPDATE environment
SET gateway_application_id = NULL, updated_at = ?
WHERE id = ?
  AND gateway_application_id = ?;
-- name: RecordEnvironmentProbe :execrows
UPDATE environment
SET last_probe_revision = ?, last_probe_status = ?, last_probe_at = ?, last_probe_diagnostic = ?,
    updated_at = ?
WHERE id = ?
  AND target_revision = ?;

-- name: RecordEnvironmentHostKey :execrows
UPDATE environment
SET host_key_fingerprint = sqlc.arg(fingerprint), updated_at = sqlc.arg(updated_at)
WHERE environment.id = sqlc.arg(id)
  AND target_type = 'ssh'
  AND target_revision = sqlc.arg(target_revision)
  AND EXISTS (
      SELECT 1 FROM environment_credential c
      WHERE c.environment_id = environment.id
        AND c.id = sqlc.arg(ssh_credential_id)
        AND c.revision = sqlc.arg(ssh_credential_revision)
  )
  AND (host_key_fingerprint IS NULL OR host_key_fingerprint = '' OR host_key_fingerprint = sqlc.arg(fingerprint));
