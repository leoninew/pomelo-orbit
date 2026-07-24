-- Domain: pipeline
-- Tables: pipeline_template, pipeline_stage, pipeline_template_stage, pipeline_snapshot
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

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

CREATE TABLE IF NOT EXISTS pipeline_stage (
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

CREATE INDEX idx_pipeline_stage_name ON pipeline_stage(name);
CREATE INDEX idx_pipeline_stage_project ON pipeline_stage(project_id);

CREATE TABLE IF NOT EXISTS pipeline_template_stage (
    id VARCHAR(26) PRIMARY KEY,
    template_id VARCHAR(26) NOT NULL,
    stage_id VARCHAR(26) NOT NULL,
    stage_name VARCHAR(255) NOT NULL,
    stage_version INT NOT NULL DEFAULT 1,
    depends_on VARCHAR(4096) NOT NULL DEFAULT ('[]'),
    sort_order INT NOT NULL DEFAULT 0,
    FOREIGN KEY (template_id) REFERENCES pipeline_template(id) ON DELETE CASCADE,
    FOREIGN KEY (stage_id) REFERENCES pipeline_stage(id),
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
