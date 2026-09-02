-- Domain: user
-- Tables: user, user_role
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS "user" (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    last_login_at TIMESTAMP,
    oauth_provider TEXT NOT NULL DEFAULT '',
    oauth_provider_id TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL,
    auth_source TEXT NOT NULL DEFAULT 'password',
    status TEXT NOT NULL DEFAULT 'enabled'
);

CREATE INDEX IF NOT EXISTS idx_user_email ON "user"(email);
CREATE INDEX IF NOT EXISTS idx_user_status ON "user"(status);
CREATE INDEX IF NOT EXISTS idx_user_oauth_account ON "user"(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_email ON "user"(email);

CREATE TABLE IF NOT EXISTS user_role (
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    PRIMARY KEY (user_id, role_id),
    FOREIGN KEY (role_id) REFERENCES role(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_role_user_id ON user_role(user_id);
CREATE INDEX IF NOT EXISTS idx_user_role_role_id ON user_role(role_id);
