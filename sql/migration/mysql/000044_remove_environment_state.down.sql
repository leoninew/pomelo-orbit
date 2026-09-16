ALTER TABLE environment
    ADD COLUMN state VARCHAR(16) NOT NULL DEFAULT 'active' AFTER code,
    ADD CONSTRAINT chk_environment_state CHECK (state IN ('active', 'disabled'));
