-- v0.6.0: CI 系统完整表结构

-- 凭据表
CREATE TABLE IF NOT EXISTS credential (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    encrypted_data TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_credential_name ON credential(name);

-- 流水线模板表
CREATE TABLE IF NOT EXISTS pipeline_template (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',  -- JSON array of VariableDeclaration
    version INTEGER NOT NULL DEFAULT 1,               -- 每次修改递增，快照直接引用此值
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_name ON pipeline_template(name);

-- 独立 Stage 表（执行最小单元，不含编排属性）
CREATE TABLE IF NOT EXISTS pipeline_stage (
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

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_name ON pipeline_stage(name);

-- 模板编排表（模板对 Stage 的引用 + 依赖 + 顺序）
CREATE TABLE IF NOT EXISTS pipeline_template_stage (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,              -- 模板内唯一标识，默认为 stage 名，用于 depends_on 引用
    depends_on TEXT NOT NULL DEFAULT '[]',  -- JSON array of stage_name
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES pipeline_stage(id),
    UNIQUE (template_id, stage_name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_template ON pipeline_template_stage(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_stage ON pipeline_template_stage(stage_id);

-- 流水线快照表（模板某一版本的不可变副本，trigger 时按需创建）
CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',                 -- JSON，不可修改
    variables_snapshot TEXT NOT NULL DEFAULT '[]',  -- JSON，不可修改
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id),
    UNIQUE (template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_template ON pipeline_snapshot(template_id);

-- 项目表
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
    FOREIGN KEY (git_credential_id) REFERENCES credential(id)
);

CREATE INDEX IF NOT EXISTS idx_repository_name ON repository(name);
CREATE INDEX IF NOT EXISTS idx_repository_code ON repository(code);
CREATE INDEX IF NOT EXISTS idx_repository_credential ON repository(git_credential_id);

-- 项目 Webhook 配置表
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

-- Pipeline 运行表
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
    FOREIGN KEY (repository_id) REFERENCES repository(id),
    FOREIGN KEY (snapshot_id) REFERENCES pipeline_snapshot(id),
    FOREIGN KEY (retry_of) REFERENCES pipeline_run(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_run_project_created ON pipeline_run(repository_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_status ON pipeline_run(status);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_snapshot ON pipeline_run(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_run_retry_of ON pipeline_run(retry_of);

-- Stage 执行记录表
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

-- 制品表
CREATE TABLE IF NOT EXISTS artifact (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    path TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (pipeline_run_id) REFERENCES pipeline_run(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_artifact_run ON artifact(pipeline_run_id);
