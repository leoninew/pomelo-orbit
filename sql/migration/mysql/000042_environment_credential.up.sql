CREATE TABLE environment_credential (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL,
    public_key TEXT NOT NULL,
    encrypted_private_key LONGTEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_environment_credential_project FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_environment_credential_project ON environment_credential(project_id);

RENAME TABLE credential TO repository_credential;
