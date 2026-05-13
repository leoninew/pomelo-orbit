-- v0.7.0: OAuth 支持（通用设计）

-- 添加 OAuth 相关字段
ALTER TABLE user ADD COLUMN oauth_provider TEXT NOT NULL DEFAULT '';
ALTER TABLE user ADD COLUMN oauth_provider_id TEXT NOT NULL DEFAULT '';
ALTER TABLE user ADD COLUMN email TEXT DEFAULT NULL;
ALTER TABLE user ADD COLUMN auth_source TEXT NOT NULL DEFAULT 'password';

-- 创建复合索引（provider + provider_id）
CREATE INDEX IF NOT EXISTS idx_user_oauth_account ON user(oauth_provider, oauth_provider_id);
CREATE INDEX IF NOT EXISTS idx_user_email ON user(email);

-- 添加唯一约束（email 可为空，但非空时必须唯一）
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_email ON user(email) WHERE email IS NOT NULL;

-- 添加唯一约束（provider + provider_id 组合唯一，但都为空时不约束）
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_oauth_account ON user(oauth_provider, oauth_provider_id)
WHERE oauth_provider != '' AND oauth_provider_id != '';
