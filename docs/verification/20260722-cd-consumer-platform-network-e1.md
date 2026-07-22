# CD E1：standard consumer 接入平台网 — 验证记录
最后修改时间: 2026-07-22 18:03:32

Review status: Accepted

Flow mode: standard

路线位置：R3 扩展 **E1**（`docs/requirement/20260722-cd-application-kind-gateway.md` 扩展表）。  
相关：E5 同批实现（挂载/rest/表单等）不在本文件完整验收；本验证 **只闭合 E1 consumer 网络**。  
人工冒烟：`kind=standard` 应用 **nginx** Deploy 后经 Traefik 可达。

## Requirement alignment

| 需求要点（E1） | 结论 |
|----------------|------|
| `standard` **有 Expose** 时自动注入平台网 `external: true` | **符合** — top-level `networks.traefik.external: true` + `name: traefik` |
| exposed 组件 **join** 平台网 | **符合** — service `networks: [default, traefik]` |
| 无 Expose 时 **不**注入 | **符合** — 单测 `TestRenderComposeStandardWithoutExposeDoesNotJoinPlatformNetwork` |
| 仅 **有 Expose 的组件** join（兄弟组件不强制） | **符合** — `TestRenderComposeStandardExposeOnlyJoinsExposedComponents` |
| 与 gateway **创建网**语义区分 | **符合** — gateway 仍 `driver: bridge` 提供者；consumer 为 `external` |
| 网络名默认 `traefik`（与 gateway D4 默认一致） | **符合** — `defaultGatewayNetworkName` |

## Spec alignment

| 项 | 结论 |
|----|------|
| 独立 E1 Spec | **不适用** — E1 写在 R3 Requirement 扩展表；实现落在 `compose_renderer` |
| 不破坏 R3 standard labels / gateway 分支 | **符合** — labels 路径保留；gateway 路径单测仍过 |

## Plan alignment

| 项 | 结论 |
|----|------|
| 独立 E1 Plan | **不适用** — 作为 R3 扩展缺口修复实现；无单独 Plan 文件 |
| 实现点 | `renderComposeServices` + `injectConsumerPlatformNetwork` + 单测 |

## Actual diff summary（E1 相关）

1. **`compose_renderer.go`**：standard + Expose → 注入 external 平台网；exposed 组件 join `default` + `traefik`。  
2. **`compose_test.go`**：有/无 Expose、仅 exposed 组件 join 的断言。  
3. **文档**：R3 扩展表 / cadence 标注 E1 **已实现**。  
4. **同分支其它改动（E5 等）**：mount content、rest 路由、Version 表单 ports/env/mounts 等 — **不计入本 E1 验收通过条件**（见 Incomplete）。

## Expected vs actual changed files（E1）

| 预期 | 实际 |
|------|------|
| `compose_renderer.go` consumer 网络注入 | 有 |
| `compose_test.go` E1 单测 | 有 |
| cadence / R3 扩展说明 | 有 |

## Acceptance criteria checklist

| # | 标准 | 结果 |
|---|------|------|
| A1 | Preview/Deploy compose：有 Expose 含 `external: true` 与 `name: traefik` | **通过** — `data/cd/nginx/local/default/docker-compose.yml` |
| A2 | 有 Expose 的 service join `default` + `traefik` | **通过** — 同上 + `docker inspect nginx_nginx` 双网 |
| A3 | Traefik 能转发到后端（非 504） | **通过** — `curl -H "Host: nginx.lvh.me" http://127.0.0.1/` → **200** + nginx 欢迎页 |
| A4 | Traefik 路由 enabled | **通过** — `nginx-local-default-nginx-http@docker` status enabled |
| A5 | 无 Expose 不注入平台网 | **通过** — 单测 |
| A6 | 自动化测试 | **通过** — 见下 |

### 运行时证据（2026-07-22）

| 检查 | 结果 |
|------|------|
| `nginx_nginx` Networks | `traefik` + `nginx-local-default_default` |
| `traefik_traefik` Networks | `traefik` |
| HTTP `Host: nginx.lvh.me` → `:80` | **200** |
| 修复前 | **504** / `dial tcp … i/o timeout`（跨网） |

## Test results

| 命令 | 结果 |
|------|------|
| `go test ./internal/application/cd/usecase/ -count=1` | **ok** |
| `go test ./internal/application/cd/... ./internal/api/http/handler/cd/... -count=1` | **ok** |
| `go test ./internal/infrastructure/external/traefik/ ./internal/config/ -count=1` | **ok** |
| `go vet ./internal/application/cd/usecase/` | **ok** |
| `yarn --cwd web typecheck` | **ok** |
| `yarn --cwd web lint:fix` | **ok**（0 errors；既有 warnings，与 E1 无关） |

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| 同分支 E5 大包 | mounts/rest/UI 等与 E1 同工作区；**本 verification 不声称 E5 全量 Accepted** |
| Environment 可配网络名 | E5 D4 对 gateway 可覆盖名；E1 初版固定 `traefik`，与当前 gateway 默认一致 |
| E6 网关领域 | Draft 路线，未实现 |

## Risks

1. 平台网必须已由 gateway（或外部）创建且名为 `traefik`；否则 `external: true` 的 compose up 会失败。  
2. 多 Environment / 多网络名（F3/E6）时固定名可能不够 — 后续与 gateway 解析名对齐。  
3. 工作区仍含 E5 未单独验收改动 — 提交前建议按主题拆 commit 或另开 E5 Verification。

## Incomplete items

1. **E5** 完整 Verification 文档与清单未做。  
2. **E6** 仍为 Requirement Draft。  
3. 本文件 Review status 待用户 **Accept**。

## Conclusion

**E1 consumer 网络：实现与运行时冒烟通过。**  
standard + Expose 的 nginx 经 Traefik `Host(nginx.lvh.me)` 返回 **200**；compose 含 external 平台网；单测覆盖有/无 Expose 与仅 exposed 组件 join。  
**建议用户 Accept 本 Verification 后**，将 E1 视为 R3 扩展闭合；E5 另走验收。
