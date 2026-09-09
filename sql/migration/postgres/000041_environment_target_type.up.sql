ALTER TABLE environment ADD COLUMN target_type TEXT NOT NULL DEFAULT 'ssh' CHECK (target_type IN ('local', 'ssh'));
ALTER TABLE environment
    ALTER COLUMN platform DROP NOT NULL,
    ALTER COLUMN host DROP NOT NULL,
    ALTER COLUMN port DROP NOT NULL,
    ALTER COLUMN username DROP NOT NULL,
    ALTER COLUMN workspace_root DROP NOT NULL,
    ALTER COLUMN ssh_credential_id DROP NOT NULL,
    ALTER COLUMN ssh_credential_revision DROP NOT NULL,
    ALTER COLUMN host_key_fingerprint DROP NOT NULL;

ALTER TABLE deployment ADD COLUMN environment_target_type TEXT;
UPDATE deployment
SET environment_target_type = 'ssh'
WHERE environment_id IS NOT NULL;

UPDATE environment
SET state = 'active',
    target_type = 'local',
    platform = NULL,
    host = NULL,
    port = NULL,
    username = NULL,
    workspace_root = NULL,
    ssh_credential_id = NULL,
    ssh_credential_revision = NULL,
    host_key_fingerprint = NULL,
    last_probe_revision = NULL,
    last_probe_status = NULL,
    last_probe_at = NULL,
    last_probe_diagnostic = NULL,
    target_revision = target_revision + 1
WHERE id = '01M202WNXYDEY0E0GDF1RCY3JB';

DELETE FROM credential
WHERE id = '01M202WNXY6FPPFGTJWCF6CP81';
