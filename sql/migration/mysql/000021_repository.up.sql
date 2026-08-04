-- Domain: repository
-- Tables: repository, repository_webhook
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
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);

CREATE INDEX idx_repository_name ON repository(name);
CREATE INDEX idx_repository_code ON repository(code);
CREATE INDEX idx_repository_credential ON repository(git_credential_id);
CREATE INDEX idx_repository_project ON repository(project_id);

CREATE TABLE IF NOT EXISTS repository_webhook (
    id VARCHAR(26) PRIMARY KEY,
    repository_id VARCHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    template_id VARCHAR(26) NOT NULL,
    branch_filter VARCHAR(255),
    encrypted_secret LONGTEXT NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE RESTRICT
);

CREATE INDEX idx_repository_webhook_repository ON repository_webhook(repository_id);
