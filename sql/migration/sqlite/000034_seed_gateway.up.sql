-- gateway seed captured from data/mysql-transfer-20260816-103803.mysql.sql.
-- Upserts support databases previously initialized by the removed data-migration loader.

-- application: 1 row(s).
INSERT INTO "application" ("id", "name", "code", "kind", "project_id") VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'Traefik', 'traefik', 'gateway', '01KRRKK0K3T519ZQZES3M4QA9Z')
ON CONFLICT DO UPDATE SET
    "name" = excluded."name",
    "code" = excluded."code",
    "kind" = excluded."kind",
    "project_id" = excluded."project_id";

-- version: 1 row(s).
INSERT INTO "version" ("id", "application_id", "label", "status", "created_from_version_id", "note", "component_summary") VALUES
    ('01M01MP0950ECGK2DS1J4N4P50', '01M01MP0950ECGK2DS1FWYNC0B', 'traefik:3.6', 'unpublished', NULL, NULL, 'traefik')
ON CONFLICT DO UPDATE SET
    "application_id" = excluded."application_id",
    "label" = excluded."label",
    "status" = excluded."status",
    "created_from_version_id" = excluded."created_from_version_id",
    "note" = excluded."note",
    "component_summary" = excluded."component_summary";

-- gateway_config: 1 row(s).
INSERT INTO "gateway_config" ("application_id", "rest_api_url", "base_domain", "default_entrypoint", "tls_mode") VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'http://localhost:8080', 'lvh.me', 'web', 'none')
ON CONFLICT DO UPDATE SET
    "rest_api_url" = excluded."rest_api_url",
    "base_domain" = excluded."base_domain",
    "default_entrypoint" = excluded."default_entrypoint",
    "tls_mode" = excluded."tls_mode";

-- version_component: 1 row(s).
INSERT INTO "version_component" ("id", "version_id", "name", "image", "artifact_id", "command_json", "pull_policy", "restart_policy", "entrypoint_json", "artifact_name", "artifact_image_ref", "artifact_local_image_sha256", "artifact_source_commit_sha") VALUES
    ('01M01MP096R73Z3MG28P91CSPN', '01M01MP0950ECGK2DS1J4N4P50', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL)
ON CONFLICT DO UPDATE SET
    "version_id" = excluded."version_id",
    "name" = excluded."name",
    "image" = excluded."image",
    "artifact_id" = excluded."artifact_id",
    "command_json" = excluded."command_json",
    "pull_policy" = excluded."pull_policy",
    "restart_policy" = excluded."restart_policy",
    "entrypoint_json" = excluded."entrypoint_json",
    "artifact_name" = excluded."artifact_name",
    "artifact_image_ref" = excluded."artifact_image_ref",
    "artifact_local_image_sha256" = excluded."artifact_local_image_sha256",
    "artifact_source_commit_sha" = excluded."artifact_source_commit_sha";

-- version_component_endpoint: 3 row(s).
INSERT INTO "version_component_endpoint" ("component_id", "protocol", "container_port", "mode", "bind_address", "listen_port", "entrypoint", "path_prefix", "position") VALUES
    ('01M01MP096R73Z3MG28P91CSPN', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1)
ON CONFLICT DO UPDATE SET
    "mode" = excluded."mode",
    "bind_address" = excluded."bind_address",
    "listen_port" = excluded."listen_port",
    "entrypoint" = excluded."entrypoint",
    "path_prefix" = excluded."path_prefix",
    "position" = excluded."position";

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
    ('01M01MP096R73Z3MG28P91CSPN', 'controlled_file', 'acme.json', '/letsencrypt/acme.json', 0, 0, '{}', 0, '0600', 1, 2)
ON CONFLICT DO UPDATE SET
    "source_type" = excluded."source_type",
    "source" = excluded."source",
    "target" = excluded."target",
    "read_only" = excluded."read_only",
    "source_is_host_path" = excluded."source_is_host_path",
    "content" = excluded."content",
    "content_masked" = excluded."content_masked",
    "mode" = excluded."mode",
    "ignore_if_exists" = excluded."ignore_if_exists";

-- service: 1 row(s).
INSERT INTO "service" ("id", "application_id", "instance_key", "code", "version_id", "status") VALUES
    ('01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP0950ECGK2DS1FWYNC0B', 'default', 'traefik-default', '01M01MP0950ECGK2DS1J4N4P50', 'stopped')
ON CONFLICT DO UPDATE SET
    "application_id" = excluded."application_id",
    "instance_key" = excluded."instance_key",
    "code" = excluded."code",
    "version_id" = excluded."version_id",
    "status" = excluded."status";

-- service_component: 1 row(s).
INSERT INTO "service_component" ("id", "service_id", "source_version_component_id", "component_name", "entrypoint_json", "command_json", "pull_policy", "restart_policy", "status") VALUES
    ('01M01RHDXW3ZXC7YKNT8GDNQNB', '01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP096R73Z3MG28P91CSPN', 'traefik', NULL, NULL, NULL, NULL, 'active')
ON CONFLICT DO UPDATE SET
    "service_id" = excluded."service_id",
    "source_version_component_id" = excluded."source_version_component_id",
    "component_name" = excluded."component_name",
    "entrypoint_json" = excluded."entrypoint_json",
    "command_json" = excluded."command_json",
    "pull_policy" = excluded."pull_policy",
    "restart_policy" = excluded."restart_policy",
    "status" = excluded."status";

-- route: 1 row(s).
INSERT INTO "route" ("id", "name", "protocol", "domain", "path_prefix", "target_url", "listen_port", "service_id", "component_name", "endpoint_protocol", "endpoint_container_port", "enabled", "https_enabled", "cert_pem", "cert_key", "cert_type", "project_id") VALUES
    ('01M01ZNW6CPQCB7P5PN669HWJ9', 'traefik', 'http', 'traefik-dashboard.lvh.me', '/', 'traefik-default/traefik/http8080', NULL, '01M01RHDXW3ZXC7YKNT54RWM1M', 'traefik', 'http', 8080, 1, 0, NULL, NULL, 'manual', '01KRRKK0K3T519ZQZES3M4QA9Z')
ON CONFLICT DO UPDATE SET
    "name" = excluded."name",
    "protocol" = excluded."protocol",
    "domain" = excluded."domain",
    "path_prefix" = excluded."path_prefix",
    "target_url" = excluded."target_url",
    "listen_port" = excluded."listen_port",
    "service_id" = excluded."service_id",
    "component_name" = excluded."component_name",
    "endpoint_protocol" = excluded."endpoint_protocol",
    "endpoint_container_port" = excluded."endpoint_container_port",
    "enabled" = excluded."enabled",
    "https_enabled" = excluded."https_enabled",
    "cert_pem" = excluded."cert_pem",
    "cert_key" = excluded."cert_key",
    "cert_type" = excluded."cert_type",
    "project_id" = excluded."project_id";
