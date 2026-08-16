-- gateway seed captured from data/mysql-transfer-20260816-103803.mysql.sql.

-- application: 1 row(s).
INSERT INTO "application" ("id", "name", "code", "kind", "project_id") VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'Traefik', 'traefik', 'gateway', '01KRRKK0K3T519ZQZES3M4QA9Z');

-- version: 1 row(s).
INSERT INTO "version" ("id", "application_id", "label", "status", "created_from_version_id", "note", "component_summary") VALUES
    ('01M01MP0950ECGK2DS1J4N4P50', '01M01MP0950ECGK2DS1FWYNC0B', 'traefik:3.6', 'unpublished', NULL, NULL, 'traefik');

-- gateway_config: 1 row(s).
INSERT INTO "gateway_config" ("application_id", "rest_api_url", "base_domain", "default_entrypoint", "tls_mode") VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'http://localhost:8080', 'lvh.me', 'web', 'none');

-- version_component: 1 row(s).
INSERT INTO "version_component" ("id", "version_id", "name", "image", "artifact_id", "command_json", "pull_policy", "restart_policy", "entrypoint_json", "artifact_name", "artifact_image_ref", "artifact_local_image_sha256", "artifact_source_commit_sha") VALUES
    ('01M01MP096R73Z3MG28P91CSPN', '01M01MP0950ECGK2DS1J4N4P50', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL);

-- version_component_endpoint: 3 row(s).
INSERT INTO "version_component_endpoint" ("component_id", "protocol", "container_port", "mode", "bind_address", "listen_port", "entrypoint", "path_prefix", "position") VALUES
    ('01M01MP096R73Z3MG28P91CSPN', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1);

-- version_component_mount: 3 row(s).
INSERT INTO "version_component_mount" ("component_id", "source_type", "source", "target", "read_only", "source_is_host_path", "content", "content_masked", "mode", "ignore_if_exists", "position") VALUES
    ('01M01MP096R73Z3MG28P91CSPN', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M01MP096R73Z3MG28P91CSPN', 'controlled_file', 'traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false
    network: traefik
  rest:
    insecure: true

log:
  level: INFO
', 0, '0644', 0, 1),
    ('01M01MP096R73Z3MG28P91CSPN', 'controlled_file', 'acme.json', '/letsencrypt/acme.json', 0, 0, '{}', 0, '0600', 1, 2);

-- service: 1 row(s).
INSERT INTO "service" ("id", "application_id", "instance_key", "code", "version_id", "status") VALUES
    ('01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP0950ECGK2DS1FWYNC0B', 'default', 'traefik-default', '01M01MP0950ECGK2DS1J4N4P50', 'stopped');

-- service_component: 1 row(s).
INSERT INTO "service_component" ("id", "service_id", "source_version_component_id", "component_name", "entrypoint_json", "command_json", "pull_policy", "restart_policy", "status") VALUES
    ('01M01RHDXW3ZXC7YKNT8GDNQNB', '01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP096R73Z3MG28P91CSPN', 'traefik', NULL, NULL, NULL, NULL, 'active');

-- route: 1 row(s).
INSERT INTO "route" ("id", "name", "protocol", "domain", "path_prefix", "target_url", "listen_port", "service_id", "component_name", "endpoint_protocol", "endpoint_container_port", "enabled", "https_enabled", "cert_pem", "cert_key", "cert_type", "project_id") VALUES
    ('01M01ZNW6CPQCB7P5PN669HWJ9', 'traefik', 'http', 'traefik-dashboard.lvh.me', '/', 'traefik-default/traefik/http8080', NULL, '01M01RHDXW3ZXC7YKNT54RWM1M', 'traefik', 'http', 8080, 1, 0, NULL, NULL, 'manual', '01KRRKK0K3T519ZQZES3M4QA9Z');
