-- Move Traefik entrypoint/TLS policy from environment to gateway_config.

ALTER TABLE gateway_config
    ADD COLUMN default_entrypoint VARCHAR(128) NOT NULL DEFAULT 'web',
    ADD COLUMN tcp_entrypoint VARCHAR(128) NULL,
    ADD COLUMN tls_mode VARCHAR(32) NOT NULL DEFAULT 'none';

UPDATE gateway_config
SET
    default_entrypoint = IF(default_entrypoint IS NULL OR TRIM(default_entrypoint) = '', 'web', default_entrypoint),
    tls_mode = IF(tls_mode IS NULL OR TRIM(tls_mode) = '', 'none', tls_mode);

ALTER TABLE environment
    DROP COLUMN default_entrypoint,
    DROP COLUMN tcp_entrypoint,
    DROP COLUMN tls_mode;
