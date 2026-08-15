-- Domain: service
-- Tables: service, service_env, service_component
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    instance_key VARCHAR(100) NOT NULL DEFAULT 'default',
    code VARCHAR(63) NOT NULL,
    version_id VARCHAR(26) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_service_application_instance (application_id, instance_key),
    UNIQUE KEY uq_service_code (code),
    KEY idx_service_version (version_id),
    KEY idx_service_application (application_id),
    CONSTRAINT fk_service_application FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_version FOREIGN KEY (version_id) REFERENCES version(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_env (
    service_id VARCHAR(26) NOT NULL,
    env_key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (service_id, env_key),
    CONSTRAINT fk_service_env_service FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_component (
    id VARCHAR(128) NOT NULL PRIMARY KEY,
    service_id VARCHAR(26) NOT NULL,
    source_version_component_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    entrypoint_json TEXT NULL,
    command_json TEXT NULL,
    pull_policy VARCHAR(32) NULL,
    restart_policy VARCHAR(32) NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_service_component_source (service_id, source_version_component_id),
    UNIQUE KEY uq_service_component_name (service_id, component_name),
    KEY idx_service_component_service (service_id),
    CONSTRAINT fk_service_component_service FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_component_source FOREIGN KEY (source_version_component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_component_pull_policy CHECK (pull_policy IS NULL OR pull_policy IN ('always', 'missing', 'never')),
    CONSTRAINT chk_service_component_restart_policy CHECK (restart_policy IS NULL OR restart_policy IN ('no', 'unless-stopped')),
    CONSTRAINT chk_service_component_status CHECK (status = 'active')
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_component_env (
    service_component_id VARCHAR(128) NOT NULL,
    env_key VARCHAR(255) NOT NULL,
    value TEXT NULL,
    state VARCHAR(16) NOT NULL,
    PRIMARY KEY (service_component_id, env_key),
    CONSTRAINT fk_service_component_env_component FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_component_env_state CHECK (
        (state = 'override' AND value IS NOT NULL)
        OR (state = 'deleted' AND value IS NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_component_mount (
    id VARCHAR(160) NOT NULL PRIMARY KEY,
    service_component_id VARCHAR(128) NOT NULL,
    target VARCHAR(1024) NOT NULL,
    source VARCHAR(1024) NULL,
    source_is_host_path TINYINT(1) NULL,
    state VARCHAR(16) NOT NULL,
    KEY idx_service_component_mount_component (service_component_id),
    CONSTRAINT fk_service_component_mount_component FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_component_mount_state CHECK (
        (state = 'override' AND source IS NOT NULL)
        OR (state = 'deleted' AND source IS NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_component_resource (
    service_component_id VARCHAR(128) NOT NULL PRIMARY KEY,
    limit_cpus VARCHAR(64) NULL,
    limit_memory VARCHAR(64) NULL,
    reservation_cpus VARCHAR(64) NULL,
    reservation_memory VARCHAR(64) NULL,
    state VARCHAR(16) NOT NULL,
    CONSTRAINT fk_service_component_resource_component FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_component_resource_state CHECK (
        (state = 'deleted' AND limit_cpus IS NULL AND limit_memory IS NULL AND reservation_cpus IS NULL AND reservation_memory IS NULL)
        OR (state = 'override' AND (limit_cpus IS NOT NULL OR limit_memory IS NOT NULL OR reservation_cpus IS NOT NULL OR reservation_memory IS NOT NULL))
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_component_endpoint (
    id VARCHAR(26) NOT NULL,
    service_component_id VARCHAR(128) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    mode VARCHAR(32) NULL,
    bind_address VARCHAR(255) NULL,
    listen_port INT NULL,
    entrypoint VARCHAR(128) NULL,
    path_prefix VARCHAR(512) NULL,
    state VARCHAR(16) NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_service_component_endpoint_contract (service_component_id, protocol, container_port),
    KEY idx_service_component_endpoint_listen (listen_port),
    CONSTRAINT fk_service_component_endpoint_component FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_component_endpoint_protocol CHECK (protocol IN ('http', 'tcp')),
    CONSTRAINT chk_service_component_endpoint_container_port CHECK (container_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_service_component_endpoint_mode CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway')),
    CONSTRAINT chk_service_component_endpoint_listen CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_service_component_endpoint_state CHECK (
        (state = 'deleted' AND mode IS NULL AND bind_address IS NULL AND listen_port IS NULL AND entrypoint IS NULL AND path_prefix IS NULL)
        OR (state = 'override' AND (mode IS NOT NULL OR bind_address IS NOT NULL OR listen_port IS NOT NULL OR entrypoint IS NOT NULL OR path_prefix IS NOT NULL))
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
