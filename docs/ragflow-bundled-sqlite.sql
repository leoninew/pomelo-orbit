-- RAGFlow bundled topology export from the development SQLite database.
--
-- This file contains non-sensitive Application and Version specification data.
-- It intentionally excludes service.runtime_config_json, service rows, and
-- deployment history because runtime configuration contains credentials.
-- Create the default Service separately with MYSQL_PASSWORD, REDIS_PASSWORD,
-- MINIO_USER, MINIO_PASSWORD, and ELASTIC_PASSWORD before deploying.

BEGIN TRANSACTION;

INSERT INTO project (id, name, code, created_at, updated_at, is_active) VALUES ('01KRRKK0K3T519ZQZES3M4QA9Z', '默认项目', 'default', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z', 1);

INSERT INTO application (id, name, code, kind, image_pull_policy, created_at, updated_at, project_id) VALUES ('mj3ihzsegd3xjwr72el5cc2cii', 'RAGFlow', 'ragflow', 'standard', 'missing', '2026-07-29 13:13:52.3369901 +0000 UTC', '2026-07-29 13:13:52.3369901 +0000 UTC', '01KRRKK0K3T519ZQZES3M4QA9Z');

INSERT INTO version (id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at) VALUES ('tnp4hkpwtjrj3hpzfpfpvbi53i', 'mj3ihzsegd3xjwr72el5cc2cii', 'ragflow-bundled', 'unpublished', NULL, 'Bundled RAGFlow: mysql, redis, minio, es01, ragflow-cpu', 'es01, minio, mysql, ragflow-cpu, redis', '2026-07-29 13:14:56.5083991 +0000 UTC', '2026-07-29 13:41:39.6134848 +0000 UTC');

INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'es01', 'elasticsearch:8.11.3', '[]', 'missing', 'unless-stopped', '2026-07-29 13:14:56.5090193 +0000 UTC', '2026-07-29 13:14:56.5090193 +0000 UTC');
INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('qnboyu23huijysnhstbwlolu2i', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'minio', 'pgsty/minio:RELEASE.2026-03-25T00-00-00Z', '["server","--console-address",":9001","/data"]', 'missing', 'unless-stopped', '2026-07-29 13:14:56.5090193 +0000 UTC', '2026-07-29 13:14:56.5090193 +0000 UTC');
INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'mysql', 'mysql:8.0.39', '["--max_connections=1000","--character-set-server=utf8mb4","--collation-server=utf8mb4_unicode_ci","--default-authentication-plugin=mysql_native_password","--tls_version=TLSv1.2,TLSv1.3","--binlog_expire_logs_seconds=604800"]', 'missing', 'unless-stopped', '2026-07-29 13:14:56.5090193 +0000 UTC', '2026-07-29 13:14:56.5090193 +0000 UTC');
INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'ragflow-cpu', 'infiniflow/ragflow:v0.26.4', '["--enable-adminserver","--init-model-provider-tables"]', 'missing', 'unless-stopped', '2026-07-29 13:14:56.5090193 +0000 UTC', '2026-07-29 13:14:56.5090193 +0000 UTC');
INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('c4elns6ryx5cuwer6x47gqth3m', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'redis', 'valkey/valkey:8', '["sh","-c","exec redis-server --requirepass \\"$${REDIS_PASSWORD}\\" --maxmemory 128mb --maxmemory-policy allkeys-lru"]', 'missing', 'unless-stopped', '2026-07-29 13:14:56.5090193 +0000 UTC', '2026-07-29 13:14:56.5090193 +0000 UTC');

INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'ELASTIC_PASSWORD', '${ELASTIC_PASSWORD}', 0);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'node.name', 'es01', 1);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'bootstrap.memory_lock', 'false', 2);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'discovery.type', 'single-node', 3);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'xpack.security.enabled', 'true', 4);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'xpack.security.http.ssl.enabled', 'false', 5);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'xpack.security.transport.ssl.enabled', 'false', 6);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'cluster.routing.allocation.disk.watermark.low', '5gb', 7);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'cluster.routing.allocation.disk.watermark.high', '3gb', 8);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'cluster.routing.allocation.disk.watermark.flood_stage', '2gb', 9);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('qnboyu23huijysnhstbwlolu2i', 'MINIO_ROOT_USER', '${MINIO_USER}', 0);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('qnboyu23huijysnhstbwlolu2i', 'MINIO_ROOT_PASSWORD', '${MINIO_PASSWORD}', 1);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'MYSQL_DATABASE', 'rag_flow', 0);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'MYSQL_ROOT_PASSWORD', '${MYSQL_PASSWORD}', 1);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'MYSQL_ROOT_HOST', '%', 2);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'DOC_ENGINE', 'elasticsearch', 0);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'DEVICE', 'cpu', 1);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'API_PROXY_SCHEME', 'python', 2);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'TZ', 'Asia/Shanghai', 3);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_HOST', 'mysql', 4);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_PORT', '3306', 5);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_DBNAME', 'rag_flow', 6);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_USER', 'root', 7);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_PASSWORD', '${MYSQL_PASSWORD}', 8);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MYSQL_MAX_PACKET', '1073741824', 9);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'REDIS_HOST', 'redis', 10);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'REDIS_PASSWORD', '${REDIS_PASSWORD}', 11);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MINIO_HOST', 'minio', 12);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MINIO_USER', '${MINIO_USER}', 13);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'MINIO_PASSWORD', '${MINIO_PASSWORD}', 14);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'ES_HOST', 'es01', 15);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'ES_USER', 'elastic', 16);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'ELASTIC_PASSWORD', '${ELASTIC_PASSWORD}', 17);
INSERT INTO version_component_env (component_id, env_key, value, position) VALUES ('c4elns6ryx5cuwer6x47gqth3m', 'REDIS_PASSWORD', '${REDIS_PASSWORD}', 0);

INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'CMD-SHELL', 'curl -fsS -u "elastic:$${ELASTIC_PASSWORD}" http://localhost:9200', '10s', '10s', 120, NULL, NULL, 0);
INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('qnboyu23huijysnhstbwlolu2i', 'CMD', 'curl -fsS http://localhost:9000/minio/health/live', '10s', '10s', 120, NULL, NULL, 0);
INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'CMD-SHELL', 'mysqladmin ping -uroot -p"$${MYSQL_ROOT_PASSWORD}"', '10s', '10s', 120, NULL, NULL, 0);
INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'CMD', 'curl -fsS http://localhost:80/', '10s', '10s', 120, NULL, NULL, 0);
INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('c4elns6ryx5cuwer6x47gqth3m', 'CMD-SHELL', 'redis-cli -a "$${REDIS_PASSWORD}" ping', '10s', '10s', 120, NULL, NULL, 0);

INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'directory', 'es01/data', '/usr/share/elasticsearch/data', 0, 0, NULL, 0, 0, 0, '');
INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('qnboyu23huijysnhstbwlolu2i', 'directory', 'minio/data', '/data', 0, 0, NULL, 0, 0, 0, '');
INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('2nz6qdza3neie5bjp32ofs2g24', 'directory', 'mysql/data', '/var/lib/mysql', 0, 0, NULL, 0, 0, 0, '');
INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'directory', 'ragflow-cpu/logs', '/ragflow/logs', 0, 0, NULL, 0, 0, 0, '');
INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('c4elns6ryx5cuwer6x47gqth3m', 'directory', 'redis/data', '/data', 0, 0, NULL, 0, 0, 0, '');

INSERT INTO version_component_dependency (component_id, depends_on_name, condition, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'mysql', 'service_healthy', 0);
INSERT INTO version_component_dependency (component_id, depends_on_name, condition, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'redis', 'service_healthy', 1);
INSERT INTO version_component_dependency (component_id, depends_on_name, condition, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'minio', 'service_healthy', 2);
INSERT INTO version_component_dependency (component_id, depends_on_name, condition, position) VALUES ('jfx6pwb6eyrofzynrdoorcdrk4', 'es01', 'service_healthy', 3);

INSERT INTO version_component_resource (component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', NULL, '8g', NULL, NULL);
INSERT INTO version_component_tmpfs (component_id, target, size_bytes, mode, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', '/tmp', 536870912, '1777', 0);
INSERT INTO version_component_ulimit (component_id, name, soft, hard, position) VALUES ('2wgo3a2rjpxwcki7pdxtivx3km', 'memlock', -1, -1, 0);

INSERT INTO version_expose (id, version_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at) VALUES ('igftqbmeal7axrmpuqeu7xv4dy', 'tnp4hkpwtjrj3hpzfpfpvbi53i', 'ragflow-cpu', 'http', 80, NULL, 'local', 9380, '2026-07-29 13:14:56.5148104 +0000 UTC', '2026-07-29 13:14:56.5148104 +0000 UTC');

COMMIT;
