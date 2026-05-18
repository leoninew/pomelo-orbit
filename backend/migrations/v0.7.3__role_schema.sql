-- v0.7.3: Role management schema

CREATE TABLE IF NOT EXISTS role (
    id TEXT PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_role_code ON role(code);
CREATE INDEX IF NOT EXISTS idx_role_name ON role(name);
CREATE INDEX IF NOT EXISTS idx_role_is_active ON role(is_active);
