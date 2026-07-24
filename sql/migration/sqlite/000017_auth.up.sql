-- Domain: auth
-- Tables: login_history, login_attempt
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS login_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    login_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    success INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_login_history_user_id ON login_history(user_id);
CREATE INDEX IF NOT EXISTS idx_login_history_login_at ON login_history(login_at DESC);

CREATE TABLE IF NOT EXISTS login_attempt (
    id TEXT PRIMARY KEY,
    username TEXT,
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_login_attempt_ip_created ON login_attempt(ip_address, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_username_created ON login_attempt(username, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_created ON login_attempt(created_at);
