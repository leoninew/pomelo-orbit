---
name: deploy-sub2api-orbit
description: "Initialize, repair, preview, or deploy the three-component Sub2API stack through the pomelo-orbit-mcp control plane. Use when Sub2API needs PostgreSQL and Redis, persistent Orbit-managed storage, no direct host ports, and a custom HTTP Route to its main service."
---

# Deploy Sub2API Through Orbit

Use this skill only for an Orbit-managed Sub2API deployment. Read [the component contract](references/sub2api-components.md) before any write. Read [the Compose reference](references/docker-compose.yml) only to review the equivalent topology or after explicit authorization to use a Docker Compose fallback.

## Control Plane

- Use the `pomelo-orbit-mcp` MCP server for every lifecycle write. Inspect the live tool schema immediately before each write.
- Do not run `docker compose` as a normal deployment path. If MCP is unavailable, perform read-only diagnosis, explain the gap and ownership risk, and obtain explicit authorization before using the bundled Compose reference.
- Keep runtime configuration in memory. Do not print, persist, infer, rotate, or report secret values.
- Treat existing Applications, Services, Route records, mounts, and data as user-owned. Read and compare them first; do not overwrite or delete them without explicit authorization.

## Preflight

1. List the target Project, its standard Applications, Versions, Services, Routes, and Gateway metadata.
2. Run `runtime_doctor(network_name="traefik")`. Ensure one managed Gateway is configured and running before creating or enabling a Route. Use `orbit_provision_gateway` only when the Gateway or network is missing or unhealthy, then deploy that Gateway explicitly.
3. Collect the five runtime keys listed in the component contract without echoing their values. Each value must remain stable across redeployments.
4. Reuse an existing `sub2api` Application only if its selected Version, Service code, component names, persistence mounts, and runtime-key contract match exactly. Otherwise request a name, migration, or replacement decision.

## Initialize Or Repair

1. Create one `standard` Application with code `sub2api`, or read the compatible existing Application.
2. Use the initial unpublished Version returned by `orbit_create_application`. Add the `postgres`, `redis`, and `app` Components using `orbit_create_version_component` and the complete declarations in the component contract. Do not add `local`, `host`, or `gateway` endpoints.
3. Publish the Version, create or reuse exactly one stopped `default` Service, then use `orbit_update_service_env` to set the required runtime K/V collection.
4. Run `orbit_preview_service`. Confirm that the rendered Compose has no `ports:` section and that `app` has only the declared `http:8080` `internal` endpoint.

The `redis` command and shell health check must retain `$${REDIS_PASSWORD}`. The double dollar escapes Docker Compose interpolation so the container shell receives `$REDIS_PASSWORD`.

## Deploy And Expose

1. Call `orbit_deploy`, then `orbit_wait_deployment` for the Sub2API Service. Run `verify_deployment` after it reaches `ran_to_completion`.
2. Create the public HTTP Route only after the Service is deployed and verified. Use a managed target, never `target_url`:

   ```json
   {
     "project_id": "<project-id>",
     "name": "sub2api",
     "protocol": "http",
     "domain": "sub2api.example.com",
     "path_prefix": "/",
     "service_id": "<sub2api-service-id>",
     "component_name": "app",
     "endpoint_protocol": "http",
     "endpoint_container_port": 8080,
     "enabled": true
   }
   ```

3. Report the deployment and Route identifiers, public domain, health/verification result, and any non-secret warnings. Do not expose the raw runtime config, full rendered Compose, logs, or bearer credentials.

## Boundary

The standard Application joins the managed `traefik` network by default so the Gateway can reach the internal `app` Component. PostgreSQL and Redis receive no host port and no public Route; joining that Docker network is not a public exposure.
