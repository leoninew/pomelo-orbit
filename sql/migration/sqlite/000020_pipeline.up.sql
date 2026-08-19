-- Domain: pipeline
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

CREATE INDEX idx_pipeline_stage_project_kind_name ON pipeline_stage(project_id, kind, name);

CREATE TABLE pipeline_stage_reference (
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
