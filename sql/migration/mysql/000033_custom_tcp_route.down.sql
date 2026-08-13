ALTER TABLE service_component_endpoint
    DROP CHECK chk_service_component_endpoint_mode,
    ADD CONSTRAINT chk_service_component_endpoint_mode CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp'));

ALTER TABLE version_component_endpoint
    DROP CHECK chk_version_component_endpoint_mode,
    ADD CONSTRAINT chk_version_component_endpoint_mode CHECK (mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp'));

ALTER TABLE route
    DROP INDEX idx_route_tcp_listen,
    DROP CHECK chk_route_listen_port,
    DROP CHECK chk_route_protocol,
    DROP COLUMN endpoint_name,
    DROP COLUMN component_name,
    DROP COLUMN service_id,
    DROP COLUMN listen_port,
    DROP COLUMN protocol;
