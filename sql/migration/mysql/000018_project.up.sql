-- Domain: project
-- Tables: project, project_member
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS project (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    is_active TINYINT(1) NOT NULL DEFAULT 1
);

CREATE INDEX idx_project_code ON project(code);

CREATE TABLE IF NOT EXISTS project_member (
    project_id VARCHAR(26) NOT NULL,
    user_id VARCHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (project_id, user_id),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);
CREATE INDEX idx_project_member_user_id ON project_member(user_id);
