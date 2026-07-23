-- Restore entrypoint/TLS policy columns on environment (values defaulted; gateway columns dropped).

CREATE TABLE environment_new (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tcp_entrypoint TEXT,
    tls_mode TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
    UNIQUE(project_id, code)
);

INSERT INTO environment_new (
    id, project_id, code, name, description,
    default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at
)
SELECT
    id, project_id, code, name, description,
    'web', NULL, 'none', created_at, updated_at
FROM environment;

DROP TABLE environment;
ALTER TABLE environment_new RENAME TO environment;
CREATE INDEX IF NOT EXISTS idx_environment_project ON environment(project_id);

-- SQLite cannot DROP COLUMN portably across all versions used here; rebuild gateway_config.
CREATE TABLE gateway_config_new (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    base_domain TEXT NOT NULL,
    image TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

INSERT INTO gateway_config_new (application_id, rest_api_url, base_domain, image, created_at, updated_at)
SELECT application_id, rest_api_url, base_domain, image, created_at, updated_at
FROM gateway_config;

DROP TABLE gateway_config;
ALTER TABLE gateway_config_new RENAME TO gateway_config;
