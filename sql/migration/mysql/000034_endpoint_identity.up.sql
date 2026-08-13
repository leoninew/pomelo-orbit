ALTER TABLE version_component_endpoint
    DROP PRIMARY KEY,
    DROP COLUMN name,
    DROP INDEX uq_version_component_endpoint_contract,
    ADD PRIMARY KEY (component_id, protocol, container_port);

ALTER TABLE service_component_endpoint
    DROP PRIMARY KEY,
    DROP COLUMN name,
    ADD COLUMN id VARCHAR(26) NOT NULL FIRST,
    ADD COLUMN protocol VARCHAR(16) NOT NULL AFTER service_component_id,
    ADD COLUMN container_port INT NOT NULL AFTER protocol,
    ADD PRIMARY KEY (id),
    ADD UNIQUE KEY uq_service_component_endpoint_contract (service_component_id, protocol, container_port),
    ADD CONSTRAINT chk_service_component_endpoint_protocol CHECK (protocol IN ('http', 'tcp')),
    ADD CONSTRAINT chk_service_component_endpoint_container_port CHECK (container_port BETWEEN 1 AND 65535);

ALTER TABLE route
    DROP COLUMN endpoint_name,
    ADD COLUMN endpoint_protocol VARCHAR(16) NULL AFTER component_name,
    ADD COLUMN endpoint_container_port INT NULL AFTER endpoint_protocol,
    ADD CONSTRAINT chk_route_endpoint_protocol CHECK (endpoint_protocol IS NULL OR endpoint_protocol IN ('http', 'tcp')),
    ADD CONSTRAINT chk_route_endpoint_container_port CHECK (endpoint_container_port IS NULL OR endpoint_container_port BETWEEN 1 AND 65535);
