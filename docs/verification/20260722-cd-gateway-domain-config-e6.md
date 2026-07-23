# CD E6：网关领域（Gateway domain）— 验证记录
最后修改时间: 2026-07-23 10:35:22

Review status: Draft

Flow mode: strict

路线位置：cadence **扩展 E6**。  
前置：R3 / E5 / E1 已交付；本文件验收 E6（managed only；无 external；无旧数据迁移）。

相关文档：

| 阶段 | 路径 | 状态（验证时） |
|------|------|----------------|
| Requirement | `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | Accepted |
| Spec | `docs/spec/20260722-cd-gateway-domain-config-e6.md` | Accepted |
| Plan | `docs/plan/20260722-cd-gateway-domain-config-e6.md` | Accepted |
| 实现 | 工作树未提交（`feature/cd-redesign` 相对 `4981a5f`） | 已落地 |

## Requirement alignment

| 需求要点 | 结论 |
|----------|------|
| Gateway domain 产品语义 + UI | **符合** — 侧栏「网关」`/cd/gateways`；列表 + 详情表单 |
| 仅 managed 容器 Traefik；无 external | **符合** — 无 external 形态/切换；创建即 `kind=gateway` + config |
| 工作负载仍 Application + Version + Deploy | **符合** — compile 写 unpublished Version；「版本与部署」进 Application 详情 |
| Gateway config 编译进 Version | **符合** — `CompileGatewayToVersion` 于 Create/Update 保存即 compile |
| Version 完整挂载保留（E5 路径） | **符合** — 非托管 target 保留；UI 仍可进 Version 编辑 |
| 应用列表默认 standard | **符合** — `ApplicationPage` / create 默认 `kind=standard`；list API 带 kind |
| rest PUT 仍 E5 全量；URL 从 Gateway | **符合** — `publishRouteSnapshot` → `gw.RestAPIURL` |
| Env 剥离 base_domain；无 domain_template | **符合** — migration 11 去列；proto/UI 无字段 |
| 全局 api_url / domain_suffix 产品路径移除 | **符合** — `TraefikConfig` 仅 `CertDir`；config.yaml / BindEnv / `.env.example` 已无死键（cleanup `20260723-cd-post-e6-cleanup`） |
| 不迁移旧数据 / 无兼容层 | **符合** — 无 dual-read；无 data backfill |
| 不吞并 F1–F7 | **符合** — PEM/鉴权/多 gateway/dashboard 产品化未做 |

## Spec alignment

| Spec 决策 | 结论 |
|-----------|------|
| D1 Gateway = Application(kind=gateway) | **符合** |
| D2 `gateway_config` 1:1；禁止 application 脏列 | **符合** — 表 + model；rest/base_domain/image 在 config 表 |
| D2.1 A 组 rest_api_url 平台唯一读源 | **符合** — RouteManager / ListRouters 入参 restAPIURL |
| D2.1 B 组 base_domain 在 Gateway | **符合** — `deriveHost` 用 gateway.BaseDomain |
| D2.1 C 组部署意图 compile | **符合（最小集）** — image + 默认 ports/mounts/static yml；**未**扩表 ports/log_level/acme 列（Plan：最小字段集 OK） |
| D3 保存 compile；挂载按 target upsert | **符合** — `mergeManagedGatewayMounts` |
| D4 Env 去 base_domain / domain_template | **符合** — SQL + API + UI |
| D5 移除全局 api_url/domain_suffix | **符合** — 结构体、yaml、BindEnv、示例 env 对齐 |
| D6 导航 / list kind 过滤 | **符合** |
| D7 单 active gateway + rest 写路径 | **符合** — 既有 `HasActiveGatewayService`；rest 走 Gateway URL |
| 无 external / F1–F7 | **符合** |

## Plan alignment

| Plan 切片 | 结论 |
|-----------|------|
| E6-A 表 + CRUD + 导航 + Env 去 domain | **完成** — migration 11；handler/routes；GatewayPage/Detail |
| E6-B Host/rest 改读 Gateway；去全局配置 | **完成** — compose/route/deployment 注入；config 瘦身 |
| E6-C CompileGatewayToVersion + UI 反馈 | **完成** — `gateway_compile.go`；详情 hint + toast |
| E6-D 测试 + guides | **完成** — 本轮测试；deployment / routing / volume-mounting 更新 |
| P8 当前 Gateway 解析 | **完成** — `ResolveActiveGatewayConfig`（active Service → 唯一 gateway app） |
| 路由单数 `/api/cd/gateway` | **完成** |

## Actual diff summary

相对 `4981a5f`（E5 verification 提交）工作树变更（未 commit）：

1. **Schema**：`000011_cd_gateway_config` 建 `gateway_config`；environment 去掉 base_domain/domain_template。  
2. **领域 API**：`gateway.proto` + handler/mapper/routes；Create 事务写 app+config 后 compile。  
3. **控制面**：`ApplySnapshot`/`ListRouters` 必填 restAPIURL；publish 从 Gateway 解析。  
4. **数据面 Host**：`deriveHost` = `{app_code}.{gateway.base_domain}`；Env 无 domain 字段。  
5. **Compile**：托管 component `traefik`；default image `traefik:v3.6`；ports 80/443/8080；docker.sock + traefik.yml(sync) + acme.json(seed)。  
6. **前端**：导航 Gateways；Gateway 列表/详情；应用默认 standard；Env 表单去 domain。  
7. **配置**：`configs/config.yaml` 去掉 api_url/domain_suffix；settings 去掉 domain_suffix 键。  
8. **文档**：Req/Spec/Plan；guides 路由改为 rest；本 Verification。

## Expected vs actual changed files

| Plan 预期面 | 实际 |
|-------------|------|
| SQL 000011 | `sql/migration/{sqlite,mysql}/000011_cd_gateway_config.*` |
| Model/Repo | `GatewayConfig`；Get/Upsert/ResolveActive；CreateWithConfig 事务 |
| Config | `TraefikConfig.CertDir` only；config.yaml 对齐 |
| Usecase | `gateway.go` + `gateway_compile.go`；environment/compose/route/version/deployment |
| Traefik | `route.go` URL 参数 |
| Proto/HTTP | gateway.proto + gen；routes/handler |
| Web | GatewayPage/Detail、nav、router、i18n、Application/Environment |
| Tests | migration v11；compose Host；compile 单测/集成；traefik/config |
| Docs | guides + plan notes |

## Acceptance criteria checklist（Plan V1–V9）

| # | 验收项 | 结果 |
|---|--------|------|
| V1 | migrate version 11；有 gateway_config；environment 无 base_domain | **通过** — `migration_test` 期望 version **11**；up SQL 明确 |
| V2 | 创建 gateway 写 config；list kind 过滤 | **通过** — 集成 `seedTestGateway` / CreateGateway；`ListApplications(..., kind)`；前端 kind=standard |
| V3 | 无全局 api_url 产品路径；PUT 使用 rest_api_url | **通过** — 结构体无字段；`publishRouteSnapshot` 用 `gw.RestAPIURL`；traefik 单测 |
| V4 | Host 推导用 gateway.base_domain | **通过** — compose 单测 + `deriveHost` |
| V5 | Env API/UI 无 base_domain | **通过** — proto/mapper/EnvironmentPage 检索无字段 |
| V6 | 导航 Gateway；应用默认 standard | **通过（代码）** — navigation + ApplicationPage；**未**跑浏览器 E2E |
| V7 | compile 后 mounts 校验通过 | **通过** — `gateway_compile_test` + `TestCreateGatewayCompilesManagedVersion` |
| V8 | go test/vet；yarn typecheck；lint:fix | **通过** — 见 Test results |
| V9 | 单 active gateway 仍拦截 | **通过** — deploy 校验 + `TestDeploySecondGatewayRejectedWhenAnotherGatewayActive`（cleanup） |

### Requirement 场景核对

| 场景 | 结果 |
|------|------|
| 创建/编辑 Gateway 领域表单 | **通过** — CRUD + 校验 rest URL / base_domain |
| 保存后 Version 可 Preview/Deploy 出 rest-capable 规格 | **通过** — mounts 含 providers.rest；**用户实机**已启 managed Traefik 并部署其他应用（2026-07-23） |
| 应用列表不混 gateway 主路径 | **通过** — 默认 kind=standard |
| 无 gateway 时 Host 失败引导 | **通过（实现）** — `no gateway configured` 文案 |
| 多 gateway 无 active 解析失败 | **通过（实现）** — Resolve 错误文案；二次部署拦截有专用单测 |

## Test results

| 命令 | 结果 |
|------|------|
| `go fmt ./cmd/... ./internal/...` | **ok** |
| `go vet ./cmd/... ./internal/...` | **ok** |
| `go test ./cmd/... ./internal/... -count=1` | **ok**（含 usecase compile/integration、database migrate=11、traefik、config、e2e） |
| `yarn --cwd web lint:fix` | **ok**（0 errors；19 既有 warnings：gen eslint-disable、dayjs 等） |
| `yarn --cwd web typecheck` | **ok** |

重点新增/相关用例：

- `TestBuildTraefikStaticConfigEnablesRestAndDocker` / merge mounts / image default  
- `TestCreateGatewayCompilesManagedVersion`（create + update 复用 unpublished）  
- compose Host / gateway dashboard labels 夹具改用 `GatewayConfig`  
- `migration_test` version **11**

## Missed or expanded scope

| 项 | 说明 |
|----|------|
| C 组完整列（ports/log_level/acme_enabled 等） | **未扩表** — 以默认常量 compile；与 Plan「最小字段集」一致，**不算**范围膨胀 |
| BindEnv `api_url` / `domain_suffix` | **已删**（cleanup） |
| config 单测 fixture | **已改为仅 cert_dir**（cleanup） |
| cert_dir 保留 | Spec/Plan 允许 F1 前短期保留 |
| liquid 测试变量 | cleanup 已改为 `config.base_domain` 示例变量名；**不**再以全局 domain_suffix 作配置语义 |
| 浏览器 E2E | 未做；用户实机路径已验收 |

## Risks

1. 无 gateway 时 standard HTTP Expose 无法推 Host — 依赖运维先建网关（文案已有）。  
2. 多 gateway 且无 active Service 时 Resolve 失败 — 需用户部署或删减。  
3. compile 与手改 Version：托管 target 每次覆盖；自定义 mount 保留，但用户改 `traefik` 组件非托管字段可能被 image/ports 重写。  
4. 默认 static yml 含 `api.insecure` / `rest.insecure` — 仅适合内网 managed；F2 鉴权后续。  
5. ~~BindEnv 死键~~ — cleanup 已删。  
6. ~~单 gateway 拦截缺专用单测~~ — cleanup 已补。

## Incomplete items

1. **V6** 无浏览器自动化（非阻塞）。  
2. 本 Verification **待用户显式 Accept** Review status。  
3. **F1–F7** / external 不在范围；用户将 F1 与 external 排除出强制后续。  
4. 可选 backlog：F2–F6 / E2–E4 / K8s（见 cadence）。

## Conclusion

在 Req / Spec / Plan 边界内，E6 **主交付已落地且可验收**：

- `gateway_config` + Gateway CRUD/导航  
- rest URL 与 base_domain 归属 Gateway；Env/全局产品路径剥离  
- Host 固定 `{app_code}.{gateway.base_domain}`  
- 保存即 compile 托管 Version（rest-capable 最小规格）  
- 应用列表默认 standard；Version 高级路径保留  

用户实机（2026-07-23）：可启动 managed Traefik 并部署其他应用。  
cleanup（`20260723-cd-post-e6-cleanup`）：关闭死 BindEnv / fixture / V9 专用测。  

建议用户将本 Verification 标为 **Accepted**；应用版本化主 cadence 无强制下一主需求。

---

**当前：严格模式 / strict，验证 / Verification — Review status: Draft（主路径完成；待用户 Accept）**
