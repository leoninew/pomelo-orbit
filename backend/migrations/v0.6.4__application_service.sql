-- v0.6.4: 应用 service 级配置

CREATE TABLE IF NOT EXISTS application_service (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    image TEXT,
    environment TEXT,
    volumes TEXT,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    UNIQUE(application_id, service_name)
);

CREATE INDEX IF NOT EXISTS idx_application_service_app ON application_service(application_id);
