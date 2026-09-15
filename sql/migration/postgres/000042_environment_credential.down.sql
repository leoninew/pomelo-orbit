ALTER INDEX IF EXISTS idx_repository_credential_name RENAME TO idx_credential_name;
ALTER INDEX IF EXISTS idx_repository_credential_project RENAME TO idx_credential_project;
ALTER TABLE repository_credential RENAME TO credential;
DROP TABLE IF EXISTS environment_credential;
