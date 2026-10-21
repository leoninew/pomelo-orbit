BEGIN IMMEDIATE;
-- Restores current bindings only; unbound historical keys require the pre-upgrade backup.
ALTER TABLE environment ADD COLUMN ssh_credential_id TEXT;
ALTER TABLE environment ADD COLUMN ssh_credential_revision BIGINT CHECK (ssh_credential_revision IS NULL OR ssh_credential_revision >= 1);
UPDATE environment SET
    ssh_credential_id = (SELECT c.id FROM environment_credential c WHERE c.environment_id = environment.id),
    ssh_credential_revision = (SELECT c.revision FROM environment_credential c WHERE c.environment_id = environment.id)
WHERE target_type = 'ssh';

CREATE TABLE environment_credential_new (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    public_key TEXT NOT NULL,
    encrypted_private_key TEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES project(id)
);
INSERT INTO environment_credential_new (id, project_id, public_key, encrypted_private_key, revision, created_at)
SELECT c.id, e.project_id, c.public_key, c.encrypted_private_key, c.revision, c.created_at
FROM environment_credential c JOIN environment e ON e.id = c.environment_id;
DROP TABLE environment_credential;
ALTER TABLE environment_credential_new RENAME TO environment_credential;
CREATE INDEX idx_environment_credential_project ON environment_credential(project_id);
COMMIT;
