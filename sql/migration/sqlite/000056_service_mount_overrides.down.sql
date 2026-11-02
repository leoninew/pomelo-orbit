CREATE TABLE service_component_mount_old (
    id TEXT PRIMARY KEY,
    service_component_id TEXT NOT NULL,
    target TEXT NOT NULL,
    source TEXT,
    source_is_host_path INTEGER,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK ((state = 'override' AND source IS NOT NULL) OR (state = 'deleted' AND source IS NULL))
);
INSERT INTO service_component_mount_old (id, service_component_id, target, source, source_is_host_path, state)
SELECT id, service_component_id, target, source, source_is_host_path, state FROM service_component_mount;
DROP TABLE service_component_mount;
ALTER TABLE service_component_mount_old RENAME TO service_component_mount;
CREATE INDEX idx_service_component_mount_component ON service_component_mount(service_component_id);
