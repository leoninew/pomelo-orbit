ALTER TABLE repository_credential
    ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;

UPDATE repository_credential SET updated_at = created_at;
