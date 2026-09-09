ALTER TABLE environment ADD COLUMN target_type VARCHAR(16) NOT NULL DEFAULT 'ssh' AFTER state;
ALTER TABLE environment
    MODIFY COLUMN platform VARCHAR(16) NULL,
    MODIFY COLUMN host VARCHAR(255) NULL,
    MODIFY COLUMN port BIGINT NULL,
    MODIFY COLUMN username VARCHAR(255) NULL,
    MODIFY COLUMN workspace_root TEXT NULL,
    MODIFY COLUMN ssh_credential_id VARCHAR(26) NULL,
    MODIFY COLUMN ssh_credential_revision BIGINT NULL,
    MODIFY COLUMN host_key_fingerprint VARCHAR(255) NULL;
ALTER TABLE environment ADD CONSTRAINT chk_environment_target_type CHECK (target_type IN ('local', 'ssh'));

ALTER TABLE deployment ADD COLUMN environment_target_type VARCHAR(16);
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
