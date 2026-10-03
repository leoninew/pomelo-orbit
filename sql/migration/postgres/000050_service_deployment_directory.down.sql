ALTER TABLE deployment DROP COLUMN working_directory;
ALTER TABLE service DROP COLUMN runtime_target_revision;
ALTER TABLE service DROP COLUMN runtime_directory;
ALTER TABLE service DROP COLUMN directory_target_revision;
ALTER TABLE service DROP COLUMN deployment_directory;
