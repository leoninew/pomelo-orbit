DELETE FROM service_component_endpoint;
DELETE FROM route WHERE service_id IS NOT NULL;
