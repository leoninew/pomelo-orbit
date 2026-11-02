ALTER TABLE service_component_mount DROP CHECK chk_service_component_mount_state;
ALTER TABLE service_component_mount DROP COLUMN content, DROP COLUMN mode, DROP COLUMN ignore_if_exists, DROP COLUMN read_only;
ALTER TABLE service_component_mount ADD CONSTRAINT chk_service_component_mount_state CHECK ((state = 'override' AND source IS NOT NULL) OR (state = 'deleted' AND source IS NULL));
