-- 登录尝试记录表
CREATE TABLE IF NOT EXISTS login_attempt (
    id TEXT PRIMARY KEY,
    username TEXT,  -- 可为空，因为用户名可能不存在
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    success INTEGER NOT NULL DEFAULT 0,  -- 0=失败, 1=成功
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

-- 索引：用于速率限制查询
CREATE INDEX IF NOT EXISTS idx_login_attempt_ip_created ON login_attempt(ip_address, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_username_created ON login_attempt(username, created_at);
CREATE INDEX IF NOT EXISTS idx_login_attempt_created ON login_attempt(created_at);
