ALTER TABLE gateway_config ADD COLUMN traefik_component_name TEXT NOT NULL DEFAULT 'traefik';
ALTER TABLE gateway_config ADD COLUMN rest_ready_timeout_seconds INTEGER NOT NULL DEFAULT 20;
ALTER TABLE gateway_config ADD COLUMN acme_profile TEXT NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns'));
ALTER TABLE gateway_config ADD COLUMN acme_email TEXT NOT NULL DEFAULT '';
ALTER TABLE gateway_config ADD COLUMN dns_api_token TEXT NOT NULL DEFAULT '';

CREATE TABLE gateway_acme_profile_version (
    application_id TEXT NOT NULL,
    profile TEXT NOT NULL CHECK (profile IN ('base', 'http', 'dns', 'http-dns')),
    version_id TEXT NOT NULL,
    PRIMARY KEY (application_id, profile),
    UNIQUE (application_id, version_id),
    FOREIGN KEY (application_id) REFERENCES gateway_config(application_id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
);
