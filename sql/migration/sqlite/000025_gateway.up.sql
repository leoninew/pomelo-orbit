-- Domain: gateway
-- Tables: gateway_config
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md
-- Note: FK application_id is deploy carrier; logical domain remains gateway.

CREATE TABLE IF NOT EXISTS gateway_config (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    base_domain TEXT NOT NULL,
    image TEXT,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tls_mode TEXT NOT NULL DEFAULT 'none',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);
