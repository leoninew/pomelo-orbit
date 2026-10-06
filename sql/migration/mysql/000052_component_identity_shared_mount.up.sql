ALTER TABLE version_component ADD COLUMN container_user TEXT;
ALTER TABLE version_component ADD COLUMN group_add_json TEXT;
ALTER TABLE service_component ADD COLUMN container_user TEXT;
ALTER TABLE service_component ADD COLUMN group_add_json TEXT;
ALTER TABLE version_component_mount ADD COLUMN shared BIGINT NOT NULL DEFAULT 0 CHECK (shared IN (0, 1));
