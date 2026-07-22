-- Environment IngressPolicy columns; drop environment_binding.

ALTER TABLE environment
    ADD COLUMN base_domain VARCHAR(255) NOT NULL DEFAULT 'local.test',
    ADD COLUMN domain_template VARCHAR(512) NULL,
    ADD COLUMN default_entrypoint VARCHAR(128) NOT NULL DEFAULT 'web',
    ADD COLUMN tcp_entrypoint VARCHAR(128) NULL,
    ADD COLUMN tls_mode VARCHAR(32) NOT NULL DEFAULT 'none';

UPDATE environment
SET
    base_domain = IF(base_domain IS NULL OR TRIM(base_domain) = '', 'local.test', base_domain),
    default_entrypoint = IF(default_entrypoint IS NULL OR TRIM(default_entrypoint) = '', 'web', default_entrypoint),
    tls_mode = IF(tls_mode IS NULL OR TRIM(tls_mode) = '', 'none', tls_mode);

DROP TABLE IF EXISTS environment_binding;
