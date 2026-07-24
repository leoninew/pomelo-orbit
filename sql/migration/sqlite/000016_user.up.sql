-- Domain: user
-- Tables: user, user_role
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS user (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
    last_login_at DATETIME,
    oauth_provider TEXT NOT NULL DEFAULT '',
    oauth_provider_id TEXT NOT NULL DEFAULT '',
    email TEXT DEFAULT NULL,
    auth_source TEXT NOT NULL DEFAULT 'password',
    status TEXT NOT NULL DEFAULT 'enabled' CHECK (status IN ('enabled', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_user_email ON user(email);
CREATE INDEX IF NOT EXISTS idx_user_status ON user(status);
CREATE INDEX IF NOT EXISTS idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_email ON user(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_oauth_account ON user(oauth_provider, oauth_provider_id)
WHERE oauth_provider != '' AND oauth_provider_id != '';

CREATE TABLE IF NOT EXISTS user_role (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_role_user_id ON user_role(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role_id ON user_role(role_id);
