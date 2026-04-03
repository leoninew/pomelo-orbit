-- v0.5.0: 数据库表结构定义

-- 用户
CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_login_at DATETIME
);

-- 登录历史
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

-- 应用（聚合根）
CREATE TABLE IF NOT EXISTS application (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    image_pull_policy TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 镜像源
CREATE TABLE IF NOT EXISTS image_source (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL UNIQUE,
    image_name TEXT NOT NULL,
    registry_url TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

-- 回调事件
CREATE TABLE IF NOT EXISTS webhook_event (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    event_type TEXT NOT NULL,
    repository_name TEXT,
    repository_url TEXT,
    branch TEXT,
    sender TEXT,
    image_name TEXT,
    payload TEXT,
    signature_valid INTEGER,
    status TEXT NOT NULL,
    matched_application_id TEXT,
    triggered_deployment_id TEXT,
    error_message TEXT,
    received_at DATETIME NOT NULL DEFAULT (datetime('now')),
    processed_at DATETIME,
    FOREIGN KEY (matched_application_id) REFERENCES application(id),
    FOREIGN KEY (triggered_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX IF NOT EXISTS idx_webhook_event_repo ON webhook_event(repository_name);
CREATE INDEX IF NOT EXISTS idx_webhook_event_status ON webhook_event(status);

-- 部署记录
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
    FOREIGN KEY (rollback_from_deployment_id) REFERENCES deployment(id)
);

CREATE INDEX IF NOT EXISTS idx_deployment_app ON deployment(application_id);
CREATE INDEX IF NOT EXISTS idx_deployment_app_name ON deployment(application_name);
CREATE INDEX IF NOT EXISTS idx_deployment_status ON deployment(status);
CREATE INDEX IF NOT EXISTS idx_deployment_started ON deployment(started_at);

-- 应用配置文件
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

-- 路由配置
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
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_route_domain ON route(domain);
CREATE INDEX IF NOT EXISTS idx_route_enabled ON route(enabled);
