DROP INDEX IF EXISTS uq_application_project_name;
DROP INDEX IF EXISTS uq_application_project_code;
PRAGMA foreign_keys = OFF;
CREATE TABLE application_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'standard',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);
INSERT INTO application_old (id, name, code, kind, created_at, updated_at, project_id)
SELECT id, name, code, kind, created_at, updated_at, project_id FROM application;
DROP TABLE application;
ALTER TABLE application_old RENAME TO application;
CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);
PRAGMA foreign_keys = ON;

UPDATE environment
SET gateway_application_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE id = project_id;

DELETE FROM environment
WHERE id = project_id
  AND target_type = 'local'
  AND target_revision = 1;
