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
SELECT environment.id, environment.project_id, environment.code, environment.target_type,
       environment.platform, environment.host, environment.port, environment.username,
       environment.workspace_root, environment.ssh_credential_id,
       environment.ssh_credential_revision, environment.host_key_fingerprint,
       environment.target_revision, environment.last_probe_revision,
       environment.last_probe_status, environment.last_probe_at,
       environment.last_probe_diagnostic, environment.gateway_application_id,
       environment.created_at, environment.updated_at
FROM environment
JOIN project ON project.id = environment.project_id
WHERE environment.project_id <> ?
  AND project.is_active = TRUE
  AND environment.target_type = ?
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
