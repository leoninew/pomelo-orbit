PRAGMA foreign_keys = OFF;

CREATE TABLE gateway_config_old (
    application_id TEXT PRIMARY KEY,
    rest_api_url TEXT NOT NULL,
    rest_ready_timeout_seconds INTEGER NOT NULL DEFAULT 20,
    base_domain TEXT NOT NULL,
    default_entrypoint TEXT NOT NULL DEFAULT 'web',
    tls_mode TEXT NOT NULL DEFAULT 'none',
    acme_profile TEXT NOT NULL DEFAULT '' CHECK (acme_profile IN ('', 'http', 'dns', 'http-dns')),
    acme_email TEXT NOT NULL DEFAULT '',
    dns_api_token TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE
);

INSERT INTO gateway_config_old SELECT * FROM gateway_config;
DROP TABLE gateway_config;
ALTER TABLE gateway_config_old RENAME TO gateway_config;

CREATE TABLE gateway_acme_profile_version_old (
    application_id TEXT NOT NULL,
    profile TEXT NOT NULL CHECK (profile IN ('base', 'http', 'dns', 'http-dns')),
    version_id TEXT NOT NULL,
    PRIMARY KEY (application_id, profile),
    UNIQUE (application_id, version_id),
    FOREIGN KEY (application_id) REFERENCES gateway_config(application_id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id) ON DELETE CASCADE
);

INSERT INTO gateway_acme_profile_version_old SELECT * FROM gateway_acme_profile_version;
DROP TABLE gateway_acme_profile_version;
ALTER TABLE gateway_acme_profile_version_old RENAME TO gateway_acme_profile_version;

CREATE TABLE service_component_old (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    source_version_component_id TEXT NOT NULL,
    component_name TEXT NOT NULL,
    entrypoint_json TEXT,
    command_json TEXT,
    pull_policy TEXT CHECK (pull_policy IS NULL OR pull_policy IN ('always', 'missing', 'never')),
    restart_policy TEXT CHECK (restart_policy IS NULL OR restart_policy IN ('no', 'unless-stopped')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active')),
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE (service_id, source_version_component_id),
    UNIQUE (service_id, component_name),
    FOREIGN KEY (service_id) REFERENCES service(id) ON DELETE CASCADE,
    FOREIGN KEY (source_version_component_id) REFERENCES version_component(id) ON DELETE CASCADE
);

INSERT INTO service_component_old SELECT * FROM service_component;
DROP TABLE service_component;
ALTER TABLE service_component_old RENAME TO service_component;
CREATE INDEX idx_service_component_service ON service_component(service_id);

CREATE TABLE service_old (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    application_id TEXT NOT NULL,
    code TEXT NOT NULL,
    version_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (application_id) REFERENCES application(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES version(id)
);

INSERT INTO service_old SELECT * FROM service;
DROP TABLE service;
ALTER TABLE service_old RENAME TO service;
CREATE UNIQUE INDEX uq_service_project_code ON service(project_id, code);
CREATE INDEX idx_service_version ON service(version_id);
CREATE INDEX idx_service_application ON service(application_id);

PRAGMA foreign_keys = ON;
