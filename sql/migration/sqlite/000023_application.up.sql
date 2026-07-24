-- Domain: application
-- Tables: application, version, version_component, version_expose
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS application (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'standard',
    image_pull_policy TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    project_id TEXT REFERENCES project(id)
);

CREATE INDEX IF NOT EXISTS idx_application_project ON application(project_id);

CREATE TABLE IF NOT EXISTS version (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    label TEXT NOT NULL,
    status TEXT NOT NULL,
    env_json TEXT,
    created_from_version_id TEXT,
    note TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (created_from_version_id) REFERENCES version(id),
    UNIQUE(application_id, label)
);

CREATE INDEX IF NOT EXISTS idx_version_application ON version(application_id);
CREATE INDEX IF NOT EXISTS idx_version_status ON version(status);

CREATE TABLE IF NOT EXISTS version_component (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    name TEXT NOT NULL,
    image TEXT NOT NULL,
    command_json TEXT,
    args_json TEXT,
    env_json TEXT,
    ports_json TEXT,
    mounts_json TEXT,
    networks_json TEXT,
    depends_on_json TEXT,
    healthcheck_json TEXT,
    resources_json TEXT,
    pull_policy TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, name)
);

CREATE INDEX IF NOT EXISTS idx_version_component_version ON version_component(version_id);

CREATE TABLE IF NOT EXISTS version_expose (
    id TEXT PRIMARY KEY,
    version_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    container_port INTEGER NOT NULL,
    path_prefix TEXT,
    access TEXT NOT NULL DEFAULT 'public',
    listen_port INTEGER,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE,
    UNIQUE(version_id, component_name, protocol, container_port)
);

CREATE INDEX IF NOT EXISTS idx_version_expose_version ON version_expose(version_id);
