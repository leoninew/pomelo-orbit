CREATE TABLE environment_credential (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES project(id),
    public_key TEXT NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_environment_credential_project ON environment_credential(project_id);

ALTER TABLE credential RENAME TO repository_credential;
ALTER INDEX IF EXISTS idx_credential_name RENAME TO idx_repository_credential_name;
ALTER INDEX IF EXISTS idx_credential_project RENAME TO idx_repository_credential_project;
