-- CI Phase 2: Webhook 支持
-- 为 Project 增加 webhook 相关字段

-- 增加 webhook_secret 字段（用于签名验证）
ALTER TABLE projects ADD COLUMN webhook_secret VARCHAR(255);

-- 增加 branch_filter 字段（可选，用于过滤分支，如 "main,develop"）
ALTER TABLE projects ADD COLUMN branch_filter VARCHAR(255);
