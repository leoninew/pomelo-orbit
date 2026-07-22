DROP TABLE IF EXISTS gateway_config;

CREATE TABLE environment_old (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    base_domain TEXT NOT NULL DEFAULT 'local.test',
    domain_template TEXT,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tcp_entrypoint TEXT,
    tls_mode TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE,
    UNIQUE(project_id, code)
);

INSERT INTO environment_old (
    id, project_id, code, name, description,
    base_domain, domain_template, default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at
)
SELECT
    id, project_id, code, name, description,
    'local.test', NULL, default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at
FROM environment;

DROP TABLE environment;
ALTER TABLE environment_old RENAME TO environment;
CREATE INDEX IF NOT EXISTS idx_environment_project ON environment(project_id);
