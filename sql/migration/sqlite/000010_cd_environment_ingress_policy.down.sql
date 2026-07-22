-- Down: recreate empty environment_binding; drop policy columns via table rebuild.

CREATE TABLE IF NOT EXISTS environment_binding (
    id TEXT PRIMARY KEY,
    environment_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    domains_json TEXT NOT NULL DEFAULT '[]',
    entrypoint TEXT NOT NULL,
    tls_mode TEXT NOT NULL DEFAULT 'none',
    sni_host TEXT,
    note TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (environment_id) REFERENCES environment(id) ON DELETE CASCADE,
    UNIQUE(environment_id, component_name, protocol, container_port)
);
CREATE INDEX IF NOT EXISTS idx_environment_binding_env ON environment_binding(environment_id);

CREATE TABLE environment_old (
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

INSERT INTO environment_old (id, project_id, code, name, description, created_at, updated_at)
SELECT id, project_id, code, name, description, created_at, updated_at FROM environment;

DROP TABLE environment;
ALTER TABLE environment_old RENAME TO environment;
CREATE INDEX IF NOT EXISTS idx_environment_project ON environment(project_id);
