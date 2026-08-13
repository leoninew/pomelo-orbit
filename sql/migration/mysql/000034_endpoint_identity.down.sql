ALTER TABLE route
    DROP CHECK chk_route_endpoint_container_port,
    DROP CHECK chk_route_endpoint_protocol,
    DROP COLUMN endpoint_container_port,
    DROP COLUMN endpoint_protocol,
    ADD COLUMN endpoint_name VARCHAR(255) NULL AFTER component_name;

ALTER TABLE service_component_endpoint
    DROP CHECK chk_service_component_endpoint_container_port,
    DROP CHECK chk_service_component_endpoint_protocol,
    DROP INDEX uq_service_component_endpoint_contract,
    DROP PRIMARY KEY,
    DROP COLUMN id,
    DROP COLUMN container_port,
    DROP COLUMN protocol,
    ADD COLUMN name VARCHAR(255) NOT NULL AFTER service_component_id,
    ADD PRIMARY KEY (service_component_id, name);

ALTER TABLE version_component_endpoint
    DROP PRIMARY KEY,
    ADD COLUMN name VARCHAR(255) NOT NULL AFTER component_id,
    ADD PRIMARY KEY (component_id, name),
    ADD UNIQUE KEY uq_version_component_endpoint_contract (component_id, protocol, container_port);
