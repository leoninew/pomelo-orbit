# CD E5：Component 全规格 / 部署运行时 / rest 路由 — 验证记录
最后修改时间: 2026-07-23 10:35:22

Review status: Draft

Flow mode: strict

路线位置：cadence **扩展 E5**。  
前置：R3 Verification Accepted；E1 Verification Accepted（consumer 网；本文件不重复验收 E1）。  
不交付：**E6** 网关领域；**F1–F7** 后续任务。

相关文档：

| 阶段 | 路径 | 状态（验证时） |
|------|------|----------------|
| Requirement | `docs/requirement/20260722-cd-gateway-component-mount-e5.md` | Accepted |
| Spec | `docs/spec/20260722-cd-gateway-component-mount-e5.md` | Accepted |
| Plan | `docs/plan/20260722-cd-gateway-component-mount-e5.md` | Accepted |
| 实现提交 | `6a784f9` … `b991106`（相对 R3 `0bf1523`） | 已落地 |

## Requirement alignment

| 需求要点 | 结论 |
|----------|------|
| Version 表单/API：结构化 mounts / env（及 ports）；非巨型 JSON 主 SoT | **符合** — `MountSpec`/`EnvVarSpec` + Version API；UI 表单行编辑 |
| mount content + content_mode（seed\|sync）；仅 logical 文件型 | **符合** — `mount_env.go` 校验；单测 seed/sync/目录拒绝 |
| 禁止 absolute logical source 作 Version 真相 | **符合** — `isAbsoluteMountSource` + `TestParseMountSpecsRejectsUnixAbsoluteLogicalSource` |
| special 白名单 `docker.sock` | **符合** — 解析为 `/var/run/docker.sock`；单测 |
| RuntimeConfig 部署面；缺键失败；不回写 Version | **符合** — `resolveDeployRuntimeConfig`；Deploy options JSON；proto `runtime_config` |
| 全 kind 共用挂载解析；gateway 无特殊路由目录树 | **符合** — 同一 `resolveMountSpecs` / workspace；无 dynamic 路由目录强制 |
| gateway top-level network（默认 `traefik`） | **符合** — `renderGatewayCompose` + `defaultGatewayNetworkName` |
| 平台动态路由 `providers.rest` PUT 全量；废除 File 写路由 | **符合** — `RouteManager.ApplySnapshot` → `PUT …/api/providers/rest`；无 per-route yaml 写路径 |
| 单 gateway 部署约束 | **符合** — `HasActiveGatewayService` + deploy 校验文案 |
| 物化替代 init.sh（目录 MkdirAll；文件 seed/sync） | **符合** — `MaterializeLogicalMountSources`；部署路径调用；无 bash init 步骤 |
| **不**交付 E6 / F1–F7 产品面 | **符合** — 无 Gateway domain UX；PEM 写盘仍为 Route 侧遗留接口（F1），非 E5 产品验收主路径 |

## Spec alignment

| Spec 决策 | 结论 |
|-----------|------|
| D1 挂载 schema / content / special | **符合** |
| D2 env 占位 + RuntimeConfig | **符合** — `${NAME}` / `${NAME:-default}` 扫描与解析 |
| D3 ports / command 表单 | **符合** — 前端 ports 行；command 既有路径保留 |
| D4 gateway network；standard 不注入该块 | **符合** — compose 单测 gateway vs standard |
| D5 部署目录平等 | **符合** — `WriteConfig` 仅 compose；无 gateway 专用 dynamic 树 |
| D6 rest 全量 PUT / 清空语义 / 串行锁 | **符合** — `putRestConfig` + `mu`；`TestRouteManagerApplySnapshotPutsFullRestConfig` / Clears |
| D6.3 废除 File 写路由 | **符合** — 产品路径无 `deployRouteFile`；配置项 `dynamic_route_dir` 已从 config 移除 |
| D6.4 自定义 PEM 非本期 | **符合（边界）** — `WriteCertificate` 仍存在（历史/Route HTTPS 路径），**不**作为 E5 挂载/rest 主交付；完整对齐 F1 |
| D9 单 gateway | **符合** — 代码拦截 + `TestDeploySecondGatewayRejectedWhenAnotherGatewayActive`（cleanup） |
| F1–F7 / E6 不吞并 | **符合** |

## Plan alignment

| Plan 步骤 / 约束 | 结论 |
|------------------|------|
| C：rest 替换 File 写路由 | **完成** — `traefik/route.go` + tests；config 去 dynamic 目录依赖 |
| D9：单 gateway | **完成** — deploy 前 `HasActiveGatewayService` |
| A：mounts/env + 物化 | **完成** — `mount_env.go` + render + materialize |
| B：RuntimeConfig + Deploy | **完成** — deploy options + proto + UI runtime_config |
| D4：gateway network | **完成** |
| P-init：不执行 init.sh | **完成** — 部署只 WriteConfig compose + materialize |
| 前端 Version 表单 | **完成** — `ApplicationDetail.vue` ports/env/mounts/content |
| Out of scope E6/F* | **未误交付** |

## Actual diff summary

相对 R3 基线 `0bf1523`（E5 相关提交 `6a784f9`…`b991106`）：

1. **路由**：`RouteManager` 改为 rest 全量 PUT；集成测与单测对齐；config 去掉 dynamic 路由目录 SoT。  
2. **挂载/env**：新 `mount_env.go`（parse/validate/resolve/materialize/RuntimeConfig）；compose render 解析 volumes/environment。  
3. **Deploy**：物化 logical 源 → 写 compose → runner；RuntimeConfig 写入 deployment options。  
4. **单 gateway**：repository `HasActiveGatewayService` + deploy 校验。  
5. **API/proto**：`ApplicationDeployReq.runtime_config`；Version component mounts 结构化读写。  
6. **前端**：Version 编辑 ports/env/mounts/content_mode；Deploy runtime_config。  
7. **同批 E1**：standard+Expose join 平台 external 网（**另**有 `docs/verification/20260722-cd-consumer-platform-network-e1.md` Accepted）。  
8. **文档**：E5 Req/Spec/Plan；E6 Draft；cadence；E1 verification。

## Expected vs actual changed files

| Plan 预期面 | 实际 |
|-------------|------|
| `compose_renderer` / `mount_*` / deploy | 有 `compose_renderer.go`、`mount_env.go`、`deployment_execution.go` |
| Traefik client rest | 有 `traefik/route.go` + `route_test.go` |
| DTO / proto / handler | 有 application proto、mapper、dto |
| Repo 单 gateway | 有 `HasActiveGatewayService` |
| Frontend Version + Deploy | 有 `ApplicationDetail.vue` + i18n + gen |
| Tests | compose/materialize/rest 覆盖充分；gateway 二次部署 **无**独立单测 |
| Docs | requirement/spec/plan + cadence + E6 draft |

## Acceptance criteria checklist（Plan V1–V12）

| # | 验收项 | 结果 |
|---|--------|------|
| V1 | rest PUT 全量；无 dynamic 写路由文件 | **通过** — 单测 PUT path；代码检索无 File 写路由主路径 |
| V2 | 单 gateway 二次部署拒绝 | **通过（实现）** — 校验文案；**弱** — 无 `*_test.go` 直接覆盖 |
| V3 | mounts schema；非法 source_type/content 拒绝 | **通过** — parse 单测 + UI 约束 |
| V4 | logical 文件 seed：首次写、二次不覆盖 | **通过** — `TestMaterializeLogicalMountSourcesCreatesFileOnce` / Seed |
| V5 | logical 目录 MkdirAll | **通过** — materialize 路径 + 相关单测结构 |
| V6 | 平台不强制 dynamic/certs 树 | **通过** — 部署无强制创建；certs 写仅 Route PEM 遗留（F1） |
| V7 | RuntimeConfig 缺键失败；默认填充 | **通过（实现）** — `resolveDeployRuntimeConfig`；**弱** — 无独立单测文件断言 |
| V8 | gateway compose 含 network 默认 traefik；standard 无该块 | **通过** — gateway/standard compose 单测 |
| V9 | Preview/Deploy 同解析 | **通过** — `RenderComposeDetailed` 共用；`TestApplicationComposePreviewMatchesDeployExposeLabels` |
| V10 | 部署日志体现物化（可选） | **未强制** — 可选；未做日志断言 |
| V11 | 前端 lint/typecheck | **通过** — 见 Test results |
| V12 | 无 init.sh 执行步骤 | **通过** — deploy 路径代码审查 |

### Requirement 场景核对

| 场景 | 结果 |
|------|------|
| 表单维护挂载/env | **通过** — UI + API |
| 部署 RuntimeConfig | **通过** — Deploy 字段 + 解析 |
| gateway 网络合并 | **通过** — render |
| standard 共用挂载 | **通过** — 同解析 |
| 平台 rest 路由 / 目录无差别 | **通过** |
| 取消路由 = 快照 PUT | **通过** — empty maps 单测 |

## Test results

| 命令 | 结果 |
|------|------|
| `go test ./internal/application/cd/usecase/ -count=1` | **ok**（含 materialize / mounts / gateway dashboard / E1 network） |
| `go test ./internal/infrastructure/external/traefik/ -count=1` | **ok**（含 ApplySnapshot PUT / clear） |
| `go test ./internal/application/cd/... -count=1` | **ok**（与上同批） |
| `go fmt ./cmd/... ./internal/...` | **ok** |
| `go vet ./cmd/... ./internal/...` | **ok** |
| `yarn --cwd web typecheck` | **ok** |
| `yarn --cwd web lint:fix` | **ok**（0 errors；既有 warnings：gen eslint-disable、dayjs 等，与 E5 无关） |

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| **E1 同批** | consumer 平台网在 E5 提交窗口一并落地；已有独立 Verification Accepted，**不**算 E5 范围膨胀为未声明能力 |
| **gateway dashboard labels** | kind=gateway × IngressPolicy 注入 api@internal labels — 属 R3/E5 观测便利；**F5** 产品化仍后续 |
| **Route WriteCertificate** | rest 路径仍可写 PEM 到 cert 目录；Spec 将完整 PEM 对齐标 **F1**，本期不验收为 E5 主目标 |
| **Environment 覆盖网络名** | Plan Assumption：默认硬编码 `traefik`；Environment 字段覆盖若未入库则按常量 — **符合** Plan 假设 |
| **E6** | 仅文档 Draft，无产品代码冒充领域配置 |

## Risks

1. rest 全量 PUT 依赖平台唯一写者；并发靠进程内 mutex，多实例部署未覆盖。  
2. Traefik 须 ≥3.6 且静态启用 `providers.rest`；平台不改用户静态 yml（E6 managed 再封装）。  
3. ~~单 gateway 专用单测~~ — cleanup 已补；RuntimeConfig 缺键路径仍可按需加强。  
4. PEM 写盘路径与「无强制 certs 树」叙事并存 — F1 可选，非主线必做。  
5. 前端 lint 既有 warnings 未清；不阻塞 E5。

## Incomplete items

1. **V2 / V7 / V10** 部分专用断言仍可加强（非主路径阻塞）。  
2. 本 Verification **待用户显式 Accept** Review status。  
3. **E6 已交付**（见 `docs/verification/20260722-cd-gateway-domain-config-e6.md`）；非本文件 Incomplete。  
4. **F1–F7** 为可选 backlog（F1/external 用户排除出强制后续）。

## Conclusion

在 Req / Spec / Plan 边界内，E5 **主交付已落地且可验收**：结构化挂载与 content 物化、RuntimeConfig 部署面、gateway 网络、rest 全量路由、单 gateway 约束、全 kind 共用解析、前端表单，以及废除 File 写平台路由。

用户实机路径与 E6 一并验收；cleanup 补强单 gateway 专用测。  
建议用户将本 Verification 标为 **Accepted**。

---

**当前：严格模式 / strict，验证 / Verification — Review status: Draft（主路径完成；待用户 Accept）**
