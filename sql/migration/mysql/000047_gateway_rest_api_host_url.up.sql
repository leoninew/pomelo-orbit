ALTER TABLE gateway_config ADD COLUMN rest_api_host_url VARCHAR(512) NULL AFTER rest_api_url;
UPDATE gateway_config SET rest_api_host_url = 'http://127.0.0.1:8080' WHERE rest_api_host_url IS NULL;
ALTER TABLE gateway_config MODIFY COLUMN rest_api_host_url VARCHAR(512) NOT NULL;

DELETE r
FROM route r
INNER JOIN application a ON a.project_id = r.project_id
INNER JOIN gateway_config gc ON gc.application_id = a.id
WHERE r.name = 'traefik'
  AND r.protocol = 'http'
  AND r.domain = CONCAT('traefik-dashboard.', gc.base_domain)
  AND r.path_prefix = '/'
  AND r.target_url = CONCAT('http://', LOWER(a.code), '-traefik:8080')
  AND r.service_id IS NULL
  AND r.component_name IS NULL
  AND r.endpoint_protocol IS NULL
  AND r.endpoint_container_port IS NULL
  AND r.enabled = 0
  AND r.https_enabled = 0
  AND r.cert_type = 'manual'
  AND r.acme_challenge = 'http';
