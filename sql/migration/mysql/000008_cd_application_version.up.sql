-- CD application versioning: version / component / service; drop legacy config tables.

CREATE TABLE IF NOT EXISTS version (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    label VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    env_json LONGTEXT,
    created_from_version_id VARCHAR(26),
    note TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (created_from_version_id) REFERENCES version(id),
    UNIQUE(application_id, label)
);

CREATE INDEX idx_version_application ON version(application_id);
CREATE INDEX idx_version_status ON version(status);

CREATE TABLE IF NOT EXISTS component (
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
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, name)
);

CREATE INDEX idx_component_version ON component(version_id);

CREATE TABLE IF NOT EXISTS service (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    version_id VARCHAR(26) NOT NULL,
    last_successful_version_id VARCHAR(26),
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE(application_id),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    FOREIGN KEY (last_successful_version_id) REFERENCES version(id)
);

CREATE INDEX idx_service_version ON service(version_id);

ALTER TABLE deployment ADD COLUMN version_id VARCHAR(26) NULL;
ALTER TABLE deployment ADD COLUMN service_id VARCHAR(26) NULL;
ALTER TABLE deployment ADD COLUMN options_json LONGTEXT NULL;

DROP TABLE IF EXISTS application_config_file;
DROP TABLE IF EXISTS application_service;

ALTER TABLE application DROP COLUMN status;
