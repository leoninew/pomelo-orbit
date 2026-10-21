-- Restores current bindings only; unbound historical keys require the pre-upgrade backup.
ALTER TABLE environment ADD COLUMN ssh_credential_id VARCHAR(26);
ALTER TABLE environment ADD COLUMN ssh_credential_revision BIGINT;
UPDATE environment SET
    ssh_credential_id = (SELECT c.id FROM environment_credential c WHERE c.environment_id = environment.id),
    ssh_credential_revision = (SELECT c.revision FROM environment_credential c WHERE c.environment_id = environment.id)
WHERE target_type = 'ssh';
ALTER TABLE environment DROP CHECK chk_environment_revision;
ALTER TABLE environment ADD CONSTRAINT chk_environment_revision CHECK ((ssh_credential_revision IS NULL OR ssh_credential_revision >= 1) AND target_revision >= 1);

CREATE TABLE environment_credential_new (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL,
    public_key TEXT NOT NULL,
    encrypted_private_key LONGTEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (project_id) REFERENCES project(id)
);
INSERT INTO environment_credential_new (id, project_id, public_key, encrypted_private_key, revision, created_at)
SELECT c.id, e.project_id, c.public_key, c.encrypted_private_key, c.revision, c.created_at
FROM environment_credential c JOIN environment e ON e.id = c.environment_id;
DROP TABLE environment_credential;
RENAME TABLE environment_credential_new TO environment_credential;
CREATE INDEX idx_environment_credential_project ON environment_credential(project_id);
