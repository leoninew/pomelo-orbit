-- Reject inconsistent current bindings before replacing any business table.
CREATE TEMPORARY TABLE environment_credential_ownership_guard (id INTEGER PRIMARY KEY);
INSERT INTO environment_credential_ownership_guard VALUES (1);
INSERT INTO environment_credential_ownership_guard
SELECT 1 WHERE EXISTS (
    SELECT 1 FROM environment e
    LEFT JOIN environment_credential c ON c.id = e.ssh_credential_id
    WHERE (e.ssh_credential_id IS NULL AND e.ssh_credential_revision IS NOT NULL)
       OR (e.ssh_credential_id IS NOT NULL AND (
           e.target_type <> 'ssh' OR c.id IS NULL OR c.project_id <> e.project_id
           OR e.ssh_credential_revision IS NULL OR c.revision <> e.ssh_credential_revision
           OR c.revision < 1
       ))
);
DROP TABLE environment_credential_ownership_guard;

CREATE TABLE environment_credential_new (
    id VARCHAR(26) PRIMARY KEY,
    environment_id VARCHAR(26) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    encrypted_private_key LONGTEXT NOT NULL,
    revision BIGINT NOT NULL DEFAULT 1 CHECK (revision >= 1),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (environment_id) REFERENCES environment(id) ON DELETE RESTRICT
);

-- Only currently bound keys belong to the new aggregate. Historical task snapshots remain untouched.
INSERT INTO environment_credential_new (id, environment_id, public_key, encrypted_private_key, revision, created_at)
SELECT c.id, e.id, c.public_key, c.encrypted_private_key, c.revision, c.created_at
FROM environment e JOIN environment_credential c ON c.id = e.ssh_credential_id
WHERE e.target_type = 'ssh';

DROP TABLE environment_credential;
RENAME TABLE environment_credential_new TO environment_credential;
ALTER TABLE environment DROP CHECK chk_environment_revision;
ALTER TABLE environment ADD CONSTRAINT chk_environment_revision CHECK (target_revision >= 1);
ALTER TABLE environment DROP COLUMN ssh_credential_id;
ALTER TABLE environment DROP COLUMN ssh_credential_revision;
