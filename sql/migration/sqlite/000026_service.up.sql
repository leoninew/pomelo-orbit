-- Domain: service
-- Tables: service, service_env, service_component
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    UNIQUE(application_id, instance_key)
);

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_application ON service(application_id);

CREATE TABLE IF NOT EXISTS service_env (
    service_id TEXT NOT NULL,
    env_key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (service_id, env_key),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS service_component (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    source_version_component_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (service_id, source_version_component_id),
    UNIQUE (service_id, component_name),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    FOREIGN KEY (source_version_component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_component_service ON service_component(service_id);

CREATE TABLE IF NOT EXISTS service_component_env (
    service_component_id TEXT NOT NULL,
    env_key TEXT NOT NULL,
    value TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    PRIMARY KEY (service_component_id, env_key),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'override' AND value IS NOT NULL)
        OR (state = 'deleted' AND value IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS service_component_mount (
    id TEXT PRIMARY KEY,
    service_component_id TEXT NOT NULL,
    target TEXT NOT NULL,
    source TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'override' AND source IS NOT NULL)
        OR (state = 'deleted' AND source IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_service_component_mount_component ON service_component_mount(service_component_id);

CREATE TABLE IF NOT EXISTS service_component_resource (
    service_component_id TEXT PRIMARY KEY,
    limit_cpus TEXT,
    limit_memory TEXT,
    reservation_cpus TEXT,
    reservation_memory TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'deleted' AND limit_cpus IS NULL AND limit_memory IS NULL AND reservation_cpus IS NULL AND reservation_memory IS NULL)
        OR (state = 'override' AND (limit_cpus IS NOT NULL OR limit_memory IS NOT NULL OR reservation_cpus IS NOT NULL OR reservation_memory IS NOT NULL))
    )
);

CREATE TABLE IF NOT EXISTS service_component_endpoint (
    service_component_id TEXT NOT NULL,
    name TEXT NOT NULL,
    mode TEXT CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp')),
    bind_address TEXT,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    entrypoint TEXT,
    path_prefix TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    PRIMARY KEY (service_component_id, name),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'deleted' AND mode IS NULL AND bind_address IS NULL AND listen_port IS NULL AND entrypoint IS NULL AND path_prefix IS NULL)
        OR (state = 'override' AND (mode IS NOT NULL OR bind_address IS NOT NULL OR listen_port IS NOT NULL OR entrypoint IS NOT NULL OR path_prefix IS NOT NULL))
    )
);

CREATE INDEX IF NOT EXISTS idx_service_component_endpoint_listen ON service_component_endpoint(listen_port);
