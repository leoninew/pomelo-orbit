-- SQLC schema snapshot for the complete SQLite database.
-- It mirrors numbered DDL migrations through 000030. Business data migrations are excluded.

CREATE TABLE IF NOT EXISTS background_task (
    id TEXT PRIMARY KEY,
    task_type TEXT NOT NULL,
    payload_json TEXT NOT NULL,
    status TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    max_attempts INTEGER NOT NULL,
    locked_by TEXT,
    locked_at DATETIME,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_background_task_status_created ON background_task(status, created_at);
CREATE INDEX IF NOT EXISTS idx_background_task_locked_at ON background_task(locked_at);
CREATE INDEX IF NOT EXISTS idx_background_task_type ON background_task(task_type);

CREATE TABLE IF NOT EXISTS role (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS permission (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS role_permission (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permission(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_role_code ON role(code);
CREATE INDEX IF NOT EXISTS idx_role_name ON role(name);
CREATE INDEX IF NOT EXISTS idx_permission_code ON permission(code);
CREATE INDEX IF NOT EXISTS idx_role_permission_role_id ON role_permission(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permission_permission_id ON role_permission(permission_id);

CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_login_at DATETIME,
    oauth_provider TEXT NOT NULL DEFAULT '',
    oauth_provider_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL,
    auth_source TEXT NOT NULL DEFAULT 'password',
    status TEXT NOT NULL DEFAULT 'enabled'
);

CREATE INDEX IF NOT EXISTS idx_user_email ON user(email);
CREATE INDEX IF NOT EXISTS idx_user_status ON user(status);
CREATE INDEX IF NOT EXISTS idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_email ON user(email);

CREATE TABLE IF NOT EXISTS user_role (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_role_user_id ON user_role(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role_id ON user_role(role_id);

CREATE TABLE IF NOT EXISTS login_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    login_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);
CREATE INDEX IF NOT EXISTS idx_login_history_login_at ON login_history(login_at DESC);

CREATE TABLE IF NOT EXISTS project (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    is_active BOOLEAN NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_project_code ON project(code);

CREATE TABLE IF NOT EXISTS project_member (
    project_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (project_id, user_id),
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_project_member_user_id ON project_member(user_id);

CREATE TABLE IF NOT EXISTS credential (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_credential_name ON credential(name);
CREATE INDEX IF NOT EXISTS idx_credential_project ON credential(project_id);

CREATE TABLE IF NOT EXISTS pipeline (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    kind TEXT NOT NULL,
    source_pipeline_id TEXT,
    source_template_name TEXT,
    source_template_version INTEGER,
    application_id TEXT,
    application_name TEXT,
    repository_id TEXT,
    repository_name TEXT,
    version_fork_strategy TEXT,
    fixed_version_id TEXT,
    fixed_version_label TEXT,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (project_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_project ON pipeline(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_kind ON pipeline(kind);
CREATE INDEX IF NOT EXISTS idx_pipeline_source_pipeline ON pipeline(source_pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_application ON pipeline(application_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_repository ON pipeline(repository_id);

CREATE TABLE IF NOT EXISTS pipeline_stage (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    pipeline_id TEXT,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    version INTEGER,
    source_template_stage_id TEXT,
    source_template_stage_name TEXT,
    source_template_stage_version INTEGER,
    source_template_stage_description TEXT,
    artifacts TEXT,
    depends_on TEXT,
    sort_order INTEGER,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline(id) ON DELETE CASCADE,
    UNIQUE (pipeline_id, name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_project_kind_name ON pipeline_stage(project_id, kind, name);

CREATE TABLE IF NOT EXISTS pipeline_stage_reference (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    source_template_stage_id TEXT NOT NULL,
    source_template_stage_name TEXT NOT NULL,
    source_template_stage_version INTEGER NOT NULL,
    source_template_stage_description TEXT NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    artifacts TEXT NOT NULL DEFAULT '[]',
    depends_on TEXT NOT NULL DEFAULT '[]',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_id) REFERENCES pipeline(id) ON DELETE CASCADE,
    UNIQUE (pipeline_id, name)
);


CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    source_pipeline_id TEXT NOT NULL,
    source_template_name TEXT NOT NULL,
    source_template_version INTEGER NOT NULL,
    application_id TEXT,
    application_name TEXT,
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL,
    version_fork_strategy TEXT,
    fixed_version_id TEXT,
    fixed_version_label TEXT,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (pipeline_id, pipeline_version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_pipeline ON pipeline_snapshot(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);

CREATE TABLE IF NOT EXISTS repository (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    repository_type TEXT NOT NULL DEFAULT 'remote_git',
    repository_url TEXT NOT NULL,
    git_credential_id TEXT,
    variable_overrides TEXT NOT NULL DEFAULT '[]',
    default_branch TEXT NOT NULL DEFAULT 'master',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);

CREATE INDEX IF NOT EXISTS idx_repository_name ON repository(name);
CREATE INDEX IF NOT EXISTS idx_repository_code ON repository(code);
CREATE INDEX IF NOT EXISTS idx_repository_credential ON repository(git_credential_id);
CREATE INDEX IF NOT EXISTS idx_repository_project ON repository(project_id);

CREATE TABLE IF NOT EXISTS pipeline_run (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL DEFAULT '',
    snapshot_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    repository_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_pipeline_created ON pipeline_run(pipeline_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_project ON pipeline_run(project_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_repository_status ON pipeline_run(repository_id, status);

CREATE TABLE IF NOT EXISTS pipeline_stage_run (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    exit_code INTEGER,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_run ON pipeline_stage_run(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_run_status ON pipeline_stage_run(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_pipeline_stage_run_run_stage
    ON pipeline_stage_run(pipeline_run_id, stage_id);

CREATE TABLE IF NOT EXISTS pipeline_run_version_binding (
    pipeline_run_id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    source_version_label TEXT NOT NULL,
    generated_version_id TEXT,
    generated_version_label TEXT
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_version_binding_source_version
    ON pipeline_run_version_binding(source_version_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_version_binding_generated_version
    ON pipeline_run_version_binding(generated_version_id);

CREATE TABLE IF NOT EXISTS artifact (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES project(id),
    pipeline_run_id TEXT NOT NULL,
    repository_id TEXT NOT NULL DEFAULT '',
    repository_name TEXT NOT NULL DEFAULT '',
    pipeline_id TEXT NOT NULL DEFAULT '',
    pipeline_name TEXT NOT NULL DEFAULT '',
    pipeline_stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    collector TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT,
    value TEXT,
    value_format TEXT,
    image_ref TEXT,
    local_image_sha256 TEXT,
    source_artifact_id TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (source_artifact_id) REFERENCES artifact(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_artifact_run_stage ON artifact(pipeline_run_id, pipeline_stage_id);
CREATE INDEX IF NOT EXISTS idx_artifact_repository ON artifact(repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
CREATE INDEX IF NOT EXISTS idx_artifact_pipeline ON artifact(pipeline_id);

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
    entrypoint_json TEXT NOT NULL DEFAULT '[]',
    artifact_name TEXT,
    artifact_image_ref TEXT,
    artifact_local_image_sha256 TEXT,
    artifact_source_commit_sha TEXT,
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
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
    protocol TEXT NOT NULL CHECK (protocol IN ('http', 'tcp')),
    container_port INTEGER NOT NULL CHECK (container_port BETWEEN 1 AND 65535),
    mode TEXT NOT NULL DEFAULT 'internal' CHECK (mode IN ('internal', 'local', 'host', 'gateway')),
    bind_address TEXT,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    entrypoint TEXT,
    path_prefix TEXT,
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, protocol, container_port),
    UNIQUE (component_id, position),
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
    "condition" TEXT NOT NULL CHECK ("condition" IN ('service_started', 'service_healthy', 'service_completed_successfully')),
    position INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (component_id, depends_on_name),
    UNIQUE (component_id, position),
    FOREIGN KEY (component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS version_component_healthcheck (
    component_id TEXT PRIMARY KEY,
    test_mode TEXT CHECK (test_mode IN ('CMD', 'CMD-SHELL')),
    test TEXT NOT NULL DEFAULT '',
    "interval" TEXT,
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

CREATE TABLE IF NOT EXISTS gateway_config (
    application_id TEXT PRIMARY KEY,
    traefik_component_name TEXT NOT NULL DEFAULT 'traefik',
    rest_api_url TEXT NOT NULL,
    rest_ready_timeout_seconds INTEGER NOT NULL DEFAULT 20,
    base_domain TEXT NOT NULL,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tls_mode TEXT NOT NULL DEFAULT 'none',
    acme_profile TEXT NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns')),
    acme_email TEXT NOT NULL DEFAULT '',
    dns_api_token TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS gateway_acme_profile_version (
    application_id TEXT NOT NULL,
    profile TEXT NOT NULL CHECK (profile IN ('base', 'http', 'dns', 'http-dns')),
    version_id TEXT NOT NULL,
    PRIMARY KEY (application_id, profile),
    UNIQUE (application_id, version_id),
    FOREIGN KEY (application_id) REFERENCES gateway_config(application_id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    code TEXT NOT NULL UNIQUE,
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    UNIQUE(application_id, instance_key)
);

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_application ON service(application_id);

CREATE TABLE IF NOT EXISTS service_env (
    service_id TEXT NOT NULL,
    env_key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (service_id, env_key),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS service_component (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    source_version_component_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    entrypoint_json TEXT,
    command_json TEXT,
    pull_policy TEXT CHECK (pull_policy IS NULL OR pull_policy IN ('always', 'missing', 'never')),
    restart_policy TEXT CHECK (restart_policy IS NULL OR restart_policy IN ('no', 'unless-stopped')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (service_id, source_version_component_id),
    UNIQUE (service_id, component_name),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    FOREIGN KEY (source_version_component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_component_service ON service_component(service_id);

CREATE TABLE IF NOT EXISTS service_component_env (
    service_component_id TEXT NOT NULL,
    env_key TEXT NOT NULL,
    value TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    PRIMARY KEY (service_component_id, env_key),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'override' AND value IS NOT NULL)
        OR (state = 'deleted' AND value IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS service_component_mount (
    id TEXT PRIMARY KEY,
    service_component_id TEXT NOT NULL,
    target TEXT NOT NULL,
    source TEXT,
    source_is_host_path INTEGER,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_service_component_mount_component ON service_component_mount(service_component_id);

CREATE TABLE IF NOT EXISTS service_component_resource (
    service_component_id TEXT PRIMARY KEY,
    limit_cpus TEXT,
    limit_memory TEXT,
    reservation_cpus TEXT,
    reservation_memory TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'deleted' AND limit_cpus IS NULL AND limit_memory IS NULL AND reservation_cpus IS NULL AND reservation_memory IS NULL)
        OR (state = 'override' AND (limit_cpus IS NOT NULL OR limit_memory IS NOT NULL OR reservation_cpus IS NOT NULL OR reservation_memory IS NOT NULL))
    )
);

CREATE TABLE IF NOT EXISTS service_component_endpoint (
    id TEXT PRIMARY KEY,
    service_component_id TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('http', 'tcp')),
    container_port INTEGER NOT NULL CHECK (container_port BETWEEN 1 AND 65535),
    mode TEXT CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway')),
    bind_address TEXT,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    entrypoint TEXT,
    path_prefix TEXT,
    state TEXT NOT NULL CHECK (state IN ('override', 'deleted')),
    UNIQUE (service_component_id, protocol, container_port),
    FOREIGN KEY (service_component_id) REFERENCES service_component(id) ON DELETE CASCADE,
    CHECK (
        (state = 'deleted' AND mode IS NULL AND bind_address IS NULL AND listen_port IS NULL AND entrypoint IS NULL AND path_prefix IS NULL)
        OR (state = 'override' AND (mode IS NOT NULL OR bind_address IS NOT NULL OR listen_port IS NOT NULL OR entrypoint IS NOT NULL OR path_prefix IS NOT NULL))
    )
);

CREATE INDEX IF NOT EXISTS idx_service_component_endpoint_listen ON service_component_endpoint(listen_port);

CREATE TABLE IF NOT EXISTS deployment (
    id TEXT PRIMARY KEY,
    application_id TEXT,
    application_name TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    env_file TEXT,
    status TEXT NOT NULL,
    started_at DATETIME NOT NULL DEFAULT (datetime('now')),
    finished_at DATETIME,
    duration_ms INTEGER,
    log_text TEXT,
    error_message TEXT,
    is_rollback INTEGER NOT NULL,
    rollback_from_deployment_id TEXT,
    project_id TEXT REFERENCES project(id),
    version_id TEXT,
    service_id TEXT,
    options_json TEXT,
    effective_plan_hash TEXT,
    command_text TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX IF NOT EXISTS idx_deployment_app ON deployment(application_id);
CREATE INDEX IF NOT EXISTS idx_deployment_app_name ON deployment(application_name);
CREATE INDEX IF NOT EXISTS idx_deployment_status ON deployment(status);
CREATE INDEX IF NOT EXISTS idx_deployment_started ON deployment(started_at);
CREATE INDEX IF NOT EXISTS idx_deployment_project ON deployment(project_id);
CREATE INDEX IF NOT EXISTS idx_deployment_service_success_hash ON deployment(service_id, status, effective_plan_hash, finished_at);

CREATE TABLE IF NOT EXISTS route (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL DEFAULT 'http' CHECK (protocol IN ('http', 'tcp')),
    domain TEXT NOT NULL,
    path_prefix TEXT NOT NULL,
    target_url TEXT NOT NULL,
    listen_port INTEGER CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    service_id TEXT,
    component_name TEXT,
    endpoint_protocol TEXT,
    endpoint_container_port INTEGER CHECK (endpoint_container_port IS NULL OR endpoint_container_port BETWEEN 1 AND 65535),
    enabled INTEGER NOT NULL,
    https_enabled INTEGER NOT NULL DEFAULT 0,
    cert_pem TEXT,
    cert_key TEXT,
    cert_type TEXT NOT NULL,
    acme_challenge TEXT NOT NULL DEFAULT 'http' CHECK (acme_challenge IN ('http', 'dns')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_route_domain ON route(domain);
CREATE INDEX IF NOT EXISTS idx_route_enabled ON route(enabled);
CREATE INDEX IF NOT EXISTS idx_route_project ON route(project_id);
CREATE INDEX IF NOT EXISTS idx_route_tcp_listen ON route(protocol, listen_port);

CREATE TABLE IF NOT EXISTS deployment_dialogue_conversation (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    created_by_user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS deployment_dialogue_message (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES deployment_dialogue_conversation(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_deployment_dialogue_conversation_project_updated
    ON deployment_dialogue_conversation(project_id, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_deployment_dialogue_message_conversation_created
    ON deployment_dialogue_message(conversation_id, created_at ASC, id ASC);
