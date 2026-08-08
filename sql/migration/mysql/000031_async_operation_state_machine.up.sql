-- Domain: async operation state machine (ordered after 000030 pipeline stage template library)
ALTER TABLE pipeline_stage_run
    ADD CONSTRAINT uq_pipeline_stage_run_run_stage UNIQUE (pipeline_run_id, stage_id);

ALTER TABLE deployment
    ADD COLUMN active_service_id VARCHAR(26)
        GENERATED ALWAYS AS (
            CASE WHEN status IN ('waiting_to_run', 'running') THEN service_id ELSE NULL END
        ) STORED,
    ADD CONSTRAINT uq_deployment_active_service UNIQUE (active_service_id);

ALTER TABLE pipeline_run
    ADD COLUMN active_repository_id VARCHAR(26)
        GENERATED ALWAYS AS (
            CASE WHEN status IN ('waiting_to_run', 'running') THEN repository_id ELSE NULL END
        ) STORED,
    ADD CONSTRAINT uq_pipeline_run_active_repository UNIQUE (active_repository_id);
