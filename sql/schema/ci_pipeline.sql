-- Target CI schema used only by sqlc. It is intentionally not an executable
-- migration; the separate database cutover must implement this contract.
CREATE TABLE pipeline (
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
    description TEXT NOT NULL,
    variable_declarations TEXT NOT NULL,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE pipeline_stage (
    id TEXT PRIMARY KEY,
    pipeline_id TEXT NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    script TEXT NOT NULL,
    artifacts TEXT,
    depends_on TEXT NOT NULL,
    sort_order INTEGER NOT NULL,
    description TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE pipeline_snapshot (
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
    stages_snapshot TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL,
    created_at DATETIME NOT NULL
);

CREATE TABLE pipeline_run (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL,
    snapshot_id TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_version INTEGER NOT NULL,
    trigger TEXT NOT NULL,
    trigger_ref TEXT NOT NULL,
    variables_snapshot TEXT NOT NULL,
    status TEXT NOT NULL,
    retry_of TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    error_message TEXT,
    created_at DATETIME NOT NULL
);

CREATE TABLE pipeline_stage_run (
    id TEXT PRIMARY KEY,
    pipeline_run_id TEXT NOT NULL,
    stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    exit_code INTEGER,
    error_message TEXT
);

CREATE TABLE pipeline_run_version_binding (
    pipeline_run_id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    application_name TEXT NOT NULL,
    source_version_id TEXT NOT NULL,
    source_version_label TEXT NOT NULL,
    generated_version_id TEXT,
    generated_version_label TEXT
);

CREATE TABLE artifact (
    id TEXT PRIMARY KEY,
    project_id TEXT,
    pipeline_run_id TEXT NOT NULL,
    repository_id TEXT NOT NULL,
    repository_name TEXT NOT NULL,
    pipeline_id TEXT NOT NULL,
    pipeline_name TEXT NOT NULL,
    pipeline_stage_id TEXT NOT NULL,
    stage_name TEXT NOT NULL,
    collector TEXT NOT NULL,
    name TEXT NOT NULL,
    location TEXT,
    value TEXT,
    value_format TEXT,
    image_ref TEXT,
    local_image_sha256 TEXT,
    source_artifact_id TEXT,
    created_at DATETIME NOT NULL
);

CREATE TABLE version_component (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    artifact_id TEXT,
    artifact_name TEXT,
    artifact_image_ref TEXT,
    artifact_local_image_sha256 TEXT,
    artifact_source_commit_sha TEXT
);
