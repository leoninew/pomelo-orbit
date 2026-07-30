-- Domain: service
-- Tables: service, service_expose
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    version_id TEXT NOT NULL,
    runtime_config_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    UNIQUE(application_id, instance_key)
);

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_application ON service(application_id);

CREATE TABLE IF NOT EXISTS service_expose (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('http', 'tcp')),
    container_port INTEGER NOT NULL CHECK (container_port BETWEEN 1 AND 65535),
    path_prefix TEXT,
    access TEXT NOT NULL CHECK (access IN ('local', 'public')),
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    UNIQUE(service_id, component_name, protocol, container_port)
);

CREATE INDEX IF NOT EXISTS idx_service_expose_service ON service_expose(service_id);
