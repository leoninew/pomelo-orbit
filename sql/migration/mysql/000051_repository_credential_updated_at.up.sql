ALTER TABLE repository_credential
    ADD COLUMN updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3);

UPDATE repository_credential SET updated_at = created_at;
