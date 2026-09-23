PRAGMA foreign_keys = OFF;
CREATE TABLE repository_old (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    repository_type TEXT NOT NULL DEFAULT 'remote_git',
    repository_url TEXT NOT NULL,
    git_credential_id TEXT,
    variable_overrides TEXT NOT NULL DEFAULT '[]',
    default_branch TEXT NOT NULL DEFAULT 'master',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);
INSERT INTO repository_old (
    id, name, code, repository_type, repository_url, git_credential_id,
    variable_overrides, default_branch, created_at, updated_at, project_id
)
SELECT
    id, name, code, repository_type, repository_url, git_credential_id,
    variable_overrides, default_branch, created_at, updated_at, project_id
FROM repository;
DROP TABLE repository;
ALTER TABLE repository_old RENAME TO repository;
CREATE INDEX IF NOT EXISTS idx_repository_name ON repository(name);
CREATE INDEX IF NOT EXISTS idx_repository_code ON repository(code);
CREATE INDEX IF NOT EXISTS idx_repository_credential ON repository(git_credential_id);
CREATE INDEX IF NOT EXISTS idx_repository_project ON repository(project_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_repository_project_code ON repository(project_id, code);
ALTER TABLE repository_credential RENAME TO credential;
DROP TABLE IF EXISTS environment_credential;
PRAGMA foreign_keys = ON;
