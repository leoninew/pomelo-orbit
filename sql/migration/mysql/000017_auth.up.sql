-- Domain: auth
-- Tables: login_history
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

CREATE TABLE IF NOT EXISTS login_history (
    id VARCHAR(26) PRIMARY KEY,
    user_id VARCHAR(26) NOT NULL,
    username VARCHAR(255) NOT NULL,
    ip_address VARCHAR(64),
    user_agent TEXT,
    login_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    success TINYINT(1) NOT NULL
);

CREATE INDEX idx_login_history_user_id ON login_history(user_id);
CREATE INDEX idx_login_history_login_at ON login_history(login_at DESC);
