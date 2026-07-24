-- Domain: pipeline
-- Tables: pipeline_template, pipeline_stage, pipeline_template_stage, pipeline_snapshot
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS pipeline_template (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    variable_declarations TEXT NOT NULL DEFAULT '[]',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_name ON pipeline_template(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_project ON pipeline_template(project_id);

CREATE TABLE IF NOT EXISTS pipeline_stage (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL DEFAULT '',
    artifacts TEXT,
    description TEXT NOT NULL DEFAULT '',
    version INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_stage_name ON pipeline_stage(name);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_project ON pipeline_stage(project_id);

CREATE TABLE IF NOT EXISTS pipeline_template_stage (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    stage_version INTEGER NOT NULL DEFAULT 1,
    depends_on TEXT NOT NULL DEFAULT '[]',
    sort_order INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES pipeline_stage(id),
    UNIQUE (template_id, stage_name)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_template ON pipeline_template_stage(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_template_stage_stage ON pipeline_template_stage(stage_id);

CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    stages_snapshot TEXT NOT NULL DEFAULT '[]',
    variables_snapshot TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id),
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id),
    UNIQUE (template_id, version)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_template ON pipeline_snapshot(template_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);
