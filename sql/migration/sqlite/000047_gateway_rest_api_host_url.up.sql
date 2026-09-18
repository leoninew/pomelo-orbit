PRAGMA foreign_keys = OFF;

CREATE TABLE gateway_config_new (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    rest_api_host_url TEXT NOT NULL,
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

INSERT INTO gateway_config_new (
    application_id, rest_api_url, rest_api_host_url, rest_ready_timeout_seconds,
    base_domain, default_entrypoint, tls_mode, acme_profile, acme_email,
    dns_api_token, created_at, updated_at
)
SELECT
    application_id, rest_api_url, 'http://127.0.0.1:8080', rest_ready_timeout_seconds,
    base_domain, default_entrypoint, tls_mode, acme_profile, acme_email,
    dns_api_token, created_at, updated_at
FROM gateway_config;

DROP TABLE gateway_config;
ALTER TABLE gateway_config_new RENAME TO gateway_config;

DELETE FROM route
WHERE id IN (
    SELECT r.id
    FROM route r
    INNER JOIN application a ON a.project_id = r.project_id
    INNER JOIN gateway_config gc ON gc.application_id = a.id
    WHERE r.name = 'traefik'
      AND r.protocol = 'http'
      AND r.domain = 'traefik-dashboard.' || gc.base_domain
      AND r.path_prefix = '/'
      AND r.target_url = 'http://' || lower(a.code) || '-traefik:8080'
      AND r.service_id IS NULL
      AND r.component_name IS NULL
      AND r.endpoint_protocol IS NULL
      AND r.endpoint_container_port IS NULL
      AND r.enabled = 0
      AND r.https_enabled = 0
      AND r.cert_type = 'manual'
      AND r.acme_challenge = 'http'
);

PRAGMA foreign_keys = ON;
