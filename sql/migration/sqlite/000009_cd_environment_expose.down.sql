-- Down: drop new environment/expose/binding columns; no application_route rebuild.

DELETE FROM role_permission
WHERE permission_id IN (
    SELECT id FROM permission WHERE code IN ('environment:read', 'environment:write')
);
DELETE FROM permission WHERE code IN ('environment:read', 'environment:write');

CREATE TABLE service_old (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL UNIQUE,
    version_id TEXT NOT NULL,
    last_successful_version_id TEXT,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    FOREIGN KEY (last_successful_version_id) REFERENCES version(id)
);

INSERT INTO service_old (id, application_id, version_id, last_successful_version_id, status, created_at, updated_at)
SELECT id, application_id, version_id, last_successful_version_id, status, created_at, updated_at
FROM service
GROUP BY application_id
HAVING MIN(rowid);

DROP TABLE service;
ALTER TABLE service_old RENAME TO service;
CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);

-- SQLite cannot DROP COLUMN environment_id reliably on older builds; rebuild deployment if needed is out of scope.
-- Keep environment_id column if present (harmless); drop environment tables.

DROP TABLE IF EXISTS environment_binding;
DROP TABLE IF EXISTS version_expose;
DROP TABLE IF EXISTS environment;
