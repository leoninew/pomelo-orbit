ALTER TABLE service_component ADD COLUMN entrypoint_json TEXT;
ALTER TABLE service_component ADD COLUMN command_json TEXT;
ALTER TABLE service_component ADD COLUMN pull_policy TEXT CHECK (pull_policy IS NULL OR pull_policy IN ('always', 'missing', 'never'));
ALTER TABLE service_component ADD COLUMN restart_policy TEXT CHECK (restart_policy IS NULL OR restart_policy IN ('no', 'unless-stopped'));
