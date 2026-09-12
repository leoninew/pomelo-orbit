-- Adopt the Gateway bundle created by the pre-Environment schema.
-- The target is intentionally unprobed and has no workspace root; the
-- Project Initialization Wizard must collect and probe those values.
-- Application names are unique within a Project; each Project may own a
-- managed Gateway with the product name 'Traefik'.
PRAGMA foreign_keys = OFF;
CREATE TABLE application_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'standard',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);
INSERT INTO application_new (id, name, code, kind, created_at, updated_at, project_id)
SELECT id, name, code, kind, created_at, updated_at, project_id FROM application;
DROP TABLE application;
ALTER TABLE application_new RENAME TO application;
CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_application_project_name ON application(project_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS uq_application_project_code ON application(project_id, code);

PRAGMA foreign_keys = OFF;
CREATE TABLE service_new (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    application_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    code TEXT NOT NULL,
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    UNIQUE(application_id, instance_key)
);
INSERT INTO service_new (id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at)
SELECT s.id, a.project_id, s.application_id, s.instance_key, s.code, s.version_id, s.status, s.created_at, s.updated_at
FROM service s JOIN application a ON a.id = s.application_id;
DROP TABLE service;
ALTER TABLE service_new RENAME TO service;
CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_application ON service(application_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_service_project_code ON service(project_id, code);
PRAGMA foreign_keys = ON;
PRAGMA foreign_keys = ON;

INSERT INTO environment (
    id, project_id, code, state, target_type, target_revision,
    gateway_application_id
)
SELECT p.id, p.id, p.code, 'active', 'local', 1, a.id
FROM project p
JOIN application a ON a.project_id = p.id
JOIN gateway_config gc ON gc.application_id = a.id
WHERE a.name = 'Traefik'
  AND a.code = 'traefik'
  AND EXISTS (
      SELECT 1 FROM service s
      WHERE s.application_id = a.id AND s.instance_key = 'default'
  )
  AND NOT EXISTS (
      SELECT 1 FROM environment e
      WHERE e.project_id = p.id OR e.id = p.id OR e.gateway_application_id = a.id
  );

UPDATE environment
SET gateway_application_id = (
        SELECT a.id
        FROM application a
        JOIN gateway_config gc ON gc.application_id = a.id
        WHERE a.project_id = environment.project_id
          AND a.name = 'Traefik'
          AND a.code = 'traefik'
          AND EXISTS (
              SELECT 1 FROM service s
              WHERE s.application_id = a.id AND s.instance_key = 'default'
          )
          AND NOT EXISTS (
              SELECT 1 FROM environment other
              WHERE other.gateway_application_id = a.id
                AND other.id <> environment.id
          )
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE gateway_application_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM application a
      JOIN gateway_config gc ON gc.application_id = a.id
      WHERE a.project_id = environment.project_id
        AND a.name = 'Traefik'
        AND a.code = 'traefik'
        AND EXISTS (
            SELECT 1 FROM service s
            WHERE s.application_id = a.id AND s.instance_key = 'default'
        )
  );
