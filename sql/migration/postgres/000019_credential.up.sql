-- Domain: credential
-- Tables: credential
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS credential (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    project_id TEXT REFERENCES project(id),
    CONSTRAINT uq_repository_credential_name UNIQUE (name)
);
