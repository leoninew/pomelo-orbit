-- Pomelo Orbit topology export for an already-migrated, empty SQLite database.
-- Included tables:
--   project
--   application
--   gateway_config
--   version
--   version_component
--   version_component_dependency
--   version_component_env
--   version_component_healthcheck
--   version_component_mount
--   version_component_port
--   version_component_resource
--   version_component_tmpfs
--   version_component_ulimit
--   service
--   service_expose
--
PRAGMA foreign_keys = OFF;
BEGIN;

-- project: 1 row(s)
INSERT INTO project (code, created_at, id, is_active, name, updated_at) VALUES ('default', '2024-03-16T00:00:00Z', '01KRRKK0K3T519ZQZES3M4QA9Z', 1, '默认项目', '2024-03-16T00:00:00Z') ON CONFLICT DO NOTHING;

-- application: 3 row(s)
INSERT INTO application (code, created_at, id, image_pull_policy, kind, name, project_id, updated_at) VALUES ('ragflow', '2026-07-29T05:43:13.6131134Z', 'd5bukyuaictz6n3u7ewqbrh32e', 'missing', 'standard', 'RAGFlow', '01KRRKK0K3T519ZQZES3M4QA9Z', '2026-07-29T05:43:13.6131134Z') ON CONFLICT DO NOTHING;
INSERT INTO application (code, created_at, id, image_pull_policy, kind, name, project_id, updated_at) VALUES ('traefik', '2026-07-30T06:32:13.7447493Z', 'n4gn47povet4x7vqrdq4nbiyby', 'missing', 'gateway', 'Traefik', '01KRRKK0K3T519ZQZES3M4QA9Z', '2026-07-30T06:32:13.7447493Z') ON CONFLICT DO NOTHING;
INSERT INTO application (code, created_at, id, image_pull_policy, kind, name, project_id, updated_at) VALUES ('bge-m3', '2026-07-30T09:47:34.9097416Z', 'te7k4h4dpxyd2xcifknvpfgxna', 'missing', 'standard', 'BGE-M3 Embeddings', '01KRRKK0K3T519ZQZES3M4QA9Z', '2026-07-30T09:47:34.9097416Z') ON CONFLICT DO NOTHING;

-- gateway_config: 1 row(s)
INSERT INTO gateway_config (application_id, base_domain, created_at, default_entrypoint, image, rest_api_url, tls_mode, updated_at) VALUES ('n4gn47povet4x7vqrdq4nbiyby', 'lvh.me', '2026-07-30T06:32:13.7447493Z', 'web', 'traefik:3.6', 'http://localhost:8080', 'none', '2026-07-30T06:32:13.7447493Z') ON CONFLICT DO NOTHING;

-- version: 4 row(s)
INSERT INTO version (application_id, component_summary, created_at, created_from_version_id, id, label, note, status, updated_at) VALUES ('te7k4h4dpxyd2xcifknvpfgxna', 'huggingface-tei', '2026-07-30T14:28:15.3558494Z', NULL, '7d63zel7q2kfm2vs6pqhqonltq', 'cpu-latest @ huaweicloud', 'swr.cn-north-4.myhuaweicloud.com/ddn-k8s/ghcr.io/huggingface/text-embeddings-inference:cpu-latest', 'unpublished', '2026-07-30T14:29:18.5484235Z') ON CONFLICT DO NOTHING;
INSERT INTO version (application_id, component_summary, created_at, created_from_version_id, id, label, note, status, updated_at) VALUES ('n4gn47povet4x7vqrdq4nbiyby', 'traefik', '2026-07-30T06:32:13.7447493Z', NULL, 'ihv3kbyqegx4m4yojwpp2fqxum', 'managed-2imm5kkem7ec2krjorewk2tgk4', NULL, 'published', '2026-07-30T06:32:14.0756321Z') ON CONFLICT DO NOTHING;
INSERT INTO version (application_id, component_summary, created_at, created_from_version_id, id, label, note, status, updated_at) VALUES ('d5bukyuaictz6n3u7ewqbrh32e', 'es01, minio, mysql, ragflow-cpu, redis', '2026-07-29T05:44:24.4516542Z', NULL, 'krq3ndmgte4mqx65uaidmeiokm', 'ragflow-v0.26.4', NULL, 'unpublished', '2026-07-29T08:32:46.1699854Z') ON CONFLICT DO NOTHING;
INSERT INTO version (application_id, component_summary, created_at, created_from_version_id, id, label, note, status, updated_at) VALUES ('te7k4h4dpxyd2xcifknvpfgxna', 'huggingface-tei', '2026-07-30T09:47:34.9097416Z', NULL, 'p3cfhibfdswoymtpkjbl5sa7s4', 'cpu-latest', '', 'unpublished', '2026-07-30T14:40:20.4796087Z') ON CONFLICT DO NOTHING;

-- version_component: 8 row(s)
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('[]', '2026-07-30T06:32:13.7447493Z', '5tsyye6bkh2vzt5iqx4mrtxzeq', 'traefik:3.6', 'traefik', NULL, NULL, '2026-07-30T06:32:13.7447493Z', 'ihv3kbyqegx4m4yojwpp2fqxum') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('["--max_connections=1000","--character-set-server=utf8mb4","--collation-server=utf8mb4_unicode_ci","--default-authentication-plugin=mysql_native_password","--tls_version=TLSv1.2,TLSv1.3","--binlog_expire_logs_seconds=604800"]', '2026-07-29T05:44:24.4522011Z', 'bvolgbvcokhtyqgsge4g2dfvva', 'mysql:8.0.39', 'mysql', NULL, 'unless-stopped', '2026-07-29T05:44:24.4522011Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('["sh","-c","exec redis-server --requirepass \"$${REDIS_PASSWORD}\" --maxmemory 128mb --maxmemory-policy allkeys-lru"]', '2026-07-29T05:44:24.4522011Z', 'cxtiz4ny3i6e2t7s65fhl4g7rm', 'valkey/valkey:8', 'redis', NULL, 'unless-stopped', '2026-07-29T05:44:24.4522011Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('[]', '2026-07-29T05:44:24.4522011Z', 'dvwrntepg5g5zktwctdiy76etm', 'elasticsearch:8.11.3', 'es01', NULL, 'unless-stopped', '2026-07-29T08:32:46.1694806Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('[]', '2026-07-30T14:29:18.5484235Z', 'gfroias7cfn5tc6fdivup3p4hq', 'swr.cn-north-4.myhuaweicloud.com/ddn-k8s/ghcr.io/huggingface/text-embeddings-inference:cpu-latest', 'huggingface-tei', 'missing', NULL, '2026-07-30T14:29:18.5484235Z', '7d63zel7q2kfm2vs6pqhqonltq') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('["--enable-adminserver","--init-model-provider-tables"]', '2026-07-29T05:44:24.4522011Z', 'ktgbpgnu2xgo3xhpujfwkzlkc4', 'infiniflow/ragflow:v0.26.4', 'ragflow-cpu', NULL, 'unless-stopped', '2026-07-29T06:01:56.1077245Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('["server","--console-address",":9001","/data"]', '2026-07-29T05:44:24.4522011Z', 'mhkgwj47tmbyq5rfokrekeupju', 'pgsty/minio:RELEASE.2026-03-25T00-00-00Z', 'minio', NULL, 'unless-stopped', '2026-07-29T05:44:24.4522011Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;
INSERT INTO version_component (command_json, created_at, id, image, name, pull_policy, restart_policy, updated_at, version_id) VALUES ('["--model-id","/data/bge-m3","--json-output"]', '2026-07-30T09:47:43.3078774Z', 'z4hyj6y63yronbuvt57baibhtq', 'ghcr.io/huggingface/text-embeddings-inference:cpu-latest', 'huggingface-tei', 'missing', 'unless-stopped', '2026-07-30T14:40:20.4790965Z', 'p3cfhibfdswoymtpkjbl5sa7s4') ON CONFLICT DO NOTHING;

-- version_component_dependency: 4 row(s)
INSERT INTO version_component_dependency (component_id, condition, depends_on_name, position) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'service_healthy', 'mysql', 0) ON CONFLICT DO NOTHING;
INSERT INTO version_component_dependency (component_id, condition, depends_on_name, position) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'service_healthy', 'redis', 1) ON CONFLICT DO NOTHING;
INSERT INTO version_component_dependency (component_id, condition, depends_on_name, position) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'service_healthy', 'minio', 2) ON CONFLICT DO NOTHING;
INSERT INTO version_component_dependency (component_id, condition, depends_on_name, position) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'service_healthy', 'es01', 3) ON CONFLICT DO NOTHING;

-- version_component_env: 34 row(s)
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('bvolgbvcokhtyqgsge4g2dfvva', 'MYSQL_DATABASE', 0, 'rag_flow') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('bvolgbvcokhtyqgsge4g2dfvva', 'MYSQL_ROOT_PASSWORD', 1, '${MYSQL_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('bvolgbvcokhtyqgsge4g2dfvva', 'MYSQL_ROOT_HOST', 2, '%') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('cxtiz4ny3i6e2t7s65fhl4g7rm', 'REDIS_PASSWORD', 0, '${REDIS_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'ELASTIC_PASSWORD', 0, '${ELASTIC_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'node.name', 1, 'es01') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'bootstrap.memory_lock', 2, 'false') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'discovery.type', 3, 'single-node') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'xpack.security.enabled', 4, 'true') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'xpack.security.http.ssl.enabled', 5, 'false') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'xpack.security.transport.ssl.enabled', 6, 'false') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'cluster.routing.allocation.disk.watermark.low', 7, '5gb') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'cluster.routing.allocation.disk.watermark.high', 8, '3gb') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('dvwrntepg5g5zktwctdiy76etm', 'cluster.routing.allocation.disk.watermark.flood_stage', 9, '2gb') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'DOC_ENGINE', 0, 'elasticsearch') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'DEVICE', 1, 'cpu') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'API_PROXY_SCHEME', 2, 'python') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'TZ', 3, 'Asia/Shanghai') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_HOST', 4, 'mysql') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_PORT', 5, '3306') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_DBNAME', 6, 'rag_flow') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_USER', 7, 'root') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_PASSWORD', 8, '${MYSQL_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MYSQL_MAX_PACKET', 9, '1073741824') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'REDIS_HOST', 10, 'redis') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'REDIS_PASSWORD', 11, '${REDIS_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MINIO_HOST', 12, 'minio') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MINIO_USER', 13, '${MINIO_USER}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'MINIO_PASSWORD', 14, '${MINIO_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'ES_HOST', 15, 'es01') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'ES_USER', 16, 'elastic') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 'ELASTIC_PASSWORD', 17, '${ELASTIC_PASSWORD}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('mhkgwj47tmbyq5rfokrekeupju', 'MINIO_ROOT_USER', 0, '${MINIO_USER}') ON CONFLICT DO NOTHING;
INSERT INTO version_component_env (component_id, env_key, position, value) VALUES ('mhkgwj47tmbyq5rfokrekeupju', 'MINIO_ROOT_PASSWORD', 1, '${MINIO_PASSWORD}') ON CONFLICT DO NOTHING;

-- version_component_healthcheck: 6 row(s)
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('bvolgbvcokhtyqgsge4g2dfvva', 0, '10s', 120, NULL, NULL, 'mysqladmin ping -uroot -p"$${MYSQL_ROOT_PASSWORD}"', 'CMD-SHELL', '10s') ON CONFLICT DO NOTHING;
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('cxtiz4ny3i6e2t7s65fhl4g7rm', 0, '10s', 120, NULL, NULL, 'redis-cli -a "$${REDIS_PASSWORD}" ping', 'CMD-SHELL', '10s') ON CONFLICT DO NOTHING;
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('dvwrntepg5g5zktwctdiy76etm', 0, '10s', 120, NULL, NULL, 'curl -fsS -u "elastic:$${ELASTIC_PASSWORD}" http://localhost:9200', 'CMD-SHELL', '10s') ON CONFLICT DO NOTHING;
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', 0, '10s', 120, NULL, NULL, 'curl -fsS http://localhost:80/', 'CMD', '10s') ON CONFLICT DO NOTHING;
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('mhkgwj47tmbyq5rfokrekeupju', 0, '10s', 120, NULL, NULL, 'curl -fsS http://localhost:9000/minio/health/live', 'CMD', '10s') ON CONFLICT DO NOTHING;
INSERT INTO version_component_healthcheck (component_id, disabled, interval, retries, start_interval, start_period, test, test_mode, timeout) VALUES ('z4hyj6y63yronbuvt57baibhtq', 0, '30s', 10, NULL, '5m', 'curl -fsS http://127.0.0.1:80/health', 'CMD-SHELL', '5s') ON CONFLICT DO NOTHING;

-- version_component_mount: 9 row(s)
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', NULL, 0, 0, '', 0, 1, '/var/run/docker.sock', 1, 'file', '/var/run/docker.sock') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', 'api:
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
', 0, 0, '0644', 1, 0, 'traefik.yml', 0, 'controlled_file', '/etc/traefik/traefik.yml') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', '{}', 0, 1, '0600', 2, 0, 'acme.json', 0, 'controlled_file', '/letsencrypt/acme.json') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('bvolgbvcokhtyqgsge4g2dfvva', NULL, 0, 0, '', 0, 0, 'mysql/data', 0, 'directory', '/var/lib/mysql') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('cxtiz4ny3i6e2t7s65fhl4g7rm', NULL, 0, 0, '', 0, 0, 'redis/data', 0, 'directory', '/data') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('dvwrntepg5g5zktwctdiy76etm', NULL, 0, 0, '', 0, 0, 'es01/data', 0, 'directory', '/usr/share/elasticsearch/data') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('ktgbpgnu2xgo3xhpujfwkzlkc4', NULL, 0, 0, '', 0, 0, 'ragflow-cpu/logs', 0, 'directory', '/ragflow/logs') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('mhkgwj47tmbyq5rfokrekeupju', NULL, 0, 0, '', 0, 0, 'minio/data', 0, 'directory', '/data') ON CONFLICT DO NOTHING;
INSERT INTO version_component_mount (component_id, content, content_masked, ignore_if_exists, mode, position, read_only, source, source_is_host_path, source_type, target) VALUES ('z4hyj6y63yronbuvt57baibhtq', NULL, 0, 0, '', 0, 0, 'D:/SourceCodes/mywork/PomeloOrbit-go/data/deployment/tei-bge-m3/default/tei/cache', 1, 'directory', '/data') ON CONFLICT DO NOTHING;

-- version_component_port: 3 row(s)
INSERT INTO version_component_port (component_id, container_port, host_port, position) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', 80, 80, 0) ON CONFLICT DO NOTHING;
INSERT INTO version_component_port (component_id, container_port, host_port, position) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', 443, 443, 1) ON CONFLICT DO NOTHING;
INSERT INTO version_component_port (component_id, container_port, host_port, position) VALUES ('5tsyye6bkh2vzt5iqx4mrtxzeq', 8080, 8080, 2) ON CONFLICT DO NOTHING;

-- version_component_resource: 1 row(s)
INSERT INTO version_component_resource (component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory) VALUES ('dvwrntepg5g5zktwctdiy76etm', NULL, '8g', NULL, NULL) ON CONFLICT DO NOTHING;

-- version_component_tmpfs: 1 row(s)
INSERT INTO version_component_tmpfs (component_id, mode, position, size_bytes, target) VALUES ('dvwrntepg5g5zktwctdiy76etm', '1777', 0, 536870912, '/tmp') ON CONFLICT DO NOTHING;

-- version_component_ulimit: 1 row(s)
INSERT INTO version_component_ulimit (component_id, hard, name, position, soft) VALUES ('dvwrntepg5g5zktwctdiy76etm', -1, 'memlock', 0, -1) ON CONFLICT DO NOTHING;

-- service: 3 row(s)
INSERT INTO service (application_id, created_at, id, instance_key, runtime_config_json, status, updated_at, version_id) VALUES ('n4gn47povet4x7vqrdq4nbiyby', '2026-07-30T06:32:14.1278022Z', 'mtcadekhtypgjmvrashrc3emym', 'default', '{}', 'running', '2026-07-30T14:01:48.6712122Z', 'ihv3kbyqegx4m4yojwpp2fqxum') ON CONFLICT DO NOTHING;
INSERT INTO service (application_id, created_at, id, instance_key, runtime_config_json, status, updated_at, version_id) VALUES ('te7k4h4dpxyd2xcifknvpfgxna', '2026-07-30T09:48:46.2005671Z', 'qcbupb3mhirfx2rzecnsynyfce', 'default', '{}', 'running', '2026-07-30T14:43:23.582005Z', '7d63zel7q2kfm2vs6pqhqonltq') ON CONFLICT DO NOTHING;
INSERT INTO service (application_id, created_at, id, instance_key, runtime_config_json, status, updated_at, version_id) VALUES ('d5bukyuaictz6n3u7ewqbrh32e', '2026-07-30T06:36:41.9913994Z', 'ys2i2wy27i6c6oee3pxvggkfgu', 'default', '{"ELASTIC_PASSWORD":"ELASTIC_PASSWORD","MINIO_PASSWORD":"MINIO_PASSWORD","MINIO_USER":"MINIO_USER","MYSQL_PASSWORD":"MYSQL_PASSWORD","REDIS_PASSWORD":"REDIS_PASSWORD"}', 'running', '2026-07-30T14:44:26.5943839Z', 'krq3ndmgte4mqx65uaidmeiokm') ON CONFLICT DO NOTHING;

-- service_expose: 2 row(s)
INSERT INTO service_expose (access, component_name, container_port, created_at, id, listen_port, path_prefix, protocol, service_id, updated_at) VALUES ('public', 'ragflow-cpu', 80, '2026-07-30T14:21:25.0275531Z', 'c4ttjpoawcgvifbfgjo733ae4q', NULL, NULL, 'http', 'ys2i2wy27i6c6oee3pxvggkfgu', '2026-07-30T14:21:25.0275531Z') ON CONFLICT DO NOTHING;
INSERT INTO service_expose (access, component_name, container_port, created_at, id, listen_port, path_prefix, protocol, service_id, updated_at) VALUES ('public', 'huggingface-tei', 8080, '2026-07-30T14:42:40.0293404Z', 'lcc6xl4ubwexj3rzh3aztj4qjq', NULL, NULL, 'http', 'qcbupb3mhirfx2rzecnsynyfce', '2026-07-30T14:42:40.0293404Z') ON CONFLICT DO NOTHING;

COMMIT;
PRAGMA foreign_keys = ON;
