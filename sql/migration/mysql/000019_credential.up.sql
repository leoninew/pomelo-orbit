-- Domain: credential
-- Tables: credential
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS credential (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    encrypted_data LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26)
);

CREATE INDEX idx_credential_name ON credential(name);
CREATE INDEX idx_credential_project ON credential(project_id);
