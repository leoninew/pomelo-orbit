-- BGE-M3 Text Embeddings Inference export from the current development SQLite database.
-- Includes the Application, Version, Component, Service, and ServiceExpose rows.

BEGIN TRANSACTION;
INSERT INTO project (id, name, code, created_at, updated_at, is_active) VALUES ('01KRRKK0K3T519ZQZES3M4QA9Z', '默认项目', 'default', '2024-03-16T00:00:00Z', '2024-03-16T00:00:00Z', 1);
INSERT INTO application (id, name, code, kind, image_pull_policy, created_at, updated_at, project_id) VALUES ('te7k4h4dpxyd2xcifknvpfgxna', 'BGE-M3 Embeddings', 'bge-m3', 'standard', 'missing', '2026-07-30T09:47:34.9097416Z', '2026-07-30T09:47:34.9097416Z', '01KRRKK0K3T519ZQZES3M4QA9Z');
INSERT INTO version (id, application_id, label, status, created_from_version_id, note, component_summary, created_at, updated_at) VALUES ('p3cfhibfdswoymtpkjbl5sa7s4', 'te7k4h4dpxyd2xcifknvpfgxna', 'bge-m3-cpu', 'unpublished', NULL, 'Text Embeddings Inference for BAAI/bge-m3', 'tei', '2026-07-30T09:47:34.9097416Z', '2026-07-30T09:48:27.8114157Z');
INSERT INTO version_component (id, version_id, name, image, command_json, pull_policy, restart_policy, created_at, updated_at) VALUES ('z4hyj6y63yronbuvt57baibhtq', 'p3cfhibfdswoymtpkjbl5sa7s4', 'tei', 'ghcr.io/huggingface/text-embeddings-inference:cpu-latest@sha256:f4fc40e4321fa174ef42d0dadfa183df1960879eb951efb2810886d26b47c3f4', '["--model-id","/data/bge-m3","--json-output"]', 'missing', 'unless-stopped', '2026-07-30T09:47:43.3078774Z', '2026-07-30T09:48:27.8109029Z');
INSERT INTO version_component_healthcheck (component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled) VALUES ('z4hyj6y63yronbuvt57baibhtq', 'CMD-SHELL', 'curl -fsS http://127.0.0.1:80/health', '30s', '5s', 10, '5m', NULL, 0);
INSERT INTO version_component_mount (component_id, source_type, source, target, read_only, source_is_host_path, content, content_masked, ignore_if_exists, position, mode) VALUES ('z4hyj6y63yronbuvt57baibhtq', 'directory', 'D:/SourceCodes/mywork/PomeloOrbit-go/data/deployment/tei-bge-m3/default/tei/cache', '/data', 0, 1, NULL, 0, 0, 0, '');
INSERT INTO service (id, application_id, instance_key, version_id, runtime_config_json, status, created_at, updated_at) VALUES ('qcbupb3mhirfx2rzecnsynyfce', 'te7k4h4dpxyd2xcifknvpfgxna', 'default', 'p3cfhibfdswoymtpkjbl5sa7s4', '{}', 'running', '2026-07-30T09:48:46.2005671Z', '2026-07-30T09:49:03.1301526Z');
INSERT INTO service_expose (id, service_id, component_name, protocol, container_port, path_prefix, access, listen_port, created_at, updated_at) VALUES ('izwnvcaqdo3gzj7zghofzugcva', 'qcbupb3mhirfx2rzecnsynyfce', 'tei', 'http', 80, NULL, 'local', 8081, '2026-07-30T09:48:46.2005671Z', '2026-07-30T09:48:46.2005671Z');

COMMIT;
