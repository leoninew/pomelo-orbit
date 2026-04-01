-- CI Phase 4: 重试机制
-- 为 PipelineRun 增加 retry_of 字段

-- 增加 retry_of 字段（引用原 Run ID）
ALTER TABLE pipeline_runs ADD COLUMN retry_of VARCHAR(26);

-- 注意：SQLite 不支持 ALTER TABLE ADD CONSTRAINT 添加外键
-- 外键约束需要在 CREATE TABLE 时定义，这里仅添加列和索引
-- 应用层需要确保 retry_of 引用的 run 存在

-- 为 retry_of 创建索引（用于查询原始 run）
CREATE INDEX idx_pipeline_runs_retry_of ON pipeline_runs(retry_of);
