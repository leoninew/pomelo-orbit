ALTER TABLE gateway_config
    ADD COLUMN traefik_component_name TEXT NOT NULL DEFAULT 'traefik',
    ADD COLUMN rest_ready_timeout_seconds BIGINT NOT NULL DEFAULT 20,
    ADD COLUMN acme_profile TEXT NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns')),
    ADD COLUMN acme_email TEXT NOT NULL DEFAULT '',
    ADD COLUMN dns_api_token TEXT NOT NULL DEFAULT '';

CREATE TABLE gateway_acme_profile_version (
    application_id TEXT NOT NULL,
    profile TEXT NOT NULL,
    version_id TEXT NOT NULL,
    PRIMARY KEY (application_id, profile),
    UNIQUE (application_id, version_id),
    CONSTRAINT chk_gateway_acme_profile_version_profile CHECK (profile IN ('base', 'http', 'dns', 'http-dns')),
    FOREIGN KEY (application_id) REFERENCES gateway_config(application_id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
);
