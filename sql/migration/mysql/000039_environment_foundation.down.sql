DROP TABLE environment;
ALTER TABLE deployment DROP COLUMN environment_target_type;
ALTER TABLE deployment DROP COLUMN gateway_application_id;
ALTER TABLE deployment DROP COLUMN ssh_credential_revision;
ALTER TABLE deployment DROP COLUMN ssh_credential_id;
ALTER TABLE deployment DROP COLUMN environment_target_revision;
ALTER TABLE deployment DROP COLUMN environment_id;
ALTER TABLE credential DROP COLUMN revision;
