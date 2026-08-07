-- sqlc target-schema extension for VersionComponent artifact display snapshots.
-- The separate database cutover must apply the corresponding table rebuild and
-- remove the legacy artifact foreign key.
ALTER TABLE version_component ADD COLUMN artifact_name TEXT;
ALTER TABLE version_component ADD COLUMN artifact_image_ref TEXT;
ALTER TABLE version_component ADD COLUMN artifact_local_image_sha256 TEXT;
ALTER TABLE version_component ADD COLUMN artifact_source_commit_sha TEXT;
