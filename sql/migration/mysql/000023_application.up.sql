-- Domain: application
-- Tables: application, version, version_component, version_component_secret_env_ref, version_expose
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS application (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    code VARCHAR(255) NOT NULL,
    kind VARCHAR(32) NOT NULL DEFAULT 'standard',
    image_pull_policy VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_application_project ON application(project_id);

CREATE TABLE IF NOT EXISTS version (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    label VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    env_json LONGTEXT,
    created_from_version_id VARCHAR(26),
    note TEXT,
    component_summary TEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (created_from_version_id) REFERENCES version(id),
    UNIQUE(application_id, label)
);

CREATE INDEX idx_version_application ON version(application_id);
CREATE INDEX idx_version_status ON version(status);

CREATE TABLE IF NOT EXISTS version_component (
    id VARCHAR(26) PRIMARY KEY,
    version_id VARCHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(512) NOT NULL,
    command_json LONGTEXT,
    args_json LONGTEXT,
    env_json LONGTEXT,
    ports_json LONGTEXT,
    mounts_json LONGTEXT,
    networks_json LONGTEXT,
    depends_on_json LONGTEXT,
    healthcheck_json LONGTEXT,
    resources_json LONGTEXT,
    pull_policy VARCHAR(32),
    restart_policy VARCHAR(32),
    tmpfs_json LONGTEXT,
    ulimits_json LONGTEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, name)
);

CREATE INDEX idx_version_component_version ON version_component(version_id);

CREATE TABLE IF NOT EXISTS version_component_secret_env_ref (
    component_id VARCHAR(26) NOT NULL,
    env_key VARCHAR(255) NOT NULL,
    credential_id VARCHAR(26) NOT NULL,
    data_key VARCHAR(255) NOT NULL,
    PRIMARY KEY (component_id, env_key),
    KEY idx_version_component_secret_env_ref_credential (credential_id),
    CONSTRAINT fk_version_component_secret_env_ref_component
        FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT fk_version_component_secret_env_ref_credential
        FOREIGN KEY (credential_id) REFERENCES credential(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_expose (
    id VARCHAR(26) PRIMARY KEY,
    version_id VARCHAR(26) NOT NULL,
    component_name VARCHAR(255) NOT NULL,
    protocol VARCHAR(16) NOT NULL,
    container_port INT NOT NULL,
    path_prefix VARCHAR(512) NULL,
    access VARCHAR(16) NOT NULL DEFAULT 'public',
    listen_port INT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_version_expose_key (version_id, component_name, protocol, container_port),
    KEY idx_version_expose_version (version_id),
    CONSTRAINT fk_version_expose_version FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
