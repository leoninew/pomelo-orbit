ALTER TABLE route DROP COLUMN endpoint_container_port;
ALTER TABLE route DROP COLUMN endpoint_protocol;
ALTER TABLE route ADD COLUMN endpoint_name TEXT;

PRAGMA foreign_keys = OFF;

CREATE TABLE version_component_endpoint_old (
    component_id TEXT NOT NULL,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('http', 'tcp')),
    container_port INTEGER NOT NULL CHECK (container_port BETWEEN 1 AND 65535),
    mode TEXT NOT NULL DEFAULT 'internal' CHECK (mode IN ('internal', 'local', 'host', 'gateway', 'tcp')),
    bind_address TEXT,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    entrypoint TEXT,
    path_prefix TEXT,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, name),
    UNIQUE (component_id, position),
    UNIQUE (component_id, protocol, container_port),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);
INSERT INTO version_component_endpoint_old
SELECT component_id, protocol || container_port, protocol, container_port, mode, bind_address, listen_port, entrypoint, path_prefix, position
FROM version_component_endpoint;
DROP TABLE version_component_endpoint;
ALTER TABLE version_component_endpoint_old RENAME TO version_component_endpoint;

CREATE TABLE service_component_endpoint_old (
    service_component_id TEXT NOT NULL,
    name TEXT NOT NULL,
    mode TEXT CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway', 'tcp')),
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
INSERT INTO service_component_endpoint_old
SELECT service_component_id, protocol || container_port, mode, bind_address, listen_port, entrypoint, path_prefix, state
FROM service_component_endpoint;
DROP TABLE service_component_endpoint;
ALTER TABLE service_component_endpoint_old RENAME TO service_component_endpoint;
CREATE INDEX idx_service_component_endpoint_listen ON service_component_endpoint(listen_port);

PRAGMA foreign_keys = ON;
