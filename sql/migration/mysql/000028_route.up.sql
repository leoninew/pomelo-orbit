-- Domain: route
-- Tables: route
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS route (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    path_prefix VARCHAR(512) NOT NULL,
    target_url VARCHAR(1024) NOT NULL,
    enabled TINYINT(1) NOT NULL,
    https_enabled TINYINT(1) NOT NULL DEFAULT 0,
    cert_pem LONGTEXT,
    cert_key LONGTEXT,
    cert_type VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_route_domain ON route(domain);
CREATE INDEX idx_route_enabled ON route(enabled);
CREATE INDEX idx_route_project ON route(project_id);
