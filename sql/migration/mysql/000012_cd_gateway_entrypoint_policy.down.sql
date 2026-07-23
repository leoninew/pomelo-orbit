-- Restore entrypoint/TLS policy columns on environment; drop from gateway_config.

ALTER TABLE environment
    ADD COLUMN default_entrypoint VARCHAR(128) NOT NULL DEFAULT 'web',
    ADD COLUMN tcp_entrypoint VARCHAR(128) NULL,
    ADD COLUMN tls_mode VARCHAR(32) NOT NULL DEFAULT 'none';

UPDATE environment
SET
    default_entrypoint = 'web',
    tls_mode = 'none';

ALTER TABLE gateway_config
    DROP COLUMN default_entrypoint,
    DROP COLUMN tcp_entrypoint,
    DROP COLUMN tls_mode;
