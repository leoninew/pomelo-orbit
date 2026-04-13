-- 制品表补充仓库和模板冗余字段，支持全局制品列表查询
ALTER TABLE artifact ADD COLUMN repository_id TEXT NOT NULL DEFAULT '';
ALTER TABLE artifact ADD COLUMN repository_name TEXT NOT NULL DEFAULT '';
ALTER TABLE artifact ADD COLUMN template_id TEXT NOT NULL DEFAULT '';
ALTER TABLE artifact ADD COLUMN template_name TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS ix_artifact_repository_id ON artifact (repository_id);
