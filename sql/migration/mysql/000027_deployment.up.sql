-- Domain: deployment
-- Tables: deployment
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS deployment (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26),
    application_name VARCHAR(255) NOT NULL,
    operation_type VARCHAR(32) NOT NULL,
    trigger_type VARCHAR(32) NOT NULL,
    env_file LONGTEXT,
    status VARCHAR(32) NOT NULL,
    started_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    finished_at DATETIME(3),
    duration_ms BIGINT,
    log_text LONGTEXT,
    error_message TEXT,
    is_rollback TINYINT(1) NOT NULL,
    rollback_from_deployment_id VARCHAR(26),
    project_id VARCHAR(26),
    version_id VARCHAR(26) NULL,
    service_id VARCHAR(26) NULL,
    environment_id VARCHAR(26) NULL,
    options_json LONGTEXT NULL,
    command_text VARCHAR(2048) NOT NULL DEFAULT '',
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id),
    CONSTRAINT fk_deployment_version FOREIGN KEY (version_id) REFERENCES version(id),
    CONSTRAINT fk_deployment_service FOREIGN KEY (service_id) REFERENCES service(id),
    CONSTRAINT fk_deployment_environment FOREIGN KEY (environment_id) REFERENCES environment(id)
);

CREATE INDEX idx_deployment_app ON deployment(application_id);
CREATE INDEX idx_deployment_app_name ON deployment(application_name);
CREATE INDEX idx_deployment_status ON deployment(status);
CREATE INDEX idx_deployment_started ON deployment(started_at);
CREATE INDEX idx_deployment_project ON deployment(project_id);
