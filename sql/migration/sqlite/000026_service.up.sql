-- Domain: service
-- Tables: service
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    instance_key TEXT NOT NULL DEFAULT 'default',
    version_id TEXT NOT NULL,
    last_successful_version_id TEXT,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (environment_id) REFERENCES environment(id),
    FOREIGN KEY (version_id) REFERENCES version(id),
    FOREIGN KEY (last_successful_version_id) REFERENCES version(id),
    UNIQUE(application_id, environment_id, instance_key)
);

CREATE INDEX IF NOT EXISTS idx_service_version ON service(version_id);
CREATE INDEX IF NOT EXISTS idx_service_environment ON service(environment_id);
CREATE INDEX IF NOT EXISTS idx_service_app_env ON service(application_id, environment_id);
