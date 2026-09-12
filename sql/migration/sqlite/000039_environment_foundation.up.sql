ALTER TABLE deployment ADD COLUMN environment_id TEXT;
ALTER TABLE deployment ADD COLUMN environment_target_revision BIGINT;
ALTER TABLE deployment ADD COLUMN ssh_credential_id TEXT;
ALTER TABLE deployment ADD COLUMN ssh_credential_revision BIGINT;
ALTER TABLE deployment ADD COLUMN gateway_application_id TEXT;
ALTER TABLE deployment ADD COLUMN environment_target_type TEXT;

ALTER TABLE credential ADD COLUMN revision INTEGER NOT NULL DEFAULT 1;

CREATE TABLE environment (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL CHECK (state IN ('active', 'disabled')),
    target_type TEXT NOT NULL CHECK (target_type IN ('local', 'ssh')),
    platform TEXT CHECK (platform IS NULL OR platform IN ('linux', 'windows')),
    host TEXT,
    port INTEGER CHECK (port IS NULL OR port BETWEEN 1 AND 65535),
    username TEXT,
    workspace_root TEXT,
    ssh_credential_id TEXT,
    ssh_credential_revision INTEGER CHECK (ssh_credential_revision IS NULL OR ssh_credential_revision >= 1),
    host_key_fingerprint TEXT,
    target_revision INTEGER NOT NULL CHECK (target_revision >= 1),
    last_probe_revision INTEGER,
    last_probe_status TEXT,
    last_probe_at DATETIME,
    last_probe_diagnostic TEXT,
    gateway_application_id TEXT UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
