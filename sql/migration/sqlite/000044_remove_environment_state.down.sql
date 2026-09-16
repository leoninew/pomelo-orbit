PRAGMA foreign_keys = OFF;

CREATE TABLE environment_old (
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
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO environment_old (
    id, project_id, code, state, target_type, platform, host, port, username, workspace_root,
    ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
    last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
    gateway_application_id, created_at, updated_at
)
SELECT
    id, project_id, code, 'active', target_type, platform, host, port, username, workspace_root,
    ssh_credential_id, ssh_credential_revision, host_key_fingerprint, target_revision,
    last_probe_revision, last_probe_status, last_probe_at, last_probe_diagnostic,
    gateway_application_id, created_at, updated_at
FROM environment;

DROP TABLE environment;
ALTER TABLE environment_old RENAME TO environment;

PRAGMA foreign_keys = ON;
