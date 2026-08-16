-- gateway seed captured from data/mysql-transfer-20260816-103803.mysql.sql.
-- Upserts support databases previously initialized by the removed data-migration loader.

-- application: 1 row(s).
INSERT INTO `application` (`id`, `name`, `code`, `kind`, `project_id`) VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'Traefik', 'traefik', 'gateway', '01KRRKK0K3T519ZQZES3M4QA9Z')
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `code` = VALUES(`code`),
    `kind` = VALUES(`kind`),
    `project_id` = VALUES(`project_id`);

-- version: 1 row(s).
INSERT INTO `version` (`id`, `application_id`, `label`, `status`, `created_from_version_id`, `note`, `component_summary`) VALUES
    ('01M01MP0950ECGK2DS1J4N4P50', '01M01MP0950ECGK2DS1FWYNC0B', 'traefik:3.6', 'unpublished', NULL, NULL, 'traefik')
ON DUPLICATE KEY UPDATE
    `application_id` = VALUES(`application_id`),
    `label` = VALUES(`label`),
    `status` = VALUES(`status`),
    `created_from_version_id` = VALUES(`created_from_version_id`),
    `note` = VALUES(`note`),
    `component_summary` = VALUES(`component_summary`);

-- gateway_config: 1 row(s).
INSERT INTO `gateway_config` (`application_id`, `rest_api_url`, `base_domain`, `default_entrypoint`, `tls_mode`) VALUES
    ('01M01MP0950ECGK2DS1FWYNC0B', 'http://localhost:8080', 'lvh.me', 'web', 'none')
ON DUPLICATE KEY UPDATE
    `rest_api_url` = VALUES(`rest_api_url`),
    `base_domain` = VALUES(`base_domain`),
    `default_entrypoint` = VALUES(`default_entrypoint`),
    `tls_mode` = VALUES(`tls_mode`);

-- version_component: 1 row(s).
INSERT INTO `version_component` (`id`, `version_id`, `name`, `image`, `artifact_id`, `command_json`, `pull_policy`, `restart_policy`, `entrypoint_json`, `artifact_name`, `artifact_image_ref`, `artifact_local_image_sha256`, `artifact_source_commit_sha`) VALUES
    ('01M01MP096R73Z3MG28P91CSPN', '01M01MP0950ECGK2DS1J4N4P50', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL)
ON DUPLICATE KEY UPDATE
    `version_id` = VALUES(`version_id`),
    `name` = VALUES(`name`),
    `image` = VALUES(`image`),
    `artifact_id` = VALUES(`artifact_id`),
    `command_json` = VALUES(`command_json`),
    `pull_policy` = VALUES(`pull_policy`),
    `restart_policy` = VALUES(`restart_policy`),
    `entrypoint_json` = VALUES(`entrypoint_json`),
    `artifact_name` = VALUES(`artifact_name`),
    `artifact_image_ref` = VALUES(`artifact_image_ref`),
    `artifact_local_image_sha256` = VALUES(`artifact_local_image_sha256`),
    `artifact_source_commit_sha` = VALUES(`artifact_source_commit_sha`);

-- version_component_endpoint: 3 row(s).
INSERT INTO `version_component_endpoint` (`component_id`, `protocol`, `container_port`, `mode`, `bind_address`, `listen_port`, `entrypoint`, `path_prefix`, `position`) VALUES
    ('01M01MP096R73Z3MG28P91CSPN', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M01MP096R73Z3MG28P91CSPN', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1)
ON DUPLICATE KEY UPDATE
    `mode` = VALUES(`mode`),
    `bind_address` = VALUES(`bind_address`),
    `listen_port` = VALUES(`listen_port`),
    `entrypoint` = VALUES(`entrypoint`),
    `path_prefix` = VALUES(`path_prefix`),
    `position` = VALUES(`position`);

-- version_component_mount: 3 row(s).
INSERT INTO `version_component_mount` (`component_id`, `source_type`, `source`, `target`, `read_only`, `source_is_host_path`, `content`, `content_masked`, `mode`, `ignore_if_exists`, `position`) VALUES
    ('01M01MP096R73Z3MG28P91CSPN', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M01MP096R73Z3MG28P91CSPN', 'controlled_file', 'traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:\n  dashboard: true\n  insecure: true\n\nentryPoints:\n  web:\n    address: ":80"\n  websecure:\n    address: ":443"\nproviders:\n  docker:\n    endpoint: "unix:///var/run/docker.sock"\n    exposedByDefault: false\n    network: traefik\n  rest:\n    insecure: true\n\nlog:\n  level: INFO\n', 0, '0644', 0, 1),
    ('01M01MP096R73Z3MG28P91CSPN', 'controlled_file', 'acme.json', '/letsencrypt/acme.json', 0, 0, '{}', 0, '0600', 1, 2)
ON DUPLICATE KEY UPDATE
    `source_type` = VALUES(`source_type`),
    `source` = VALUES(`source`),
    `target` = VALUES(`target`),
    `read_only` = VALUES(`read_only`),
    `source_is_host_path` = VALUES(`source_is_host_path`),
    `content` = VALUES(`content`),
    `content_masked` = VALUES(`content_masked`),
    `mode` = VALUES(`mode`),
    `ignore_if_exists` = VALUES(`ignore_if_exists`);

-- service: 1 row(s).
INSERT INTO `service` (`id`, `application_id`, `instance_key`, `code`, `version_id`, `status`) VALUES
    ('01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP0950ECGK2DS1FWYNC0B', 'default', 'traefik-default', '01M01MP0950ECGK2DS1J4N4P50', 'stopped')
ON DUPLICATE KEY UPDATE
    `application_id` = VALUES(`application_id`),
    `instance_key` = VALUES(`instance_key`),
    `code` = VALUES(`code`),
    `version_id` = VALUES(`version_id`),
    `status` = VALUES(`status`);

-- service_component: 1 row(s).
INSERT INTO `service_component` (`id`, `service_id`, `source_version_component_id`, `component_name`, `entrypoint_json`, `command_json`, `pull_policy`, `restart_policy`, `status`) VALUES
    ('01M01RHDXW3ZXC7YKNT8GDNQNB', '01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP096R73Z3MG28P91CSPN', 'traefik', NULL, NULL, NULL, NULL, 'active')
ON DUPLICATE KEY UPDATE
    `service_id` = VALUES(`service_id`),
    `source_version_component_id` = VALUES(`source_version_component_id`),
    `component_name` = VALUES(`component_name`),
    `entrypoint_json` = VALUES(`entrypoint_json`),
    `command_json` = VALUES(`command_json`),
    `pull_policy` = VALUES(`pull_policy`),
    `restart_policy` = VALUES(`restart_policy`),
    `status` = VALUES(`status`);

-- route: 1 row(s).
INSERT INTO `route` (`id`, `name`, `protocol`, `domain`, `path_prefix`, `target_url`, `listen_port`, `service_id`, `component_name`, `endpoint_protocol`, `endpoint_container_port`, `enabled`, `https_enabled`, `cert_pem`, `cert_key`, `cert_type`, `project_id`) VALUES
    ('01M01ZNW6CPQCB7P5PN669HWJ9', 'traefik', 'http', 'traefik-dashboard.lvh.me', '/', 'traefik-default/traefik/http8080', NULL, '01M01RHDXW3ZXC7YKNT54RWM1M', 'traefik', 'http', 8080, 1, 0, NULL, NULL, 'manual', '01KRRKK0K3T519ZQZES3M4QA9Z')
ON DUPLICATE KEY UPDATE
    `name` = VALUES(`name`),
    `protocol` = VALUES(`protocol`),
    `domain` = VALUES(`domain`),
    `path_prefix` = VALUES(`path_prefix`),
    `target_url` = VALUES(`target_url`),
    `listen_port` = VALUES(`listen_port`),
    `service_id` = VALUES(`service_id`),
    `component_name` = VALUES(`component_name`),
    `endpoint_protocol` = VALUES(`endpoint_protocol`),
    `endpoint_container_port` = VALUES(`endpoint_container_port`),
    `enabled` = VALUES(`enabled`),
    `https_enabled` = VALUES(`https_enabled`),
    `cert_pem` = VALUES(`cert_pem`),
    `cert_key` = VALUES(`cert_key`),
    `cert_type` = VALUES(`cert_type`),
    `project_id` = VALUES(`project_id`);
