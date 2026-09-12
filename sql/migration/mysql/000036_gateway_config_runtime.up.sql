ALTER TABLE gateway_config
    ADD COLUMN traefik_component_name VARCHAR(255) NOT NULL DEFAULT 'traefik',
    ADD COLUMN rest_ready_timeout_seconds BIGINT NOT NULL DEFAULT 20,
    ADD COLUMN acme_profile VARCHAR(16) NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns')),
    ADD COLUMN acme_email VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN dns_api_token VARCHAR(255) NOT NULL DEFAULT '';

CREATE TABLE gateway_acme_profile_version (
    application_id VARCHAR(26) NOT NULL,
    profile VARCHAR(16) NOT NULL,
    version_id VARCHAR(26) NOT NULL,
    PRIMARY KEY (application_id, profile),
    UNIQUE (application_id, version_id),
    CONSTRAINT chk_gateway_acme_profile_version_profile CHECK (profile IN ('base', 'http', 'dns', 'http-dns'))
);
