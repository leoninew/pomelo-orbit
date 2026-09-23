ALTER TABLE pipeline_run
    DROP COLUMN ssh_credential_revision,
    DROP COLUMN ssh_credential_id,
    DROP COLUMN environment_target_revision,
    DROP COLUMN environment_target_type,
    DROP COLUMN environment_id;
