-- Domain: auth
-- Tables: login_history, login_attempt
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS login_history (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    username VARCHAR(255) NOT NULL,
    ip_address VARCHAR(64),
    user_agent TEXT,
    login_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    success TINYINT(1) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX idx_login_history_user_id ON login_history(user_id);
CREATE INDEX idx_login_history_login_at ON login_history(login_at DESC);

CREATE TABLE IF NOT EXISTS login_attempt (
    id VARCHAR(26) PRIMARY KEY,
    username VARCHAR(255),
    ip_address VARCHAR(64) NOT NULL,
    user_agent TEXT,
    success TINYINT(1) NOT NULL DEFAULT 0,
    created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
);

CREATE INDEX idx_login_attempt_ip_created ON login_attempt(ip_address, created_at);
CREATE INDEX idx_login_attempt_username_created ON login_attempt(username, created_at);
CREATE INDEX idx_login_attempt_created ON login_attempt(created_at);
