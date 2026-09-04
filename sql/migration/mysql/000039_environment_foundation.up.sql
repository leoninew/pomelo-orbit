ALTER TABLE deployment ADD COLUMN environment_id VARCHAR(26);
ALTER TABLE deployment ADD COLUMN environment_target_revision BIGINT;
ALTER TABLE deployment ADD COLUMN ssh_credential_id VARCHAR(26);
ALTER TABLE deployment ADD COLUMN ssh_credential_revision BIGINT;
ALTER TABLE deployment ADD COLUMN gateway_application_id VARCHAR(26);

ALTER TABLE credential ADD COLUMN revision BIGINT NOT NULL DEFAULT 1;

CREATE TABLE environment (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL UNIQUE,
    code VARCHAR(100) NOT NULL UNIQUE,
    state VARCHAR(16) NOT NULL,
    platform VARCHAR(16) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port BIGINT NOT NULL,
    username VARCHAR(255) NOT NULL,
    workspace_root TEXT NOT NULL,
    ssh_credential_id VARCHAR(26) NOT NULL,
    ssh_credential_revision BIGINT NOT NULL,
    host_key_fingerprint VARCHAR(255) NOT NULL,
    target_revision BIGINT NOT NULL,
    last_probe_revision BIGINT,
    last_probe_status VARCHAR(32),
    last_probe_at DATETIME,
    last_probe_diagnostic TEXT,
    gateway_application_id VARCHAR(26) UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_environment_state CHECK (state IN ('active', 'disabled')),
    CONSTRAINT chk_environment_platform CHECK (platform IN ('linux', 'windows')),
    CONSTRAINT chk_environment_port CHECK (port BETWEEN 1 AND 65535),
    CONSTRAINT chk_environment_revision CHECK (ssh_credential_revision >= 1 AND target_revision >= 1)
);
