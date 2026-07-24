-- Domain: service
-- Tables: service
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    environment_id VARCHAR(26) NOT NULL,
    instance_key VARCHAR(100) NOT NULL DEFAULT 'default',
    version_id VARCHAR(26) NOT NULL,
    last_successful_version_id VARCHAR(26),
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_service_app_env_instance (application_id, environment_id, instance_key),
    KEY idx_service_version (version_id),
    KEY idx_service_environment (environment_id),
    KEY idx_service_app_env (application_id, environment_id),
    CONSTRAINT fk_service_application FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    CONSTRAINT fk_service_environment FOREIGN KEY (environment_id) REFERENCES environment(id),
    CONSTRAINT fk_service_version FOREIGN KEY (version_id) REFERENCES version(id),
    CONSTRAINT fk_service_last_successful_version FOREIGN KEY (last_successful_version_id) REFERENCES version(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
