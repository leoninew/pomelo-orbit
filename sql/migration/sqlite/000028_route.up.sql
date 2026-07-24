-- Domain: route
-- Tables: route
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS route (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    domain TEXT NOT NULL,
    path_prefix TEXT NOT NULL,
    target_url TEXT NOT NULL,
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
