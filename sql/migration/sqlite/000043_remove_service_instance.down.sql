PRAGMA foreign_keys = OFF;

CREATE TABLE service_old (
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

INSERT INTO service_old (id, project_id, application_id, instance_key, code, version_id, status, created_at, updated_at)
SELECT id, project_id, application_id, code, code, version_id, status, created_at, updated_at
FROM service;

DROP TABLE service;
ALTER TABLE service_old RENAME TO service;

CREATE UNIQUE INDEX uq_service_project_code ON service(project_id, code);
CREATE INDEX idx_service_version ON service(version_id);
CREATE INDEX idx_service_application ON service(application_id);

PRAGMA foreign_keys = ON;
