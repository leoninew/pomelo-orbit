-- name: EnvironmentById :one
SELECT id, project_id, code, target_type, platform, host, port, username, workspace_root,
       ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
       last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
       gateway_application_id, created_at, updated_at
FROM environment
WHERE id = ?;

-- name: EnvironmentByProjectId :one
SELECT id, project_id, code, target_type, platform, host, port, username, workspace_root,
       ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
       last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
       gateway_application_id, created_at, updated_at
FROM environment
WHERE project_id = ?;

-- name: EnvironmentByTarget :one
SELECT id, project_id, code, target_type, platform, host, port, username, workspace_root,
       ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
       last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
       gateway_application_id, created_at, updated_at
FROM environment
WHERE project_id <> ?
  AND target_type = ?
  AND (? = 'local' OR (host = ? AND port = ?));

-- name: CreateEnvironment :exec
INSERT INTO environment (
  id, project_id, code, target_type, platform, host, port, username, workspace_root,
  ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
  last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
  gateway_application_id, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEnvironment :exec
UPDATE environment
SET code = ?, target_type = ?, platform = ?, host = ?, port = ?, username = ?, workspace_root = ?,
    ssh_credential_id = ?, ssh_credential_revision = ?, host_key_fingerprint = ?,
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
