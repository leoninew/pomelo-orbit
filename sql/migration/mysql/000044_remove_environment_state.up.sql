ALTER TABLE environment
    DROP CHECK chk_environment_state,
    DROP COLUMN state;
