-- Domain: pipeline
CREATE TABLE IF NOT EXISTS pipeline (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26),
    kind VARCHAR(32) NOT NULL,
    source_pipeline_id VARCHAR(26),
    source_template_name VARCHAR(255),
    source_template_version INT,
    application_id VARCHAR(26),
    application_name VARCHAR(255),
    repository_id VARCHAR(26),
    repository_name VARCHAR(255),
    version_fork_strategy VARCHAR(16),
    fixed_version_id VARCHAR(26),
    fixed_version_label VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    variable_declarations LONGTEXT NOT NULL,
    version INT NOT NULL DEFAULT 1,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_pipeline_project_name (project_id, name)
) ENGINE=InnoDB;

CREATE INDEX idx_pipeline_project ON pipeline(project_id);
CREATE INDEX idx_pipeline_kind ON pipeline(kind);
CREATE INDEX idx_pipeline_source_pipeline ON pipeline(source_pipeline_id);
CREATE INDEX idx_pipeline_application ON pipeline(application_id);
CREATE INDEX idx_pipeline_repository ON pipeline(repository_id);

CREATE TABLE IF NOT EXISTS pipeline_stage (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26) NOT NULL,
    kind VARCHAR(16) NOT NULL,
    pipeline_id VARCHAR(26),
    name VARCHAR(255) NOT NULL,
    image VARCHAR(512) NOT NULL,
    script LONGTEXT NOT NULL,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    version INT,
    source_template_stage_id VARCHAR(26),
    source_template_stage_name VARCHAR(255),
    source_template_stage_version INT,
    source_template_stage_description VARCHAR(1024),
    artifacts LONGTEXT,
    depends_on LONGTEXT,
    sort_order INT,
    template_name VARCHAR(255) GENERATED ALWAYS AS (
        CASE WHEN kind = 'template' THEN name ELSE NULL END
    ) STORED,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_pipeline_stage_pipeline FOREIGN KEY (pipeline_id) REFERENCES pipeline(id) ON DELETE CASCADE,
    CONSTRAINT uq_pipeline_stage_pipeline_name UNIQUE (pipeline_id, name)
) ENGINE=InnoDB;

CREATE UNIQUE INDEX uq_pipeline_stage_template_project_name ON pipeline_stage(project_id, template_name);
CREATE INDEX idx_pipeline_stage_project_kind_name ON pipeline_stage(project_id, kind, name);

CREATE TABLE pipeline_stage_reference (
    id VARCHAR(26) PRIMARY KEY,
    pipeline_id VARCHAR(26) NOT NULL,
    source_template_stage_id VARCHAR(26) NOT NULL,
    source_template_stage_name VARCHAR(255) NOT NULL,
    source_template_stage_version INT NOT NULL,
    source_template_stage_description VARCHAR(1024) NOT NULL,
    name VARCHAR(255) NOT NULL,
    image VARCHAR(512) NOT NULL,
    script LONGTEXT NOT NULL,
    description VARCHAR(1024) NOT NULL DEFAULT '',
    artifacts LONGTEXT NOT NULL,
    depends_on LONGTEXT NOT NULL,
    sort_order INT NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_pipeline_stage_reference_pipeline FOREIGN KEY (pipeline_id) REFERENCES pipeline(id) ON DELETE CASCADE,
    CONSTRAINT uq_pipeline_stage_reference_name UNIQUE (pipeline_id, name)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS pipeline_snapshot (
    id VARCHAR(26) PRIMARY KEY,
    project_id VARCHAR(26),
    pipeline_id VARCHAR(26) NOT NULL,
    pipeline_name VARCHAR(255) NOT NULL,
    pipeline_version INT NOT NULL,
    source_pipeline_id VARCHAR(26) NOT NULL,
    source_template_name VARCHAR(255) NOT NULL,
    source_template_version INT NOT NULL,
    application_id VARCHAR(26),
    application_name VARCHAR(255),
    repository_id VARCHAR(26) NOT NULL,
    repository_name VARCHAR(255) NOT NULL,
    version_fork_strategy VARCHAR(16),
    fixed_version_id VARCHAR(26),
    fixed_version_label VARCHAR(255),
    stages_snapshot LONGTEXT NOT NULL,
    variables_snapshot LONGTEXT NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    UNIQUE KEY uq_pipeline_snapshot_pipeline_version (pipeline_id, pipeline_version)
) ENGINE=InnoDB;

CREATE INDEX idx_pipeline_snapshot_pipeline ON pipeline_snapshot(pipeline_id);
CREATE INDEX idx_pipeline_snapshot_project ON pipeline_snapshot(project_id);
