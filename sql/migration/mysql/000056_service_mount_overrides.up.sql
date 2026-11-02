ALTER TABLE service_component_mount DROP CHECK chk_service_component_mount_state;
UPDATE version_component_mount SET read_only = 0 WHERE source_type = 'controlled_file';
ALTER TABLE service_component_mount ADD COLUMN content LONGTEXT NULL, ADD COLUMN mode VARCHAR(4) NULL, ADD COLUMN ignore_if_exists TINYINT(1) NULL, ADD COLUMN read_only TINYINT(1) NULL;
ALTER TABLE service_component_mount ADD CONSTRAINT chk_service_component_mount_state CHECK (state IN ('override', 'deleted'));
