-- Domain: service
-- Tables: service, service_expose
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    instance_key VARCHAR(100) NOT NULL DEFAULT 'default',
    version_id VARCHAR(26) NOT NULL,
    runtime_config_json LONGTEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_service_application_instance (application_id, instance_key),
    KEY idx_service_version (version_id),
    KEY idx_service_application (application_id),
    CONSTRAINT fk_service_application FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_version FOREIGN KEY (version_id) REFERENCES version(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS service_expose (
    id VARCHAR(26) PRIMARY KEY,
    service_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    path_prefix VARCHAR(512) NULL,
    access VARCHAR(16) NOT NULL,
    listen_port INT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_service_expose_key (service_id, component_name, protocol, container_port),
    KEY idx_service_expose_service (service_id),
    CONSTRAINT fk_service_expose_service FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    CONSTRAINT chk_service_expose_protocol CHECK (protocol IN ('http', 'tcp')),
    CONSTRAINT chk_service_expose_container_port CHECK (container_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_service_expose_access CHECK (access IN ('local', 'public')),
    CONSTRAINT chk_service_expose_listen_port CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
