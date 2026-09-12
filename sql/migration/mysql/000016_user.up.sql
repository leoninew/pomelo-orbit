-- Domain: user
-- Tables: user, user_role
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS user (
    id VARCHAR(26) PRIMARY KEY,
    username VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    last_login_at DATETIME(3),
    oauth_provider VARCHAR(64) NOT NULL DEFAULT '',
    oauth_provider_id VARCHAR(255) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL,
    auth_source VARCHAR(64) NOT NULL DEFAULT 'password',
    status VARCHAR(32) NOT NULL DEFAULT 'enabled'
);

CREATE INDEX idx_user_email ON user(email);
CREATE INDEX idx_user_status ON user(status);
CREATE INDEX idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE UNIQUE INDEX uq_user_email ON user(email);

CREATE TABLE IF NOT EXISTS user_role (
    user_id VARCHAR(26) NOT NULL,
    role_id VARCHAR(26) NOT NULL,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_role_user_id ON user_role(user_id);
CREATE INDEX idx_user_role_role_id ON user_role(role_id);
