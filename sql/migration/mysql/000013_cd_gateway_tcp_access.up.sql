-- Expose access/listen_port; drop gateway tcp_entrypoint (public TCP uses dynamic entryPoints).

ALTER TABLE version_expose
    ADD COLUMN access VARCHAR(16) NOT NULL DEFAULT 'public',
    ADD COLUMN listen_port INT NULL;

ALTER TABLE gateway_config
    DROP COLUMN tcp_entrypoint;
