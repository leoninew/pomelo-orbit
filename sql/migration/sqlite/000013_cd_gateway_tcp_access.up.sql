-- Expose access/listen_port; drop gateway tcp_entrypoint (public TCP uses dynamic entryPoints).

ALTER TABLE version_expose ADD COLUMN access TEXT NOT NULL DEFAULT 'public';
ALTER TABLE version_expose ADD COLUMN listen_port INTEGER;

CREATE TABLE gateway_config_new (
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

INSERT INTO gateway_config_new (
    application_id, rest_api_url, base_domain, image,
    default_entrypoint, tls_mode, created_at, updated_at
)
SELECT
    application_id, rest_api_url, base_domain, image,
    COALESCE(NULLIF(TRIM(default_entrypoint), ''), 'web'),
    COALESCE(NULLIF(TRIM(tls_mode), ''), 'none'),
    created_at, updated_at
FROM gateway_config;

DROP TABLE gateway_config;
ALTER TABLE gateway_config_new RENAME TO gateway_config;
