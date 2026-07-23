-- Move Traefik entrypoint/TLS policy from environment to gateway_config.

ALTER TABLE gateway_config ADD COLUMN default_entrypoint TEXT NOT NULL DEFAULT 'web';
ALTER TABLE gateway_config ADD COLUMN tcp_entrypoint TEXT;
ALTER TABLE gateway_config ADD COLUMN tls_mode TEXT NOT NULL DEFAULT 'none';

UPDATE gateway_config
SET
    default_entrypoint = COALESCE(NULLIF(TRIM(default_entrypoint), ''), 'web'),
    tls_mode = COALESCE(NULLIF(TRIM(tls_mode), ''), 'none');

CREATE TABLE environment_new (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
    UNIQUE(project_id, code)
);

INSERT INTO environment_new (id, project_id, code, name, description, created_at, updated_at)
SELECT id, project_id, code, name, description, created_at, updated_at
FROM environment;

DROP TABLE environment;
ALTER TABLE environment_new RENAME TO environment;
CREATE INDEX IF NOT EXISTS idx_environment_project ON environment(project_id);
