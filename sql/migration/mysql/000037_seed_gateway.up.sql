-- Replace the incomplete historical Gateway seed with the current complete topology.
DELETE FROM `route` WHERE `id` = '01M01ZNW6CPQCB7P5PN669HWJ9';
DELETE FROM `service_component` WHERE `service_id` = '01M01RHDXW3ZXC7YKNT54RWM1M';
DELETE FROM `service` WHERE `application_id` = '01M01MP0950ECGK2DS1FWYNC0B';
DELETE FROM `application` WHERE `id` = '01M01MP0950ECGK2DS1FWYNC0B';

INSERT INTO `application` (`id`, `name`, `code`, `kind`, `project_id`) VALUES
    ('01M10RRA8F863EJ2N9TC3Z2EC1', 'Traefik', 'traefik', 'standard', '01KRRKK0K3T519ZQZES3M4QA9Z');

INSERT INTO `version` (`id`, `application_id`, `label`, `status`, `created_from_version_id`, `note`, `component_summary`) VALUES
    ('01M10RRA8F863EJ2N9TDN9JSBW', '01M10RRA8F863EJ2N9TC3Z2EC1', 'traefik:3.6', 'unpublished', NULL, NULL, 'traefik'),
    ('01M10RRA8F863EJ2N9TFH32002', '01M10RRA8F863EJ2N9TC3Z2EC1', 'traefik:3.6 (http)', 'unpublished', NULL, NULL, 'traefik'),
    ('01M10RRA8F863EJ2N9TH6R2HSB', '01M10RRA8F863EJ2N9TC3Z2EC1', 'traefik:3.6 (dns)', 'unpublished', NULL, NULL, 'traefik'),
    ('01M10RRA8F863EJ2N9TKGXYZXE', '01M10RRA8F863EJ2N9TC3Z2EC1', 'traefik:3.6 (http-dns)', 'unpublished', NULL, NULL, 'traefik');

INSERT INTO `gateway_config` (
    `application_id`, `traefik_component_name`, `rest_api_url`, `rest_ready_timeout_seconds`, `base_domain`,
    `default_entrypoint`, `tls_mode`, `acme_profile`, `acme_email`, `dns_api_token`
) VALUES (
    '01M10RRA8F863EJ2N9TC3Z2EC1', 'traefik', 'http://localhost:8080', 20, 'lvh.me',
    'web', 'none', '', '', ''
);

INSERT INTO `gateway_acme_profile_version` (`application_id`, `profile`, `version_id`) VALUES
    ('01M10RRA8F863EJ2N9TC3Z2EC1', 'base', '01M10RRA8F863EJ2N9TDN9JSBW'),
    ('01M10RRA8F863EJ2N9TC3Z2EC1', 'http', '01M10RRA8F863EJ2N9TFH32002'),
    ('01M10RRA8F863EJ2N9TC3Z2EC1', 'dns', '01M10RRA8F863EJ2N9TH6R2HSB'),
    ('01M10RRA8F863EJ2N9TC3Z2EC1', 'http-dns', '01M10RRA8F863EJ2N9TKGXYZXE');

INSERT INTO `version_component` (
    `id`, `version_id`, `name`, `image`, `artifact_id`, `command_json`, `pull_policy`, `restart_policy`,
    `entrypoint_json`, `artifact_name`, `artifact_image_ref`, `artifact_local_image_sha256`, `artifact_source_commit_sha`
) VALUES
    ('01M10RRA8F863EJ2N9TN8CWTEY', '01M10RRA8F863EJ2N9TDN9JSBW', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', 'unless-stopped', '[]', NULL, NULL, NULL, NULL),
    ('01M10RRA8F863EJ2N9TPPNAAN1', '01M10RRA8F863EJ2N9TFH32002', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL),
    ('01M10RRA8F863EJ2N9TSRZBPC3', '01M10RRA8F863EJ2N9TH6R2HSB', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL),
    ('01M10RRA8F863EJ2N9TT9T3NK7', '01M10RRA8F863EJ2N9TKGXYZXE', 'traefik', 'traefik:3.6', NULL, '[]', 'missing', NULL, '[]', NULL, NULL, NULL, NULL);

INSERT INTO `version_component_endpoint` (
    `component_id`, `protocol`, `container_port`, `mode`, `bind_address`, `listen_port`, `entrypoint`, `path_prefix`, `position`
) VALUES
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1),
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'tcp', 80, 'host', '0.0.0.0', 80, NULL, NULL, 0),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'tcp', 443, 'host', '0.0.0.0', 443, NULL, NULL, 1),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'http', 8080, 'local', '127.0.0.1', 8080, NULL, NULL, 2);

INSERT INTO `version_component_mount` (
    `component_id`, `source_type`, `source`, `target`, `read_only`, `source_is_host_path`, `content`, `content_masked`, `mode`, `ignore_if_exists`, `position`
) VALUES
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'controlled_file', './traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:
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
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'directory', './gateway/certs', '/etc/traefik/certs', 0, 0, NULL, 0, '', 0, 2),
    ('01M10RRA8F863EJ2N9TN8CWTEY', 'directory', './gateway/acme', '/letsencrypt', 0, 0, NULL, 0, '', 0, 3),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'controlled_file', './traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:
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

certificatesResolvers:
  letsencrypt:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web
', 0, '0644', 0, 1),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'directory', './gateway/certs', '/etc/traefik/certs', 0, 0, NULL, 0, '', 0, 2),
    ('01M10RRA8F863EJ2N9TPPNAAN1', 'directory', './gateway/acme', '/letsencrypt', 0, 0, NULL, 0, '', 0, 3),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'controlled_file', './traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:
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

certificatesResolvers:
  letsencrypt-dns:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      dnsChallenge:
        provider: cloudflare
        resolvers:
          - "1.1.1.1:53"
          - "8.8.8.8:53"
        propagation:
          delayBeforeChecks: 60s
', 0, '0644', 0, 1),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'directory', './gateway/certs', '/etc/traefik/certs', 0, 0, NULL, 0, '', 0, 2),
    ('01M10RRA8F863EJ2N9TSRZBPC3', 'directory', './gateway/acme', '/letsencrypt', 0, 0, NULL, 0, '', 0, 3),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'file', '/var/run/docker.sock', '/var/run/docker.sock', 1, 1, NULL, 0, '', 0, 0),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'controlled_file', './traefik.yml', '/etc/traefik/traefik.yml', 0, 0, 'api:
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

certificatesResolvers:
  letsencrypt:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      httpChallenge:
        entryPoint: web

  letsencrypt-dns:
    acme:
      email: ""
      storage: /letsencrypt/acme.json
      dnsChallenge:
        provider: cloudflare
        resolvers:
          - "1.1.1.1:53"
          - "8.8.8.8:53"
        propagation:
          delayBeforeChecks: 60s
', 0, '0644', 0, 1),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'directory', './gateway/certs', '/etc/traefik/certs', 0, 0, NULL, 0, '', 0, 2),
    ('01M10RRA8F863EJ2N9TT9T3NK7', 'directory', './gateway/acme', '/letsencrypt', 0, 0, NULL, 0, '', 0, 3);

INSERT INTO `service` (`id`, `application_id`, `instance_key`, `code`, `version_id`, `status`) VALUES
    ('01M10RRA8F863EJ2N9TTG49G6S', '01M10RRA8F863EJ2N9TC3Z2EC1', 'default', 'traefik-default', '01M10RRA8F863EJ2N9TDN9JSBW', 'stopped');

INSERT INTO `service_component` (
    `id`, `service_id`, `source_version_component_id`, `component_name`, `entrypoint_json`, `command_json`, `pull_policy`, `restart_policy`, `status`
) VALUES (
    '01M10RRA8F863EJ2N9TYG2XPCP', '01M10RRA8F863EJ2N9TTG49G6S', '01M10RRA8F863EJ2N9TN8CWTEY', 'traefik', NULL, NULL, NULL, NULL, 'active'
);

INSERT INTO `route` (
    `id`, `name`, `protocol`, `domain`, `path_prefix`, `target_url`, `listen_port`, `service_id`, `component_name`,
    `endpoint_protocol`, `endpoint_container_port`, `enabled`, `https_enabled`, `cert_pem`, `cert_key`, `cert_type`, `acme_challenge`, `project_id`
) VALUES (
    '01M10RRA8F863EJ2N9TYPF0CV7', 'traefik', 'http', 'traefik-dashboard.lvh.me', '/', 'http://traefik-traefik:8080', NULL, NULL, NULL,
    NULL, NULL, 0, 0, NULL, NULL, 'manual', 'http', '01KRRKK0K3T519ZQZES3M4QA9Z'
);
