-- Environment (project-scoped), Expose, EnvironmentBinding; multi-Service keys; drop application_route / route_managed.

CREATE TABLE IF NOT EXISTS environment (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL,
    code VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_environment_project_code (project_id, code),
    KEY idx_environment_project (project_id),
    CONSTRAINT fk_environment_project FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_expose (
    id VARCHAR(26) PRIMARY KEY,
    version_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    path_prefix VARCHAR(512) NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_version_expose_key (version_id, component_name, protocol, container_port),
    KEY idx_version_expose_version (version_id),
    CONSTRAINT fk_version_expose_version FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS environment_binding (
    id VARCHAR(26) PRIMARY KEY,
    environment_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    domains_json LONGTEXT NOT NULL,
    entrypoint VARCHAR(128) NOT NULL,
    tls_mode VARCHAR(32) NOT NULL DEFAULT 'none',
    sni_host VARCHAR(255) NULL,
    note TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_env_binding_key (environment_id, component_name, protocol, container_port),
    KEY idx_environment_binding_env (environment_id),
    CONSTRAINT fk_environment_binding_env FOREIGN KEY (environment_id) REFERENCES environment(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO environment (id, project_id, code, name, description, created_at, updated_at)
SELECT
    LEFT(REPLACE(UUID(), '-', ''), 26),
    p.id,
    'local',
    'Local',
    NULL,
    CURRENT_TIMESTAMP(3),
    CURRENT_TIMESTAMP(3)
FROM project p
WHERE NOT EXISTS (
    SELECT 1 FROM environment e WHERE e.project_id = p.id AND e.code = 'local'
);

ALTER TABLE service DROP INDEX application_id;
ALTER TABLE service ADD COLUMN environment_id VARCHAR(26) NULL;
ALTER TABLE service ADD COLUMN instance_key VARCHAR(100) NOT NULL DEFAULT 'default';

UPDATE service s
INNER JOIN application a ON a.id = s.application_id
INNER JOIN environment e ON e.project_id = a.project_id AND e.code = 'local'
SET s.environment_id = e.id, s.instance_key = 'default';

ALTER TABLE service MODIFY environment_id VARCHAR(26) NOT NULL;
ALTER TABLE service ADD CONSTRAINT fk_service_environment FOREIGN KEY (environment_id) REFERENCES environment(id);
ALTER TABLE service ADD UNIQUE KEY uq_service_app_env_instance (application_id, environment_id, instance_key);
ALTER TABLE service ADD KEY idx_service_environment (environment_id);
ALTER TABLE service ADD KEY idx_service_app_env (application_id, environment_id);

ALTER TABLE deployment ADD COLUMN environment_id VARCHAR(26) NULL;
ALTER TABLE deployment ADD CONSTRAINT fk_deployment_environment FOREIGN KEY (environment_id) REFERENCES environment(id);

DROP TABLE IF EXISTS application_route;

ALTER TABLE application DROP COLUMN route_managed;

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAF',
    'environment:read',
    'View Environments',
    'View project environments and bindings',
    CURRENT_TIMESTAMP(3),
    CURRENT_TIMESTAMP(3)
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'environment:read');

INSERT INTO permission (id, code, name, description, created_at, updated_at)
SELECT
    '01KRXJXVPC6MQ75SZPWYZJSSAG',
    'environment:write',
    'Manage Environments',
    'Create and update project environments and bindings',
    CURRENT_TIMESTAMP(3),
    CURRENT_TIMESTAMP(3)
FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM permission WHERE code = 'environment:write');

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, CURRENT_TIMESTAMP(3)
FROM role
JOIN permission ON permission.code = 'environment:read'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );

INSERT INTO role_permission (role_id, permission_id, created_at)
SELECT role.id, permission.id, CURRENT_TIMESTAMP(3)
FROM role
JOIN permission ON permission.code = 'environment:write'
WHERE role.code = 'admin'
  AND NOT EXISTS (
      SELECT 1 FROM role_permission
      WHERE role_permission.role_id = role.id
        AND role_permission.permission_id = permission.id
  );
