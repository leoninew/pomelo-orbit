-- Domain: gateway
-- Tables: gateway_config
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md
-- Note: FK application_id is deploy carrier; logical domain remains gateway.

CREATE TABLE IF NOT EXISTS gateway_config (
    application_id VARCHAR(26) NOT NULL PRIMARY KEY,
    rest_api_url VARCHAR(512) NOT NULL,
    base_domain VARCHAR(255) NOT NULL,
    default_entrypoint VARCHAR(128) NOT NULL DEFAULT 'web',
    tls_mode VARCHAR(32) NOT NULL DEFAULT 'none',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;