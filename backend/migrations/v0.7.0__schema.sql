-- v1.0.0: Complete database schema

-- User table
CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_login_at DATETIME,
    oauth_provider TEXT NOT NULL DEFAULT '',
    oauth_provider_id TEXT NOT NULL DEFAULT '',
    email TEXT DEFAULT NULL,
    auth_source TEXT NOT NULL DEFAULT 'password',
    is_active INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_user_email ON user(email);
CREATE INDEX IF NOT EXISTS idx_user_is_active ON user(is_active);
CREATE INDEX IF NOT EXISTS idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_email ON user(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_oauth_account ON user(oauth_provider, oauth_provider_id)
WHERE oauth_provider != '' AND oauth_provider_id != '';

-- Login history table
CREATE TABLE IF NOT EXISTS login_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    login_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);
CREATE INDEX IF NOT EXISTS idx_login_history_login_at ON login_history(login_at DESC);

-- Login attempt table
CREATE TABLE IF NOT EXISTS login_attempt (
    id TEXT PRIMARY KEY,
    username TEXT,
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_login_attempt_ip_created ON login_attempt(ip_address, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_username_created ON login_attempt(username, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_created ON login_attempt(created_at);

-- Project table
CREATE TABLE IF NOT EXISTS project (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL,
    owner_user_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    is_active BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (owner_user_id) REFERENCES user(id) ON DELETE CASCADE,
    UNIQUE(owner_user_id, code)
);

CREATE INDEX IF NOT EXISTS idx_project_owner_user_id ON project(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_project_code ON project(code);

-- Application table
CREATE TABLE IF NOT EXISTS application (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    image_pull_policy TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    route_managed INTEGER NOT NULL DEFAULT 0,
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);

-- Deployment table
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
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX IF NOT EXISTS idx_deployment_app ON deployment(application_id);
CREATE INDEX IF NOT EXISTS idx_deployment_app_name ON deployment(application_name);
CREATE INDEX IF NOT EXISTS idx_deployment_status ON deployment(status);
CREATE INDEX IF NOT EXISTS idx_deployment_started ON deployment(started_at);
CREATE INDEX IF NOT EXISTS idx_deployment_project ON deployment(project_id);

-- Application config file table
CREATE TABLE IF NOT EXISTS application_config_file (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    UNIQUE(application_id, path)
);

CREATE INDEX IF NOT EXISTS idx_app_config_file_app ON application_config_file(application_id);

-- Application route table
CREATE TABLE IF NOT EXISTS application_route (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    domain TEXT NOT NULL,
    port INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_app_route_app ON application_route(application_id);

-- Application service table
CREATE TABLE IF NOT EXISTS application_service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    image TEXT,
    environment TEXT,
    volumes TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    UNIQUE(application_id, service_name)
);

CREATE INDEX IF NOT EXISTS idx_application_service_app ON application_service(application_id);

-- Route table
CREATE TABLE IF NOT EXISTS route (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    domain TEXT NOT NULL,
    path_prefix TEXT NOT NULL,
    target_url TEXT NOT NULL,
    enabled INTEGER NOT NULL,
    https_enabled INTEGER NOT NULL DEFAULT 0,
    cert_pem TEXT,
    cert_key TEXT,
    cert_type TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_route_domain ON route(domain);
CREATE INDEX IF NOT EXISTS idx_route_enabled ON route(enabled);
CREATE INDEX IF NOT EXISTS idx_route_project ON route(project_id);

-- Credential table
CREATE TABLE IF NOT EXISTS credential (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_credential_name ON credential(name);
CREATE INDEX IF NOT EXISTS idx_credential_project ON credential(project_id);

-- Pipeline template table
CREATE TABLE IF NOT EXISTS pipeline_template (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_name ON pipeline_template(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_project ON pipeline_template(project_id);

-- Build stage table
CREATE TABLE IF NOT EXISTS build_stage (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    artifacts TEXT,
    description TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_build_stage_name ON build_stage(name);
CREATE INDEX IF NOT EXISTS idx_build_stage_project ON build_stage(project_id);

-- Pipeline template stage table
CREATE TABLE IF NOT EXISTS pipeline_template_stage (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    stage_version INTEGER NOT NULL DEFAULT 1,
    depends_on TEXT NOT NULL DEFAULT '[]',
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES build_stage(id),
    UNIQUE (template_id, stage_name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_template ON pipeline_template_stage(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_stage ON pipeline_template_stage(stage_id);

-- Pipeline snapshot table
CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id),
    UNIQUE (template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_template ON pipeline_snapshot(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);

-- Repository table
CREATE TABLE IF NOT EXISTS repository (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    repository_url TEXT NOT NULL,
    git_credential_id TEXT,
    variable_overrides TEXT NOT NULL DEFAULT '[]',
    default_branch TEXT NOT NULL DEFAULT 'master',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);

CREATE INDEX IF NOT EXISTS idx_repository_name ON repository(name);
CREATE INDEX IF NOT EXISTS idx_repository_code ON repository(code);
CREATE INDEX IF NOT EXISTS idx_repository_credential ON repository(git_credential_id);
CREATE INDEX IF NOT EXISTS idx_repository_project ON repository(project_id);

-- Repository webhook table
CREATE TABLE IF NOT EXISTS repository_webhook (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    branch_filter TEXT,
    encrypted_secret TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_repository_webhook_repository ON repository_webhook(repository_id);

-- Pipeline run table
CREATE TABLE IF NOT EXISTS pipeline_run (
    id TEXT PRIMARY KEY,
    repository_id TEXT NOT NULL,
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    snapshot_id TEXT NOT NULL,
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    template_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at TEXT,
    finished_at TEXT,
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_project_created ON pipeline_run(repository_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_project ON pipeline_run(project_id);

-- Stage run table
CREATE TABLE IF NOT EXISTS stage_run (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id)
);

CREATE INDEX IF NOT EXISTS idx_stage_run_run ON stage_run(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_stage_run_status ON stage_run(status);

-- Artifact table
CREATE TABLE IF NOT EXISTS artifact (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    repository_id TEXT NOT NULL DEFAULT '',
    repository_name TEXT NOT NULL DEFAULT '',
    template_id TEXT NOT NULL DEFAULT '',
    template_name TEXT NOT NULL DEFAULT '',
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX IF NOT EXISTS ix_artifact_repository_id ON artifact (repository_id);
CREATE INDEX IF NOT EXISTS idx_artifact_project ON artifact(project_id);
