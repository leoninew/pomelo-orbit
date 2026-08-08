-- Domain: pipeline stage template library
--
-- Existing pipeline stages have no template-source snapshot and cannot be
-- represented by the new model. Keep them for the explicitly separate
-- offline data migration; runtime code must not read this legacy table.
DROP INDEX IF EXISTS idx_pipeline_stage_pipeline;
DROP INDEX IF EXISTS idx_pipeline_stage_pipeline_sort;
ALTER TABLE pipeline_stage RENAME TO pipeline_stage_legacy_000030;

CREATE TABLE pipeline_stage (
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

CREATE UNIQUE INDEX uq_pipeline_stage_template_project_name
    ON pipeline_stage(project_id, name) WHERE kind = 'template';
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
