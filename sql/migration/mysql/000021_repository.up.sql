-- Domain: repository
-- Tables: repository
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS repository (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(255) NOT NULL UNIQUE,
    repository_type VARCHAR(32) NOT NULL DEFAULT 'remote_git',
    repository_url VARCHAR(1024) NOT NULL,
    git_credential_id VARCHAR(26),
    variable_overrides VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    default_branch VARCHAR(255) NOT NULL DEFAULT 'master',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26)
);

CREATE INDEX idx_repository_name ON repository(name);
CREATE INDEX idx_repository_code ON repository(code);
CREATE INDEX idx_repository_credential ON repository(git_credential_id);
CREATE INDEX idx_repository_project ON repository(project_id);
