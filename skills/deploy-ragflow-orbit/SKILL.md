---
name: deploy-ragflow-orbit
description: Deploy the bundled RAGFlow stack in Pomelo Orbit through pomelo_orbit MCP as one standard Application, one multi-component Version, and one Service. Use when deploying, rebuilding, validating, or diagnosing scripts/ragflow-bundled/docker-compose.yml, especially for MySQL, Valkey, MinIO, Elasticsearch, RAGFlow CPU, the traefik network, and local port 9380.
---

# Deploy RAGFlow With Orbit

Deploy the bundled RAGFlow Compose topology only through `pomelo_orbit` MCP lifecycle tools. Keep it as one `RAGFlow` Application containing five Components and one `default` Service; do not create one Service per Component.

Read [references/ragflow-bundled.md](references/ragflow-bundled.md) before creating or changing a Version.

## Scope And Safety

- Read the target Compose input and only the repository files required to complete the requested deployment work.
- Use Orbit tools for every lifecycle write. Do not run `docker compose up`, `down`, `restart`, or arbitrary container commands.
- Treat `runtime_config`, deployment options, rendered Compose, and write summaries as secret-bearing. Generate strong independent values, send them directly to Orbit, and never relay or persist their values.
- Never reset persistent data without confirming it belongs to a failed fresh deployment. `orbit_stop(remove_volumes=true)` does not remove logical directory bind mounts.
- Use `orbit_list_projects` and `orbit_list_applications` before creating resources. Reuse an existing application only after identifying its current resources and deployment state.

## Workflow

1. Read the target Compose and list available Orbit projects/applications.
2. Run `runtime_doctor(network_name="traefik")`. The rendered standard application needs the external `traefik` bridge network.
3. When the managed `traefik` Gateway is absent, its `default` Service is absent or unhealthy, or the `traefik` network is absent or unhealthy, call `orbit_provision_gateway` once before provisioning RAGFlow. It creates or uniquely reuses the managed Gateway, prepares its `default` Service, publishes its current Version, deploys it, waits for a successful terminal state, and confirms the network. Do not replace this with a hand-assembled sequence of low-level Gateway writes.
4. Create a standard Application with code `ragflow`, `image_pull_policy: missing`, then use the initial unpublished Version returned by `orbit_create_application`. When the Application already exists, list and reuse its single intended Version. Do not create a second Version; configure the five Components on that Version and create the local HTTP Expose on the Service.
5. Before creating the Service or deploying, check whether `mysql/data` or `es01/data` already contains data. MySQL retains its root host rule and password; Elasticsearch retains the initial `elastic` password. Obtain either explicit confirmation to delete the affected data directories for a clean reset, or the matching existing `MYSQL_PASSWORD` and `ELASTIC_PASSWORD` from the user. Do not generate replacements while retaining unknown bootstrap credentials.
6. Create exactly one Service with `instance_key: default`. Put the five random runtime values in `runtime_config`: `MYSQL_PASSWORD`, `REDIS_PASSWORD`, `MINIO_USER`, `MINIO_PASSWORD`, and `ELASTIC_PASSWORD`.
7. Deploy the saved Service with `orbit_deploy(service_id)` without publishing the Version, then wait for the Deployment terminal state.
8. Use `runtime_compose_ps` until every component is `running` and `healthy`. Its default response is a concise summary; use `detail=true` only for a diagnostic need. Run `runtime_http_probe` against `ragflow-cpu` port `80`, path `/`.
9. Run `verify_deployment`. Its default response includes conclusion, component health, constraints, and stability summary; use `detail=true` only for raw evidence. Report the conclusion, local URL, component health, and resource IDs without printing credentials.

## Version Rules

- Use the actual Component schema: `env` is `[{key,value}]`; mounts use `source_type`, `source`, `target`, and `read_only`; dependencies use `name` and `condition`; healthchecks separate `test_mode` and `test`.
- Use logical directory mounts from the reference, such as `mysql/data`. Do not send relative Compose paths like `./data/mysql` as mount sources.
- Do not send a `networks` field: the Component API no longer accepts it. Orbit injects the service default network and external `traefik` network with the expected `ragflow-<component>` aliases.
- Use `${NAME}` for required runtime placeholders and `${NAME:-default}` only where a default is intentional. Do not use Compose's `${NAME:?message}` form; the current Orbit parser does not support it.
- Create the UI endpoint as `{component_name: ragflow-cpu, protocol: http, container_port: 80, access: local, listen_port: 9380}`. Do not add Exposes for dependencies.
- Preserve `restart_policy: unless-stopped`, original commands, healthchecks, Elasticsearch resources, tmpfs, ulimit, and `service_healthy` dependencies.

## Persistent Bootstrap Credentials

RAGFlow uses `MYSQL_USER=root` from another container. Set `MYSQL_ROOT_HOST: "%"` on the MySQL Component at first initialization, in addition to `MYSQL_ROOT_PASSWORD` and `MYSQL_DATABASE`.

MySQL initializer variables, including `MYSQL_ROOT_HOST` and `MYSQL_ROOT_PASSWORD`, apply only to an empty `mysql/data` directory. `ELASTIC_PASSWORD` similarly only initializes Elasticsearch's `elastic` user when `es01/data` is empty. When either directory already exists, changing the Service runtime configuration does not update those credentials. A RAGFlow startup error such as `Access denied for user 'root'@'172.*'` can therefore indicate pre-existing MySQL data, not a malformed current runtime configuration.

Before deployment, check both `mysql/data` and `es01/data` and resolve their credentials before creating a Service. If either contains data, ask the user to either confirm deletion for a clean reset or provide the existing `MYSQL_PASSWORD` and `ELASTIC_PASSWORD`. Do not silently replace either configured password while retaining its directory.

If logs report `Host '172.*' is not allowed to connect to this MySQL server`, this setting was absent during initialization. It cannot repair an existing MySQL data directory. For a freshly created failed deployment only:

1. Stop the Service through Orbit and wait for completion.
2. With explicit confirmation, manually remove the affected `mysql/data` and/or `es01/data` directory to reset it. Do not add this recovery action to the Orbit Application API or MCP tool surface. For a full fresh reset, `orbit_delete_application(remove_dir=true)` remains available when the user confirms removal of every managed directory.
3. Redeploy the existing Application, Version, and Service with `MYSQL_ROOT_HOST: "%"`.

For any deployment that may contain user data, stop and request a backup/recovery decision instead of deleting it.

## Diagnostics

- If a Version Component create or update returns `Invalid tool input`, verify the Component schema and placeholder syntax before retrying.
- If `runtime_http_probe` fails while dependencies are starting, inspect `runtime_compose_ps` and scoped `runtime_compose_logs`, then wait for healthchecks before retrying.
- If a standard deployment lacks `traefik`, do not bypass Orbit with Docker CLI. Use `orbit_provision_gateway`; if it reports a conflict or unsuccessful deployment, inspect the returned resource IDs and use scoped Orbit diagnostics before retrying.
- If write or runtime MCP tools return generic errors, preserve the Deployment ID and use the scoped Orbit status/log tools. Do not guess or leak runtime configuration in the report.
