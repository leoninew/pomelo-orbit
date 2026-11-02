ALTER TABLE service_component_mount DROP CONSTRAINT service_component_mount_check;
UPDATE version_component_mount SET read_only = 0 WHERE source_type = 'controlled_file';
ALTER TABLE service_component_mount ADD COLUMN content TEXT, ADD COLUMN mode TEXT, ADD COLUMN ignore_if_exists INTEGER, ADD COLUMN read_only INTEGER;
