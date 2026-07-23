-- Reverse 000013.

ALTER TABLE gateway_config
    ADD COLUMN tcp_entrypoint VARCHAR(128) NULL;

ALTER TABLE version_expose
    DROP COLUMN access,
    DROP COLUMN listen_port;
