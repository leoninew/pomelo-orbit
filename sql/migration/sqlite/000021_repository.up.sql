-- Domain: repository
-- Tables: repository
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS repository (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
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

CREATE INDEX IF NOT EXISTS idx_repository_name ON repository(name);
CREATE INDEX IF NOT EXISTS idx_repository_code ON repository(code);
CREATE INDEX IF NOT EXISTS idx_repository_credential ON repository(git_credential_id);
CREATE INDEX IF NOT EXISTS idx_repository_project ON repository(project_id);
