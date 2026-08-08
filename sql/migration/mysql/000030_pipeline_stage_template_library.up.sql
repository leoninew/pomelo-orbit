-- Domain: pipeline stage template library
-- Preserve old rows for the separately-run offline data migration. The new
-- application does not read or adapt this legacy table.
RENAME TABLE pipeline_stage TO pipeline_stage_legacy_000030;

CREATE TABLE pipeline_stage (
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
