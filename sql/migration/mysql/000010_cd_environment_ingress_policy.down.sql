-- Down: recreate empty environment_binding; drop policy columns.

CREATE TABLE IF NOT EXISTS environment_binding (
    id VARCHAR(26) PRIMARY KEY,
    environment_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    domains_json LONGTEXT NOT NULL,
    entrypoint VARCHAR(128) NOT NULL,
    tls_mode VARCHAR(32) NOT NULL DEFAULT 'none',
    sni_host VARCHAR(255) NULL,
    note TEXT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_environment_binding (environment_id, component_name, protocol, container_port),
    KEY idx_environment_binding_env (environment_id),
    CONSTRAINT fk_environment_binding_env FOREIGN KEY (environment_id) REFERENCES environment(id) ON DELETE CASCADE
);

ALTER TABLE environment
    DROP COLUMN base_domain,
    DROP COLUMN domain_template,
    DROP COLUMN default_entrypoint,
    DROP COLUMN tcp_entrypoint,
    DROP COLUMN tls_mode;
