PRAGMA foreign_keys = OFF;

CREATE TABLE gateway_config_old (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    rest_ready_timeout_seconds INTEGER NOT NULL DEFAULT 20,
    base_domain TEXT NOT NULL,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tls_mode TEXT NOT NULL DEFAULT 'none',
    acme_profile TEXT NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns')),
    acme_email TEXT NOT NULL DEFAULT '',
    dns_api_token TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id)
);

INSERT INTO gateway_config_old (
    application_id, rest_api_url, rest_ready_timeout_seconds, base_domain,
    default_entrypoint, tls_mode, acme_profile, acme_email, dns_api_token,
    created_at, updated_at
)
SELECT
    application_id, rest_api_url, rest_ready_timeout_seconds, base_domain,
    default_entrypoint, tls_mode, acme_profile, acme_email, dns_api_token,
    created_at, updated_at
FROM gateway_config;

DROP TABLE gateway_config;
ALTER TABLE gateway_config_old RENAME TO gateway_config;

PRAGMA foreign_keys = ON;
