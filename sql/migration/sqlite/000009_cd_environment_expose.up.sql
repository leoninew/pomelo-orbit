-- Environment (project-scoped), Expose, EnvironmentBinding; multi-Service keys; drop application_route / route_managed.

CREATE TABLE IF NOT EXISTS environment (
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

CREATE INDEX IF NOT EXISTS idx_environment_project ON environment(project_id);

CREATE TABLE IF NOT EXISTS version_expose (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    path_prefix TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, component_name, protocol, container_port)
);

CREATE INDEX IF NOT EXISTS idx_version_expose_version ON version_expose(version_id);

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

-- Seed local environment per existing project.
INSERT INTO environment (id, project_id, code, name, description, created_at, updated_at)
SELECT
    lower(hex(randomblob(13))),
    p.id,
    'local',
    'Local',
    NULL,
    datetime('now'),
    datetime('now')
FROM project p
WHERE NOT EXISTS (
    SELECT 1 FROM environment e WHERE e.project_id = p.id AND e.code = 'local'
);

-- Rebuild service with multi-instance keys.
CREATE TABLE service_new (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    version_id TEXT NOT NULL,
    last_successful_version_id TEXT,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES environment(id),
    FOREIGN KEY (version_id) REFERENCES version(id),
    FOREIGN KEY (last_successful_version_id) REFERENCES version(id),
    UNIQUE(application_id, environment_id, instance_key)
);

INSERT INTO service_new (
    id, application_id, environment_id, instance_key,
    version_id, last_successful_version_id, status, created_at, updated_at
)
SELECT
    s.id,
    s.application_id,
    e.id,
    'default',
    s.version_id,
    s.last_successful_version_id,
    s.status,
    s.created_at,
    s.updated_at
FROM service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN environment e ON e.project_id = a.project_id AND e.code = 'local';

DROP TABLE service;
ALTER TABLE service_new RENAME TO service;

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_environment ON service(environment_id);
CREATE INDEX IF NOT EXISTS idx_service_app_env ON service(application_id, environment_id);

ALTER TABLE deployment ADD COLUMN environment_id TEXT REFERENCES environment(id);

DROP TABLE IF EXISTS application_route;

ALTER TABLE application DROP COLUMN route_managed;

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAF',
    'environment:read',
    'View Environments',
    'View project environments and bindings',
    datetime('now'),
    datetime('now')
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'environment:read');

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAG',
    'environment:write',
    'Manage Environments',
    'Create and update project environments and bindings',
    datetime('now'),
    datetime('now')
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'environment:write');

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, datetime('now')
FROM role
JOIN permission ON permission.code = 'environment:read'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, datetime('now')
FROM role
JOIN permission ON permission.code = 'environment:write'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );
