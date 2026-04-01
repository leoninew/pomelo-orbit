-- v0.6.0: CI 系统表结构

-- 凭据表
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL CHECK(type IN ('git_ssh', 'git_token')),
    encrypted_data TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_credentials_name ON credentials(name);

-- Pipeline 模板表
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL,
    variable_declarations TEXT NOT NULL DEFAULT '[]',  -- JSON array
    is_builtin INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_templates_name ON pipeline_templates(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_templates_builtin ON pipeline_templates(is_builtin);

-- 项目表
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repository_url TEXT NOT NULL,
    pipeline_template_id TEXT NOT NULL,
    git_credential_id TEXT NOT NULL,
    variable_overrides TEXT NOT NULL DEFAULT '{}',  -- JSON object
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_template_id) REFERENCES pipeline_templates(id),
    FOREIGN KEY (git_credential_id) REFERENCES credentials(id)
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_template ON projects(pipeline_template_id);
CREATE INDEX IF NOT EXISTS idx_projects_credential ON projects(git_credential_id);

-- Pipeline 运行表
CREATE TABLE IF NOT EXISTS pipeline_runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    trigger TEXT NOT NULL CHECK(trigger IN ('manual', 'webhook')),
    trigger_ref TEXT NOT NULL,
    resolved_pipeline TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '{}',  -- JSON object
    status TEXT NOT NULL CHECK(status IN ('waiting', 'running', 'success', 'failed')),
    started_at TEXT,
    finished_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project_created ON pipeline_runs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);

-- Job 表
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    name TEXT NOT NULL,
    parent_job_id TEXT,
    status TEXT NOT NULL CHECK(status IN ('waiting', 'running', 'success', 'failed', 'faulted', 'skipped', 'canceled')),
    started_at TEXT,
    finished_at TEXT,
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id),
    FOREIGN KEY (parent_job_id) REFERENCES jobs(id)
);

CREATE INDEX IF NOT EXISTS idx_jobs_run_parent ON jobs(pipeline_run_id, parent_job_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

-- Job 日志表
CREATE TABLE IF NOT EXISTS job_logs (
    id TEXT PRIMARY KEY,
    job_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (job_id) REFERENCES jobs(id)
);

CREATE INDEX IF NOT EXISTS idx_job_logs_job ON job_logs(job_id);
