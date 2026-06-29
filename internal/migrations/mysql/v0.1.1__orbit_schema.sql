-- v0.1.1: Complete database schema aligned with backend v0.7.0

CREATE TABLE IF NOT EXISTS user (
    id VARCHAR(26) PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    last_login_at DATETIME(3),
    oauth_provider VARCHAR(64) NOT NULL DEFAULT '',
    oauth_provider_id VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) DEFAULT NULL,
    auth_source VARCHAR(64) NOT NULL DEFAULT 'password',
    status VARCHAR(32) NOT NULL DEFAULT 'enabled',
    CHECK (status IN ('enabled', 'disabled'))
);

CREATE INDEX idx_user_email ON user(email);
CREATE INDEX idx_user_status ON user(status);
CREATE INDEX idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX uq_user_email ON user(email);

CREATE TABLE IF NOT EXISTS login_history (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    username VARCHAR(255) NOT NULL,
    ip_address VARCHAR(64),
    user_agent TEXT,
    login_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    success TINYINT(1) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_login_history_user_id ON login_history(user_id);
CREATE INDEX idx_login_history_login_at ON login_history(login_at DESC);

CREATE TABLE IF NOT EXISTS login_attempt (
    id VARCHAR(26) PRIMARY KEY,
    username VARCHAR(255),
    ip_address VARCHAR(64) NOT NULL,
    user_agent TEXT,
    success TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE INDEX idx_login_attempt_ip_created ON login_attempt(ip_address, created_at);
CREATE INDEX idx_login_attempt_username_created ON login_attempt(username, created_at);
CREATE INDEX idx_login_attempt_created ON login_attempt(created_at);

CREATE TABLE IF NOT EXISTS project (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(255) NOT NULL UNIQUE,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    is_active TINYINT(1) NOT NULL DEFAULT 1
);

CREATE INDEX idx_project_code ON project(code);

CREATE TABLE IF NOT EXISTS application (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    code VARCHAR(255) NOT NULL,
    image_pull_policy VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    route_managed TINYINT(1) NOT NULL DEFAULT 0,
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_application_project ON application(project_id);

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
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX idx_deployment_app ON deployment(application_id);
CREATE INDEX idx_deployment_app_name ON deployment(application_name);
CREATE INDEX idx_deployment_status ON deployment(status);
CREATE INDEX idx_deployment_started ON deployment(started_at);
CREATE INDEX idx_deployment_project ON deployment(project_id);

CREATE TABLE IF NOT EXISTS application_config_file (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    path VARCHAR(512) NOT NULL,
    content LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    UNIQUE(application_id, path)
);

CREATE INDEX idx_app_config_file_app ON application_config_file(application_id);

CREATE TABLE IF NOT EXISTS application_route (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    port INT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

CREATE INDEX idx_app_route_app ON application_route(application_id);

CREATE TABLE IF NOT EXISTS application_service (
    id VARCHAR(26) PRIMARY KEY,
    application_id VARCHAR(26) NOT NULL,
    service_name VARCHAR(255) NOT NULL,
    image VARCHAR(512),
    environment LONGTEXT,
    volumes LONGTEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    UNIQUE(application_id, service_name)
);

CREATE INDEX idx_application_service_app ON application_service(application_id);

CREATE TABLE IF NOT EXISTS route (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL,
    path_prefix VARCHAR(512) NOT NULL,
    target_url VARCHAR(1024) NOT NULL,
    enabled TINYINT(1) NOT NULL,
    https_enabled TINYINT(1) NOT NULL DEFAULT 0,
    cert_pem LONGTEXT,
    cert_key LONGTEXT,
    cert_type VARCHAR(32) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_route_domain ON route(domain);
CREATE INDEX idx_route_enabled ON route(enabled);
CREATE INDEX idx_route_project ON route(project_id);

CREATE TABLE IF NOT EXISTS credential (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    encrypted_data LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_credential_name ON credential(name);
CREATE INDEX idx_credential_project ON credential(project_id);

CREATE TABLE IF NOT EXISTS pipeline_template (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    variable_declarations VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_pipeline_template_name ON pipeline_template(name);
CREATE INDEX idx_pipeline_template_project ON pipeline_template(project_id);

CREATE TABLE IF NOT EXISTS build_stage (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(512) NOT NULL,
    script LONGTEXT NOT NULL,
    artifacts LONGTEXT,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id)
);

CREATE INDEX idx_build_stage_name ON build_stage(name);
CREATE INDEX idx_build_stage_project ON build_stage(project_id);

CREATE TABLE IF NOT EXISTS pipeline_template_stage (
    id VARCHAR(26) PRIMARY KEY,
    template_id VARCHAR(26) NOT NULL,
    stage_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    stage_version INT NOT NULL DEFAULT 1,
    depends_on VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    sort_order INT NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES build_stage(id),
    UNIQUE (template_id, stage_name)
);

CREATE INDEX idx_pipeline_template_stage_template ON pipeline_template_stage(template_id);
CREATE INDEX idx_pipeline_template_stage_stage ON pipeline_template_stage(stage_id);

CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id VARCHAR(26) PRIMARY KEY,
    template_id VARCHAR(26) NOT NULL,
    version INT NOT NULL,
    stages_snapshot VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    variables_snapshot VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id),
    UNIQUE (template_id, version)
);

CREATE INDEX idx_pipeline_snapshot_template ON pipeline_snapshot(template_id);
CREATE INDEX idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);

CREATE TABLE IF NOT EXISTS repository (
    id VARCHAR(26) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    code VARCHAR(255) NOT NULL UNIQUE,
    repository_url VARCHAR(1024) NOT NULL,
    git_credential_id VARCHAR(26),
    variable_overrides VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    default_branch VARCHAR(255) NOT NULL DEFAULT 'master',
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);

CREATE INDEX idx_repository_name ON repository(name);
CREATE INDEX idx_repository_code ON repository(code);
CREATE INDEX idx_repository_credential ON repository(git_credential_id);
CREATE INDEX idx_repository_project ON repository(project_id);

CREATE TABLE IF NOT EXISTS repository_webhook (
    id VARCHAR(26) PRIMARY KEY,
    repository_id VARCHAR(26) NOT NULL,
    name VARCHAR(255) NOT NULL,
    template_id VARCHAR(26) NOT NULL,
    branch_filter VARCHAR(255),
    encrypted_secret LONGTEXT NOT NULL,
    enabled TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    FOREIGN KEY (repository_id) REFERENCES repository(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE RESTRICT
);

CREATE INDEX idx_repository_webhook_repository ON repository_webhook(repository_id);

CREATE TABLE IF NOT EXISTS pipeline_run (
    id VARCHAR(26) PRIMARY KEY,
    repository_id VARCHAR(26) NOT NULL,
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    snapshot_id VARCHAR(26) NOT NULL,
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    template_version INT NOT NULL,
    `trigger` VARCHAR(64) NOT NULL,
    trigger_ref VARCHAR(255) NOT NULL,
    variables_snapshot VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    status VARCHAR(32) NOT NULL,
    retry_of VARCHAR(26),
    started_at DATETIME(3),
    finished_at DATETIME(3),
    error_message TEXT,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
);

CREATE INDEX idx_pipeline_run_project_created ON pipeline_run(repository_id, created_at DESC);
CREATE INDEX idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX idx_pipeline_run_retry_of ON pipeline_run(retry_of);
CREATE INDEX idx_pipeline_run_project ON pipeline_run(project_id);

CREATE TABLE IF NOT EXISTS stage_run (
    id VARCHAR(26) PRIMARY KEY,
    pipeline_run_id VARCHAR(26) NOT NULL,
    stage_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at DATETIME(3),
    finished_at DATETIME(3),
    exit_code INT,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id)
);

CREATE INDEX idx_stage_run_run ON stage_run(pipeline_run_id);
CREATE INDEX idx_stage_run_status ON stage_run(status);

CREATE TABLE IF NOT EXISTS artifact (
    id VARCHAR(26) PRIMARY KEY,
    pipeline_run_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    type VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    path VARCHAR(1024),
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    repository_id VARCHAR(26) NOT NULL DEFAULT '',
    repository_name VARCHAR(255) NOT NULL DEFAULT '',
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    project_id VARCHAR(26),
    FOREIGN KEY (project_id) REFERENCES project(id),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
);

CREATE INDEX idx_artifact_run ON artifact(pipeline_run_id);
CREATE INDEX ix_artifact_repository_id ON artifact (repository_id);
CREATE INDEX idx_artifact_project ON artifact(project_id);
