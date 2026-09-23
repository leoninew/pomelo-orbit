ALTER TABLE pipeline_run ADD COLUMN environment_id TEXT;
ALTER TABLE pipeline_run ADD COLUMN environment_target_type TEXT;
ALTER TABLE pipeline_run ADD COLUMN environment_target_revision INTEGER;
ALTER TABLE pipeline_run ADD COLUMN ssh_credential_id TEXT;
ALTER TABLE pipeline_run ADD COLUMN ssh_credential_revision INTEGER;
