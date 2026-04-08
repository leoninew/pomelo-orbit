-- v0.6.0: CI 系统完整表结构

-- 凭据表
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_credentials_name ON credentials(name);

-- 流水线模板表
CREATE TABLE IF NOT EXISTS pipeline_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',  -- JSON array of VariableDeclaration
    version INTEGER NOT NULL DEFAULT 1,               -- 每次修改递增，快照直接引用此值
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_templates_name ON pipeline_templates(name);

-- 独立 Stage 表（执行最小单元，不含编排属性）
CREATE TABLE IF NOT EXISTS pipeline_stages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    env TEXT NOT NULL DEFAULT '{}',       -- JSON object
    artifacts TEXT,                        -- JSON array | NULL
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stages_name ON pipeline_stages(name);

-- 模板编排表（模板对 Stage 的引用 + 依赖 + 顺序）
CREATE TABLE IF NOT EXISTS pipeline_template_stages (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_key TEXT NOT NULL,              -- 模板内唯一标识，默认为 stage 名，用于 depends_on 引用
    depends_on TEXT NOT NULL DEFAULT '[]',  -- JSON array of stage_key
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES pipeline_stages(id),
    UNIQUE (template_id, stage_key)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_stages_template ON pipeline_template_stages(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_stages_stage ON pipeline_template_stages(stage_id);

-- 流水线快照表（模板某一版本的不可变副本，trigger 时按需创建）
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

-- 项目表
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    repository_url TEXT NOT NULL,
    git_credential_id TEXT,
    variable_overrides TEXT NOT NULL DEFAULT '{}',
    default_branch TEXT NOT NULL DEFAULT 'master',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (git_credential_id) REFERENCES credentials(id)
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_code ON projects(code);
CREATE INDEX IF NOT EXISTS idx_projects_credential ON projects(git_credential_id);

-- 项目 Webhook 配置表
CREATE TABLE IF NOT EXISTS project_webhooks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    template_id TEXT NOT NULL,
    branch_filter TEXT,
    encrypted_secret TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (template_id) REFERENCES pipeline_templates(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_project_webhooks_project ON project_webhooks(project_id);

-- Pipeline 运行表
CREATE TABLE IF NOT EXISTS pipeline_runs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    project_name VARCHAR(255) NOT NULL DEFAULT '',
    pipeline_snapshot_id TEXT NOT NULL,
    template_id VARCHAR(26) NOT NULL DEFAULT '',
    template_name VARCHAR(255) NOT NULL DEFAULT '',
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL DEFAULT '{}',
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

-- Stage 执行记录表
CREATE TABLE IF NOT EXISTS stage_runs (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    exit_code INTEGER,
    error_message TEXT,
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_stage_runs_run ON stage_runs(pipeline_run_id);
CREATE INDEX IF NOT EXISTS idx_stage_runs_status ON stage_runs(status);

-- 制品表
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run ON artifacts(pipeline_run_id);
