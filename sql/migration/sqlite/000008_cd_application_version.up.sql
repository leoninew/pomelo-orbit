-- CD application versioning: version / component / service; drop legacy config tables.

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

CREATE TABLE IF NOT EXISTS component (
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

CREATE INDEX IF NOT EXISTS idx_component_version ON component(version_id);

CREATE TABLE IF NOT EXISTS service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL UNIQUE,
    version_id TEXT NOT NULL,
    last_successful_version_id TEXT,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id),
    FOREIGN KEY (last_successful_version_id) REFERENCES version(id)
);

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);

ALTER TABLE deployment ADD COLUMN version_id TEXT REFERENCES version(id);
ALTER TABLE deployment ADD COLUMN service_id TEXT REFERENCES service(id);
ALTER TABLE deployment ADD COLUMN options_json TEXT;

DROP TABLE IF EXISTS application_config_file;
DROP TABLE IF EXISTS application_service;

-- SQLite 3.35+: drop application runtime status (Service is source of truth).
ALTER TABLE application DROP COLUMN status;
