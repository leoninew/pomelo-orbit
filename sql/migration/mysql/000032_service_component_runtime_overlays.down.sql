ALTER TABLE service_component
    DROP CONSTRAINT chk_service_component_restart_policy,
    DROP CONSTRAINT chk_service_component_pull_policy,
    DROP COLUMN restart_policy,
    DROP COLUMN pull_policy,
    DROP COLUMN command_json,
    DROP COLUMN entrypoint_json;
