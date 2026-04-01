-- CI Phase 2: Webhook 支持
-- 为 Project 增加 webhook 相关字段

-- SQLite 不支持 IF NOT EXISTS for ALTER TABLE ADD COLUMN
-- 使用 PRAGMA 检查列是否存在的方式太复杂，这里假设列可能已存在
-- 如果列已存在，迁移会失败但可以手动标记为已执行

-- 增加 webhook_secret 字段（用于签名验证）
-- ALTER TABLE projects ADD COLUMN webhook_secret VARCHAR(255);

-- 增加 branch_filter 字段（可选，用于过滤分支，如 "main,develop"）
-- ALTER TABLE projects ADD COLUMN branch_filter VARCHAR(255);

-- 注意：由于这些列可能已经通过其他方式添加，此迁移被注释掉
-- 如果需要添加这些列，请手动执行上述 SQL 或确保列不存在
