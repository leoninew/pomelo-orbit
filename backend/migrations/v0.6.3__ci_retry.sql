-- CI Phase 4: 重试机制
-- 为 PipelineRun 增加 retry_of 字段

-- 增加 retry_of 字段（引用原 Run ID）
ALTER TABLE pipeline_runs ADD COLUMN retry_of VARCHAR(26);

-- 添加外键约束（禁止删除被引用的 run，保持重试链完整性）
ALTER TABLE pipeline_runs ADD CONSTRAINT fk_pipeline_runs_retry_of 
    FOREIGN KEY (retry_of) REFERENCES pipeline_runs(id) ON DELETE RESTRICT;

-- 为 retry_of 创建索引（用于查询原始 run）
CREATE INDEX idx_pipeline_runs_retry_of ON pipeline_runs(retry_of);
