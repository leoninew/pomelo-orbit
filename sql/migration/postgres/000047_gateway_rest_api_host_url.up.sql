ALTER TABLE gateway_config
    ADD COLUMN rest_api_host_url TEXT NOT NULL DEFAULT 'http://127.0.0.1:8080';

DELETE FROM route r
USING application a, gateway_config gc
WHERE a.project_id = r.project_id
  AND gc.application_id = a.id
  AND r.name = 'traefik'
  AND r.protocol = 'http'
  AND r.domain = 'traefik-dashboard.' || gc.base_domain
  AND r.path_prefix = '/'
  AND r.target_url = 'http://' || lower(a.code) || '-traefik:8080'
  AND r.service_id IS NULL
  AND r.component_name IS NULL
  AND r.endpoint_protocol IS NULL
  AND r.endpoint_container_port IS NULL
  AND r.enabled = 0
  AND r.https_enabled = 0
  AND r.cert_type = 'manual'
  AND r.acme_challenge = 'http';
