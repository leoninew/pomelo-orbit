DROP TABLE IF EXISTS pipeline_stage_reference;
DROP INDEX IF EXISTS uq_pipeline_stage_template_project_name;
DROP INDEX IF EXISTS idx_pipeline_stage_project_kind_name;
DROP TABLE IF EXISTS pipeline_stage;
ALTER TABLE pipeline_stage_legacy_000030 RENAME TO pipeline_stage;
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_pipeline ON pipeline_stage(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_stage_pipeline_sort ON pipeline_stage(pipeline_id, sort_order);
