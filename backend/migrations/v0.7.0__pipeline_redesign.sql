-- v0.7.0: 流水线重新设计
-- 不向后兼容，删除旧 CI 表后重建

-- 删除旧表（按依赖顺序）
DROP TABLE IF EXISTS job_logs;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS pipeline_runs;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS pipeline_templates;

-- 流水线模板表（stages 替代原 content）
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    stages TEXT NOT NULL DEFAULT '[]',              -- JSON array of StageDefinition
    variable_declarations TEXT NOT NULL DEFAULT '[]', -- JSON array of VariableDeclaration
    is_builtin INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_templates_name ON pipeline_templates(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_templates_builtin ON pipeline_templates(is_builtin);

-- 流水线快照表（模板的不可变版本副本）
CREATE TABLE IF NOT EXISTS pipeline_snapshots (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',                 -- JSON，不可修改
    variable_declarations_snapshot TEXT NOT NULL DEFAULT '[]',  -- JSON，不可修改
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (template_id) REFERENCES pipeline_templates(id),
    UNIQUE (template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_snapshots_template ON pipeline_snapshots(template_id);

-- 项目表（引用快照而非模板）
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    repository_url TEXT NOT NULL,
    pipeline_snapshot_id TEXT NOT NULL,
    git_credential_id TEXT,
    variable_overrides TEXT NOT NULL DEFAULT '{}',  -- JSON object
    default_branch TEXT NOT NULL DEFAULT 'master',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_snapshot_id) REFERENCES pipeline_snapshots(id),
    FOREIGN KEY (git_credential_id) REFERENCES credentials(id)
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_snapshot ON projects(pipeline_snapshot_id);
CREATE INDEX IF NOT EXISTS idx_projects_credential ON projects(git_credential_id);

-- Pipeline 运行表（引用快照 ID，不再存储完整 YAML）
CREATE TABLE IF NOT EXISTS pipeline_runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    pipeline_snapshot_id TEXT NOT NULL,
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '{}',  -- JSON，本次运行合并后的变量副本
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at TEXT,
    finished_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id),
    FOREIGN KEY (pipeline_snapshot_id) REFERENCES pipeline_snapshots(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_runs_project_created ON pipeline_runs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_status ON pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_snapshot ON pipeline_runs(pipeline_snapshot_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_runs_retry_of ON pipeline_runs(retry_of);

-- Job 表
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    name TEXT NOT NULL,
    parent_job_id TEXT,
    status TEXT NOT NULL,
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

-- 制品表
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    job_name TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(pipeline_run_id);
