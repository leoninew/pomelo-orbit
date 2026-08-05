-- Domain: application
-- Tables: application, version, version_component
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS application (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'standard',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);

CREATE TABLE IF NOT EXISTS version (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    label TEXT NOT NULL,
    status TEXT NOT NULL,
    created_from_version_id TEXT,
    note TEXT,
    component_summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (created_from_version_id) REFERENCES version(id),
    UNIQUE(application_id, label)
);

CREATE INDEX IF NOT EXISTS idx_version_application ON version(application_id);
CREATE INDEX IF NOT EXISTS idx_version_status ON version(status);

CREATE TABLE IF NOT EXISTS pipeline_stage_build_version_binding (
    pipeline_stage_id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    component_name TEXT NOT NULL,
    fork_strategy TEXT NOT NULL CHECK (fork_strategy IN ('latest', 'fixed')),
    fixed_version_id TEXT,
    FOREIGN KEY (pipeline_stage_id) REFERENCES pipeline_stage(id) ON DELETE CASCADE,
    FOREIGN KEY (application_id) REFERENCES application(id),
    FOREIGN KEY (fixed_version_id) REFERENCES version(id),
    CHECK (
        (fork_strategy = 'latest' AND fixed_version_id IS NULL) OR
        (fork_strategy = 'fixed' AND fixed_version_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_build_version_binding_application
    ON pipeline_stage_build_version_binding(application_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_build_version_binding_fixed_version
    ON pipeline_stage_build_version_binding(fixed_version_id);

CREATE TABLE IF NOT EXISTS pipeline_run_build_version_binding (
    pipeline_run_id TEXT NOT NULL,
    pipeline_stage_id TEXT NOT NULL,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    component_name TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    source_version_label TEXT NOT NULL,
    generated_version_id TEXT,
    generated_version_label TEXT,
    artifact_id TEXT,
    PRIMARY KEY (pipeline_run_id, pipeline_stage_id),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE,
    FOREIGN KEY (application_id) REFERENCES application(id),
    FOREIGN KEY (source_version_id) REFERENCES version(id),
    FOREIGN KEY (generated_version_id) REFERENCES version(id),
    FOREIGN KEY (artifact_id) REFERENCES artifact(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_build_version_binding_source_version
    ON pipeline_run_build_version_binding(source_version_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_build_version_binding_generated_version
    ON pipeline_run_build_version_binding(generated_version_id);

CREATE TABLE IF NOT EXISTS version_component (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    artifact_id TEXT,
    command_json TEXT NOT NULL DEFAULT '[]',
    pull_policy TEXT NOT NULL CHECK (pull_policy IN ('always', 'missing', 'never')),
    restart_policy TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    FOREIGN KEY (artifact_id) REFERENCES artifact(id) ON DELETE SET NULL,
    UNIQUE(version_id, name)
);

CREATE INDEX IF NOT EXISTS idx_version_component_version ON version_component(version_id);
CREATE INDEX IF NOT EXISTS idx_version_component_artifact_id ON version_component(artifact_id);

CREATE TABLE IF NOT EXISTS version_component_env (
    component_id TEXT NOT NULL,
    env_key TEXT NOT NULL,
    value TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, env_key),
    UNIQUE (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_endpoint (
    component_id TEXT NOT NULL,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('http', 'tcp')),
    container_port INTEGER NOT NULL CHECK (container_port BETWEEN 1 AND 65535),
    mode TEXT NOT NULL DEFAULT 'internal' CHECK (mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp')),
    bind_address TEXT,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    entrypoint TEXT,
    path_prefix TEXT,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, name),
    UNIQUE (component_id, position),
    UNIQUE (component_id, protocol, container_port),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_mount (
    component_id TEXT NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('directory', 'file', 'named_volume', 'controlled_file')),
    source TEXT NOT NULL,
    target TEXT NOT NULL,
    read_only INTEGER NOT NULL DEFAULT 0 CHECK (read_only IN (0, 1)),
    source_is_host_path INTEGER NOT NULL DEFAULT 0 CHECK (source_is_host_path IN (0, 1)),
    content TEXT,
    content_masked INTEGER NOT NULL DEFAULT 0 CHECK (content_masked IN (0, 1)),
    mode TEXT NOT NULL DEFAULT '',
    ignore_if_exists INTEGER NOT NULL DEFAULT 0 CHECK (ignore_if_exists IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_dependency (
    component_id TEXT NOT NULL,
    depends_on_name TEXT NOT NULL,
    condition TEXT NOT NULL CHECK (condition IN ('service_started', 'service_healthy', 'service_completed_successfully')),
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, depends_on_name),
    UNIQUE (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_healthcheck (
    component_id TEXT PRIMARY KEY,
    test_mode TEXT CHECK (test_mode IN ('CMD', 'CMD-SHELL')),
    test TEXT NOT NULL DEFAULT '',
    interval TEXT,
    timeout TEXT,
    retries INTEGER,
    start_period TEXT,
    start_interval TEXT,
    disabled INTEGER NOT NULL DEFAULT 0 CHECK (disabled IN (0, 1)),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_resource (
    component_id TEXT PRIMARY KEY,
    limit_cpus TEXT,
    limit_memory TEXT,
    reservation_cpus TEXT,
    reservation_memory TEXT,
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_tmpfs (
    component_id TEXT NOT NULL,
    target TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    mode TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, position),
    UNIQUE (component_id, target),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_ulimit (
    component_id TEXT NOT NULL,
    name TEXT NOT NULL,
    soft INTEGER NOT NULL,
    hard INTEGER NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, name),
    UNIQUE (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_device (
    component_id TEXT NOT NULL,
    driver TEXT NOT NULL,
    device_count TEXT NOT NULL,
    capabilities_json TEXT NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);
