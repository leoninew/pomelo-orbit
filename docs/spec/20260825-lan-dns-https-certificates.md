# 局域网 DNS-01 HTTPS 与 Gateway 配置收敛规格
最后修改时间: 2026-08-29 15:14:37

Review status: Accepted

流程模式: 严格 / strict

## Requirement Basis

依据 [Requirement](../requirement/20260825-lan-dns-https-certificates.md)：Application、Version、Component 与 Service 是 Traefik Compose 拓扑唯一来源。GatewayConfig 是业务配置和部署选择器；它不持有也不投影 listener、mount、endpoint 或 resolver 结构。

## Overview

```text
GatewayConfig
  ├── base gateway settings
  ├── optional ACME profile + email + DNS token
  └── profile -> Version binding
              |
              | before Deployment creation
              v
Gateway Service selects ordinary Version
              |
              | snapshot GatewayConfig
              v
Version/Component plan + narrow Gateway deployment enrichment
  ├── set declared resolver email
  └── DNS profile: CF_DNS_API_TOKEN environment
              v
docker compose
```

## Version Profiles

Gateway creation creates one base Version and three profile Version records. Every Version has the same `traefik` Component name, image, socket mount, cert/acme directory mounts, static HTTP/HTTPS/API endpoints and initial static config. They are ordinary editable Version records.

| Version role | Resolver content | Gateway requirements |
| --- | --- | --- |
| base | none | none; supports HTTP/HTTPS and manual certificates |
| `http` | `letsencrypt` + HTTP challenge | ACME email |
| `dns` | `letsencrypt-dns` + Cloudflare DNS challenge | ACME email and DNS token |
| `http-dns` | both resolver names | ACME email and DNS token |

`gateway_acme_profile_version` binds a Gateway Application to its base and three profile Version IDs. `GatewayConfig.acme_profile` is empty for the base Version or one of `http` / `dns` / `http-dns`. The deployment command resolves the binding and updates the default Gateway Service to that Version before creating its Deployment. The worker must execute the Deployment's persisted `version_id`, never resolve a mutable profile binding again.

TCP entrypoints and their host endpoints remain Version declarations. Administrators add/fork them through existing Component APIs, deploy the selected Version, and then create TCP Routes. GatewayConfig has no listener model or table.

## Gateway Configuration

GatewayConfig contains:

- `traefik_component_name`, `rest_api_url`, `rest_ready_timeout_seconds`, `base_domain`.
- `default_entrypoint` and `tls_mode`, used only as the active Gateway context while rendering ordinary Component Docker labels.
- `acme_profile`, `acme_email` and `dns_api_token`.

`acme_email` is required only for a non-empty profile. `dns_api_token` is required only for `dns` and `http-dns`. Both are normal Gateway fields returned by Gateway reads and copied into the Gateway deployment snapshot. `POMELO_ORBIT_CF_DNS_API_TOKEN`, `cert.letsencrypt.*`, Cloudflare Settings accessors and availability projections are removed.

## Deployment Enrichment

The worker builds the normal effective plan exclusively from the stored Version/Service declarations. For a Gateway deployment it then performs these checks and transformations on the plan copy:

1. Confirm the selected Version is the base Version or the snapshot's mapped profile Version and contains `traefik_component_name`.
2. Locate the declared controlled file at `/etc/traefik/traefik.yml`. For a profile Version, parse it as YAML and set the existing resolver `acme.email` scalar from the snapshot; do not add resolver blocks or change mounts/endpoints.
3. For a DNS profile, append `CF_DNS_API_TOKEN=<snapshot dns_api_token>` to the bound Traefik Component's effective environment. No secret file, Compose secret or `_FILE` variable is used.

The enrichment fails before Compose if the expected resolver structure is absent, the profile/email/token requirements fail, or the binding Component is absent. It does not write Version, Service overlay or GatewayConfig. The rendered Compose is the only output containing email/token values.

## Route Contract

Route challenge options derive from the active GatewayConfig profile:

- `http` is available for `http` and `http-dns`.
- `dns` is available for `dns` and `http-dns` only when `dns_api_token` is non-empty.

REST snapshots continue using `letsencrypt` and `letsencrypt-dns`. Manual and mkcert routes need no profile. Route save/enable neither changes GatewayConfig nor switches a Gateway Version.

## Scope Boundary

本规格只约束 Gateway profile 对 Route challenge capability、resolver 和 TCP Version endpoint 的影响。Route 业务数据与 Traefik REST provider 的全量同步、前端启停草稿及预览/确认交互属于独立轻量任务，见 [Route REST 快照防误删](../requirement/20260819-route-rest-snapshot-safety.md) 及其验收记录；它不改变本规格的 Gateway 拓扑和 profile 设计。

## Interfaces And Cutover

1. Update the uncommitted SQLite/MySQL `000036` migration in place for `acme_profile`, `dns_api_token` and `gateway_acme_profile_version`.
2. Update schema, query, sqlc, model, repository, DTO, protobuf, HTTP mapper and web types. Gateway API carries profile/email/token and profile Version bindings, not listeners or credential availability.
3. Gateway create creates the base/profile Version set and their mappings. Gateway update validates profile-specific values but never alters Version declarations.
4. Remove global DNS token configuration, secret workspace, runner redaction and their tests/wiring. The gateway renderer directly injects the snapshot token into the DNS Component environment.
5. Replace broad `projectGatewayPlan` with the narrow enrichment above. Restore Gateway Component construction with the complete static Version declaration.

The local development migration is edited in place. Recreated Gateways receive the base/profile Version set; no recovery rewrite, compatibility conversion or parallel model is retained.

## Risks

- GatewayConfig now stores a DNS token plainly and deployment Compose exposes it; this is accepted product behavior for this change.
- Profile Version topology can drift through ordinary Version editing. Deployment and TCP Route validation must identify missing required Component/file/endpoint declarations rather than reconstruct them.
- A profile change is saved before deployment; the active container retains its prior resolver configuration until the selected Version is deployed.
