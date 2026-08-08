-- Reverse 000031 async operation state machine.
ALTER TABLE pipeline_run
    DROP INDEX uq_pipeline_run_active_repository,
    DROP COLUMN active_repository_id;

ALTER TABLE deployment
    DROP INDEX uq_deployment_active_service,
    DROP COLUMN active_service_id;

ALTER TABLE pipeline_stage_run
    DROP INDEX uq_pipeline_stage_run_run_stage;
