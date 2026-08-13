-- Run after schema migration 000033 and before deploying the binary that
-- accepts only the new endpoint modes. This file is intentionally not part of
-- the application's automatic data-migration path.

UPDATE version_component_endpoint
SET mode = CASE mode
    WHEN 'gateway_http' THEN 'gateway'
    WHEN 'gateway_tcp' THEN 'internal'
    WHEN 'tcp' THEN 'internal'
    ELSE mode
END
WHERE mode IN ('gateway_http', 'gateway_tcp', 'tcp');

UPDATE service_component_endpoint
SET mode = CASE mode
    WHEN 'gateway_http' THEN 'gateway'
    WHEN 'gateway_tcp' THEN 'internal'
    WHEN 'tcp' THEN 'internal'
    ELSE mode
END
WHERE mode IN ('gateway_http', 'gateway_tcp', 'tcp');
