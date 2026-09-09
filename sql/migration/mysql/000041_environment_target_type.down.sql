ALTER TABLE deployment DROP COLUMN environment_target_type;
ALTER TABLE environment DROP CONSTRAINT chk_environment_target_type;
ALTER TABLE environment DROP COLUMN target_type;
