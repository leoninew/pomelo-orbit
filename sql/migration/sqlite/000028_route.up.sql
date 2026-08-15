-- Domain: route
-- Tables: route
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS route (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'http' CHECK (protocol IN ('http', 'tcp')),
    domain TEXT NOT NULL,
    path_prefix TEXT NOT NULL,
    target_url TEXT NOT NULL,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    service_id TEXT,
    component_name TEXT,
    endpoint_protocol TEXT,
    endpoint_container_port INTEGER CHECK (endpoint_container_port IS NULL OR endpoint_container_port BETWEEN 1 AND 65535),
    enabled INTEGER NOT NULL,
    https_enabled INTEGER NOT NULL DEFAULT 0,
    cert_pem TEXT,
    cert_key TEXT,
    cert_type TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_route_domain ON route(domain);
CREATE INDEX IF NOT EXISTS idx_route_enabled ON route(enabled);
CREATE INDEX IF NOT EXISTS idx_route_project ON route(project_id);
CREATE INDEX IF NOT EXISTS idx_route_tcp_listen ON route(protocol, listen_port);
