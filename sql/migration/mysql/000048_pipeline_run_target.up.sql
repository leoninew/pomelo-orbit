ALTER TABLE pipeline_run
    ADD COLUMN environment_id TEXT,
    ADD COLUMN environment_target_type TEXT,
    ADD COLUMN environment_target_revision BIGINT,
    ADD COLUMN ssh_credential_id TEXT,
    ADD COLUMN ssh_credential_revision BIGINT;
