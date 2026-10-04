PRAGMA foreign_keys = OFF;

BEGIN;

CREATE TABLE repository_credential_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    CONSTRAINT uq_repository_credential_name UNIQUE (name)
);

INSERT INTO repository_credential_new (
    id, name, type, encrypted_data, revision, created_at, updated_at, project_id
)
SELECT id, name, type, encrypted_data, revision, created_at, created_at, project_id
FROM repository_credential;

DROP TABLE repository_credential;
ALTER TABLE repository_credential_new RENAME TO repository_credential;

COMMIT;

PRAGMA foreign_keys = ON;
