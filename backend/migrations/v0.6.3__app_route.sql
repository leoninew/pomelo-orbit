-- v0.7.0: 应用路由托管

ALTER TABLE application ADD COLUMN route_managed INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS application_route (
    id TEXT PRIMARY KEY,
    application_id TEXT NOT NULL,
    service_name TEXT NOT NULL,
    domain TEXT NOT NULL,
    port INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_app_route_app ON application_route(application_id);
