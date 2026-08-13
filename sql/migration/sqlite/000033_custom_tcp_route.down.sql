DROP INDEX IF EXISTS idx_route_tcp_listen;
ALTER TABLE route DROP COLUMN endpoint_name;
ALTER TABLE route DROP COLUMN component_name;
ALTER TABLE route DROP COLUMN service_id;
ALTER TABLE route DROP COLUMN listen_port;
ALTER TABLE route DROP COLUMN protocol;
