-- Domain: application
-- Tables: application, version, version_component, version_expose
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
    pull_policy VARCHAR(32),
    restart_policy VARCHAR(32),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, name)
);

CREATE INDEX idx_version_component_version ON version_component(version_id);

CREATE TABLE IF NOT EXISTS version_component_argument (
    component_id VARCHAR(26) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    position INT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (component_id, kind, position),
    CONSTRAINT fk_version_component_argument_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_argument_kind CHECK (kind IN ('command', 'args')),
    CONSTRAINT chk_version_component_argument_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_env (
    component_id VARCHAR(26) NOT NULL,
    env_key VARCHAR(255) NOT NULL,
    value TEXT NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, env_key),
    UNIQUE KEY uq_version_component_env_position (component_id, position),
    CONSTRAINT fk_version_component_env_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_env_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_port (
    component_id VARCHAR(26) NOT NULL,
    host_port INT NOT NULL,
    container_port INT NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, position),
    CONSTRAINT fk_version_component_port_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_port_host CHECK (host_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_version_component_port_container CHECK (container_port BETWEEN 1 AND 65535),
    CONSTRAINT chk_version_component_port_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_mount (
    component_id VARCHAR(26) NOT NULL,
    source_type VARCHAR(32) NOT NULL,
    source VARCHAR(1024) NOT NULL,
    target VARCHAR(1024) NOT NULL,
    read_only TINYINT(1) NOT NULL DEFAULT 0,
    content LONGTEXT,
    content_mode VARCHAR(16),
    position INT NOT NULL,
    PRIMARY KEY (component_id, position),
    CONSTRAINT fk_version_component_mount_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_mount_type CHECK (source_type IN ('directory', 'file', 'named_volume', 'special')),
    CONSTRAINT chk_version_component_mount_read_only CHECK (read_only IN (0, 1)),
    CONSTRAINT chk_version_component_mount_content_mode CHECK (content_mode IN ('seed', 'sync')),
    CONSTRAINT chk_version_component_mount_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_network (
    component_id VARCHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, position),
    UNIQUE KEY uq_version_component_network_name (component_id, name),
    CONSTRAINT fk_version_component_network_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_network_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_dependency (
    component_id VARCHAR(26) NOT NULL,
    depends_on_name VARCHAR(255) NOT NULL,
    condition VARCHAR(64) NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, depends_on_name),
    UNIQUE KEY uq_version_component_dependency_position (component_id, position),
    CONSTRAINT fk_version_component_dependency_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_dependency_condition CHECK (condition IN ('service_started', 'service_healthy', 'service_completed_successfully')),
    CONSTRAINT chk_version_component_dependency_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_healthcheck (
    component_id VARCHAR(26) PRIMARY KEY,
    test_mode VARCHAR(16),
    interval VARCHAR(64),
    timeout VARCHAR(64),
    retries INT,
    start_period VARCHAR(64),
    start_interval VARCHAR(64),
    disabled TINYINT(1) NOT NULL DEFAULT 0,
    CONSTRAINT fk_version_component_healthcheck_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_healthcheck_mode CHECK (test_mode IN ('CMD', 'CMD-SHELL')),
    CONSTRAINT chk_version_component_healthcheck_disabled CHECK (disabled IN (0, 1))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_healthcheck_arg (
    component_id VARCHAR(26) NOT NULL,
    position INT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (component_id, position),
    CONSTRAINT fk_version_component_healthcheck_arg_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_healthcheck_arg_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_resource (
    component_id VARCHAR(26) PRIMARY KEY,
    limit_cpus VARCHAR(64),
    limit_memory VARCHAR(64),
    reservation_cpus VARCHAR(64),
    reservation_memory VARCHAR(64),
    CONSTRAINT fk_version_component_resource_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_tmpfs (
    component_id VARCHAR(26) NOT NULL,
    target VARCHAR(1024) NOT NULL,
    size_bytes BIGINT NOT NULL,
    mode VARCHAR(8) NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, position),
    UNIQUE KEY uq_version_component_tmpfs_target (component_id, target),
    CONSTRAINT fk_version_component_tmpfs_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_tmpfs_position CHECK (position >= 0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS version_component_ulimit (
    component_id VARCHAR(26) NOT NULL,
    name VARCHAR(32) NOT NULL,
    soft BIGINT NOT NULL,
    hard BIGINT NOT NULL,
    position INT NOT NULL,
    PRIMARY KEY (component_id, name),
    UNIQUE KEY uq_version_component_ulimit_position (component_id, position),
    CONSTRAINT fk_version_component_ulimit_component FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE,
    CONSTRAINT chk_version_component_ulimit_position CHECK (position >= 0)
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
