-- Reverse 000031 async operation state machine.
DROP INDEX IF EXISTS uq_pipeline_run_active_repository;
DROP INDEX IF EXISTS uq_deployment_active_service;
DROP INDEX IF EXISTS uq_pipeline_stage_run_run_stage;
