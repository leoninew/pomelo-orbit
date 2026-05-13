-- 用户状态字段
ALTER TABLE user ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;  -- 0=禁用, 1=启用

-- 索引：用于查询禁用账号列表
CREATE INDEX idx_user_is_active ON user(is_active);
