-- Reverse 000013: restore tcp_entrypoint; drop expose access/listen_port.

CREATE TABLE gateway_config_old (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    base_domain TEXT NOT NULL,
    image TEXT,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tcp_entrypoint TEXT,
    tls_mode TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

INSERT INTO gateway_config_old (
    application_id, rest_api_url, base_domain, image,
    default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at
)
SELECT
    application_id, rest_api_url, base_domain, image,
    default_entrypoint, NULL, tls_mode, created_at, updated_at
FROM gateway_config;

DROP TABLE gateway_config;
ALTER TABLE gateway_config_old RENAME TO gateway_config;

CREATE TABLE version_expose_old (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    path_prefix TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, component_name, protocol, container_port)
);

INSERT INTO version_expose_old (
    id, version_id, component_name, protocol, container_port, path_prefix, created_at, updated_at
)
SELECT
    id, version_id, component_name, protocol, container_port, path_prefix, created_at, updated_at
FROM version_expose;

DROP TABLE version_expose;
ALTER TABLE version_expose_old RENAME TO version_expose;
CREATE INDEX IF NOT EXISTS idx_version_expose_version ON version_expose(version_id);
