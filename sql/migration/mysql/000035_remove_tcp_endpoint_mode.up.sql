ALTER TABLE version_component_endpoint
    DROP CHECK chk_version_component_endpoint_mode,
    ADD CONSTRAINT chk_version_component_endpoint_mode CHECK (mode IN ('internal', 'local', 'host', 'gateway'));

ALTER TABLE service_component_endpoint
    DROP CHECK chk_service_component_endpoint_mode,
    ADD CONSTRAINT chk_service_component_endpoint_mode CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway'));
