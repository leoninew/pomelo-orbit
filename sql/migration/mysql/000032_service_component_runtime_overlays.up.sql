ALTER TABLE service_component
    ADD COLUMN entrypoint_json TEXT NULL AFTER component_name,
    ADD COLUMN command_json TEXT NULL AFTER entrypoint_json,
    ADD COLUMN pull_policy VARCHAR(32) NULL AFTER command_json,
    ADD COLUMN restart_policy VARCHAR(32) NULL AFTER pull_policy,
    ADD CONSTRAINT chk_service_component_pull_policy CHECK (pull_policy IS NULL OR pull_policy IN ('always', 'missing', 'never')),
    ADD CONSTRAINT chk_service_component_restart_policy CHECK (restart_policy IS NULL OR restart_policy IN ('no', 'unless-stopped'));
