-- Domain: route
-- Tables: route
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS route (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL DEFAULT 'http',
    domain VARCHAR(255) NOT NULL,
    path_prefix VARCHAR(512) NOT NULL,
    target_url VARCHAR(1024) NOT NULL,
    listen_port INT NULL,
    service_id VARCHAR(26) NULL,
    component_name VARCHAR(255) NULL,
    endpoint_protocol VARCHAR(16) NULL,
    endpoint_container_port INT NULL,
    enabled TINYINT(1) NOT NULL,
    https_enabled TINYINT(1) NOT NULL DEFAULT 0,
    cert_pem LONGTEXT,
    cert_key LONGTEXT,
    cert_type VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    CONSTRAINT chk_route_protocol CHECK (protocol IN ('http', 'tcp')),
    CONSTRAINT chk_route_listen_port CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_route_endpoint_protocol CHECK (endpoint_protocol IS NULL OR endpoint_protocol IN ('http', 'tcp')),
    CONSTRAINT chk_route_endpoint_container_port CHECK (endpoint_container_port IS NULL OR endpoint_container_port BETWEEN 1 AND 65535)
);

CREATE INDEX idx_route_domain ON route(domain);
CREATE INDEX idx_route_enabled ON route(enabled);
CREATE INDEX idx_route_project ON route(project_id);
CREATE INDEX idx_route_tcp_listen ON route(protocol, listen_port);
