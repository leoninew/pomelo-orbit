---
name: deploy-ragflow-orbit
description: Deploy the bundled RAGFlow stack in Pomelo Orbit through pomelo_orbit MCP as one standard Application, one multi-component Version, and one Service. Use when deploying, rebuilding, validating, or diagnosing scripts/ragflow-bundled/docker-compose.yml, especially for MySQL, Valkey, MinIO, Elasticsearch, RAGFlow CPU, the traefik network, and local port 9380.
---

# Deploy RAGFlow With Orbit

Deploy the bundled RAGFlow Compose topology only through `pomelo_orbit` MCP lifecycle tools. Keep it as one `RAGFlow` Application containing five Components and one `default` Service; do not create one Service per Component.

Read [references/ragflow-bundled.md](references/ragflow-bundled.md) before creating or changing a Version.

## Scope And Safety

- Read only the requested Compose input unless the user authorizes broader repository inspection.
- Use Orbit tools for every lifecycle write. Do not run `docker compose up`, `down`, `restart`, or arbitrary container commands.
- Treat `runtime_config`, deployment options, rendered Compose, and write summaries as secret-bearing. Generate strong independent values, send them directly to Orbit, and never relay or persist their values.
- Never reset persistent data without confirming it belongs to a failed fresh deployment. `orbit_stop(remove_volumes=true)` does not remove logical directory bind mounts.
- Use `orbit_list_projects` and `orbit_list_applications` before creating resources. Reuse or update an existing application only when the user authorizes it.

## Workflow

1. Read the target Compose and list available Orbit projects/applications.
2. Run `runtime_doctor`. The rendered standard application needs the external `traefik` bridge network.
3. When `traefik` is absent, explain the blocker and obtain authorization before creating and deploying an Orbit Gateway. A Gateway is an additional managed resource, not an implicit side effect.
4. Create a standard Application with code `ragflow`, `image_pull_policy: missing`, then create one Version with the five Components and one local HTTP Expose.
5. Create exactly one Service with `instance_key: default`. Put the five random runtime values in `runtime_config`: `MYSQL_PASSWORD`, `REDIS_PASSWORD`, `MINIO_USER`, `MINIO_PASSWORD`, and `ELASTIC_PASSWORD`.
6. Publish the Version, deploy it, and wait for the Deployment terminal state.
7. Use `runtime_compose_ps` until every component is `running` and `healthy`. Run `runtime_http_probe` against `ragflow-cpu` port `80`, path `/`.
8. Run `verify_deployment`. Report the conclusion, local URL, component health, and resource IDs without printing credentials.

## Version Rules

- Use the actual Component schema: `env` is `[{key,value}]`; mounts use `source_type`, `source`, `target`, and `read_only`; dependencies use `name` and `condition`; healthchecks separate `test_mode` and `test`.
- Use logical directory mounts from the reference, such as `mysql/data`. Do not send relative Compose paths like `./data/mysql` as mount sources.
- Leave `networks` empty. Orbit injects the service default network and external `traefik` network with the expected `ragflow-<component>` aliases.
- Use `${NAME}` for required runtime placeholders and `${NAME:-default}` only where a default is intentional. Do not use Compose's `${NAME:?message}` form; the current Orbit parser does not support it.
- Create the UI endpoint as `{component_name: ragflow-cpu, protocol: http, container_port: 80, access: local, listen_port: 9380}`. Do not add Exposes for dependencies.
- Preserve `restart_policy: unless-stopped`, original commands, healthchecks, Elasticsearch resources, tmpfs, ulimit, and `service_healthy` dependencies.

## MySQL First Initialization

RAGFlow uses `MYSQL_USER=root` from another container. Set `MYSQL_ROOT_HOST: "%"` on the MySQL Component at first initialization, in addition to `MYSQL_ROOT_PASSWORD` and `MYSQL_DATABASE`.

If logs report `Host '172.*' is not allowed to connect to this MySQL server`, this setting was absent during initialization. It cannot repair an existing MySQL data directory. For a freshly created failed deployment only:

1. Stop the Service through Orbit and wait for completion.
2. Delete the Application with `remove_dir: true` to remove the failed managed directory.
3. Recreate the Application, Version, Service, credentials, and Deployment with `MYSQL_ROOT_HOST: "%"`.

For any deployment that may contain user data, stop and request a backup/recovery decision instead of deleting it.

## Diagnostics

- If `orbit_create_version` returns `Invalid tool input`, verify the Component schema and placeholder syntax before retrying.
- If `runtime_http_probe` fails while dependencies are starting, inspect `runtime_compose_ps` and scoped `runtime_compose_logs`, then wait for healthchecks before retrying.
- If a standard deployment lacks `traefik`, do not bypass Orbit with Docker CLI. Create or repair the managed Gateway only with explicit authorization.
- If write or runtime MCP tools return generic errors, preserve the Deployment ID and use the scoped Orbit status/log tools. Do not guess or leak runtime configuration in the report.
