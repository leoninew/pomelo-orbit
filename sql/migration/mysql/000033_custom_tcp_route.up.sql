ALTER TABLE route
    ADD COLUMN protocol VARCHAR(16) NOT NULL DEFAULT 'http' AFTER name,
    ADD COLUMN listen_port INT NULL AFTER target_url,
    ADD COLUMN service_id VARCHAR(26) NULL AFTER listen_port,
    ADD COLUMN component_name VARCHAR(255) NULL AFTER service_id,
    ADD COLUMN endpoint_name VARCHAR(255) NULL AFTER component_name,
    ADD CONSTRAINT chk_route_protocol CHECK (protocol IN ('http', 'tcp')),
    ADD CONSTRAINT chk_route_listen_port CHECK (listen_port IS NULL OR listen_port BETWEEN 1 AND 65535),
    ADD KEY idx_route_tcp_listen (protocol, listen_port);

ALTER TABLE version_component_endpoint
    DROP CHECK chk_version_component_endpoint_mode,
    ADD CONSTRAINT chk_version_component_endpoint_mode CHECK (mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp', 'gateway', 'tcp'));

ALTER TABLE service_component_endpoint
    DROP CHECK chk_service_component_endpoint_mode,
    ADD CONSTRAINT chk_service_component_endpoint_mode CHECK (mode IS NULL OR mode IN ('internal', 'local', 'host', 'gateway_http', 'gateway_tcp', 'gateway', 'tcp'));
