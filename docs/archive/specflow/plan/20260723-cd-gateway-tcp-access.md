# CD：Gateway TCP 对外访问 — 实现计划
最后修改时间: 2026-07-23 13:30:00

Review status: Accepted

Flow mode: strict

## Requirement / Spec basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260723-cd-gateway-tcp-access.md` | **Accepted** |
| `docs/spec/20260723-cd-gateway-tcp-access.md` | **Accepted**（进入 Plan 时确认） |
| `docs/requirement/20260723-cd-gateway-entrypoint-ownership.md` | Accepted；本期删除 `tcp_entrypoint` |
| `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | Accepted；`base_domain` / Host |
| 当前 migrate | **12**（`000012_cd_gateway_entrypoint_policy`）；本期 **000013** |

**硬约束（不得偏离 Spec）**：

1. 出口 SoT = **Version Expose**（`protocol` + `access`）；无 Gateway TCP 路由 CRUD。  
2. 三层：**集群内**（不依赖 expose）/ **local**（loopback ports）/ **public**（HTTP 现逻辑 + TCP 网关）。  
3. 内网名/容器名：`{app_code}-{component}`；**单 app 单运行套**；`instance_key` **仅允许 `default`**。  
4. public 域名：`{app}.{base_domain}`；**不强制 HTTPS**；`tls_mode=none` 可测。  
5. 删除 `gateway_config.tcp_entrypoint` 与产品面 TCP→web/websecure。  
6. **不**改已执行 migration 文件；新 migration sqlite+mysql 成对。  
7. 门禁：`task check` / `task test`；前端 `yarn --cwd web lint:fix` + `typecheck`。  
8. 缩写：`RestApiUrl`、`TLSMode` 等项目内部首字母大写。

## Plan decisions（闭合 Spec Technical questions）

| # | 决策 | 说明 |
|---|------|------|
| P1 | migration | **000013**：`version_expose` 增 `access`/`listen_port`；`gateway_config` **DROP** `tcp_entrypoint`；`migration_test` 期望 **13** |
| P2 | access 默认 | 列 `access TEXT NOT NULL DEFAULT 'public'`；旧行平滑为 public |
| P3 | listen_port | `INTEGER NULL`；业务层 `effectiveListen = listen>0 ? listen : container` |
| P4 | instance_key | Deploy/Preview：**非 `default`（空按 default）→ validation 拒绝**；不静默改写 |
| P5 | 单运行套 | standard Deploy：若 `application_id` 已有 **非终态** Service（running/deploying/… 按现 `ServiceStatus*`）且 key≠本次绑定，**拒绝**并行第二套；同一 `default` 允许原地滚动替换 version |
| P6 | 网络 | standard：**所有** component join `traefik` external + **aliases=`{runtimeName}`** + `container_name=runtimeName`；无 expose 也 join |
| P7 | local | 仅该 component 注入 `ports: ["127.0.0.1:{listen}:{container}"]`；**不**写 Traefik labels（该 expose） |
| P8 | public http | 现 labels；仍需 Gateway + base_domain；entrypoints=`default_entrypoint` |
| P9 | public tcp | 业务 **仅** TCP labels：`entrypoints=tcp{listen}`；`HostSNI(*)` 或 HostSNI(publicHost)；**禁止**业务 host ports 对该 public TCP |
| P10 | Gateway static | `buildTraefikStaticConfig` 接受 **TCP listen 集合 S**：生成 `tcp{p}: address: ":{p}"`；ports JSON 含 `80:80`/`443:443`/`8080:8080` + 每个 `p:p` |
| P11 | ACME | `tls_mode=letsencrypt` 时 static 增加 `certificatesResolvers.letsencrypt`（httpChallenge entryPoint=web，storage 与现 acme.json 挂载一致 `/letsencrypt/acme.json`）；`none`/`tls` 不写 resolver |
| P12 | Gateway reconcile 顺序 | **先**冲突校验 → 算目标 S（活跃 Service 的 public+tcp listen 并集 ∪ 本次 deploy 将产生）→ 若 S ≠ 当前 Gateway 已编译 TCP 集合：`CompileGatewayToVersion(..., S)` → **同步部署 Gateway**（失败则 **中止**业务部署，Service 标 faulted）→ 再渲染/部署业务。Gateway 自身 Deploy 同样按当前 S 编译 |
| P13 | 回滚 | Gateway 已滚动成功、业务部署失败：**不自动回滚 Gateway**（避免二次抖动）；错误信息标明「网关已更新、业务失败」；人工可重部署 |
| P14 | local 与 public 同 listen | **允许**（bind 不同）；文档/注释提示 |
| P15 | normalize | `runtimeName`：小写；非法字符替换 `-`；压缩连续 `-`；trim `-`；仅保留 `[a-z0-9-]`；空则失败 |
| P16 | Gateway 展示 | **扩展 `GET /api/cd/gateway/:id`**（`GatewayResp` 增 `repeated exposures`）；聚合查询，无独立 CRUD RPC |
| P17 | router 命名 | 继续 `sanitize(app-env-instance-component-protocol)` 防撞；内网 DNS **不**用 env/instance |
| P18 | 冲突源 | DB 中活跃 Service（status ∈ running/deploying 及项目既有「非终态失败」约定，与 undeploy 清理一致）+ 其 Version exposes + 本次待部署 exposes |

### 活跃 Service 状态（实现时对照 `constant`）

- 计入占用/展示：**running**、**deploying**（若存在 intermediate）。  
- 不计入：**stopped**、**faulted**（或项目已有终态集合）— 实现时与现 Service 列表/undeploy 语义对齐，单测钉死。

### Gateway reconcile 伪代码

```text
func DeployStandard(...):
  if instance_key != "default": reject
  if other active service for app: reject  // 单运行套
  validateExposes(access, listen, path+tcp, ...)
  conflictCheck(local + public-tcp, including self-replace)
  S = union publicTCP listens from active services (exclude self) ∪ self publicTCP
  if gateway.compiledTCP != S:
    CompileGateway(S)
    DeployGateway(force)  // fail → abort
  RenderCompose(standard) + Deploy
```

## Implementation steps

### Step A — Schema + model + repo（无行为变化可先合并）

#### A1. Migration `000013_cd_gateway_tcp_access`

**sqlite + mysql 成对**（风格对齐 000012 / 000009）：

1. `version_expose`：  
   - `access TEXT NOT NULL DEFAULT 'public'`  
   - `listen_port INTEGER NULL`（mysql：`INT NULL`）  
2. `gateway_config`：DROP `tcp_entrypoint`  
   - sqlite：按现网 **rebuild table** 模式（见 000012 down / 000011）  
   - mysql：`ALTER TABLE ... DROP COLUMN tcp_entrypoint`  
3. **不**改 000009–000012 已执行文件内容。  
4. `migration_test` / e2e / `app_test`：期望 version **13**。

#### A2. Model / DTO / SQL 列

- `model.VersionExpose`：`Access string`，`ListenPort *int`  
- `model.GatewayConfig`：**删除** `TCPEntrypoint`  
- `dto.VersionExposeInput` / Gateway create-update：**同步**  
- repo `versionExposeColumns` / insert / gateway columns：去掉 `tcp_entrypoint`，写入 access/listen_port  
- sqlc 若本仓生成：按项目流程 regenerate；否则仅 sqlx 列字符串

**A 验收**：migrate=13；读写 expose 带 access；gateway 无 tcp 列编译通过。

---

### Step B — Proto + API 契约

#### B1. `version.proto`

```text
// VersionExposeReq / Resp
string access = N;              // "local" | "public"
optional int32 listen_port = M; // 0/omit → default container
```

校验（usecase create/update version）：

- access 必填语义：缺省 **public**（与 DB default 一致）或显式拒绝空串—Plan：**空串/omit → public**  
- listen_port：若 set 则 1–65535  
- protocol=tcp 且 path_prefix 非空 → 拒绝  

#### B2. `gateway.proto`

- Create/Update/Resp：**删除** `tcp_entrypoint`  
- Resp 增加：

```text
message GatewayExposureItem {
  string application_id = 1;
  string application_code = 2;
  string component_name = 3;
  string protocol = 4;
  string access = 5;
  int32 container_port = 6;
  int32 listen_port = 7;       // effective
  string public_host = 8;
  string internal_dns = 9;
  string client_hint = 10;
}
// GatewayResp
repeated GatewayExposureItem exposures = N;
```

- List 可 **不**带 exposures 明细（避免 N+1）；Get 带满量。  
- `task proto` 生成 Go + `web/src/gen`。

#### B3. Mapper / HTTP

- version create/update/list/get 映射 access/listen_port  
- gateway get：调用聚合 `ListActiveExposures(ctx)`  
- gateway create/update：去掉 tcp 字段处理  

**B 验收**：buf/gen 无旧 tcp 字段；OpenAPI/前端 gen 对齐。

---

### Step C — Compose 渲染（核心）

文件主战场：`internal/application/cd/usecase/compose_renderer.go` + `compose_test.go`。

#### C1. 公共 helper

- `effectiveListen(e)`  
- `runtimeName(appCode, component)`（P15）  
- `publicHost(gateway, appCode)`（现 `deriveHost`）  
- `tcpEntrypointName(listen) = "tcp" + strconv.Itoa(listen)`  

#### C2. container_name + 全量 join 网

- `renderVersionComponentService`：`container_name = runtimeName`（替换 `app_code_component`）  
- standard：**始终** `injectConsumerPlatformNetworkAll`：每个 service  
  - `networks.default: {}`  
  - `networks.traefik.aliases: [runtimeName]`  
- 顶层 external `traefik` **始终**出现（standard），**不**再要求 `len(Exposes)>0`  

#### C3. 按 access 分流 `injectExposeLabels` 重写

对每条 expose：

| access × protocol | 动作 |
|-------------------|------|
| local + * | 追加 ports `127.0.0.1:listen:container`；**无** traefik labels |
| public + http | 现 HTTP labels；需 gateway |
| public + tcp | TCP labels：`entrypoints=tcp{listen}`；HostSNI 规则按 tls_mode；**无** ports |
| （缺 access） | 当 public |

- **删除** `TCPEntrypoint` / `validGatewayEntrypoint` 用于 TCP 的 web/websecure 路径。  
- public tcp **不再**要求 default_entrypoint 为 web（HTTP 仍要）。  
- public 任一协议：无 Gateway / 无 base_domain（HTTP 或 TLS TCP）→ 明确错误。  
- `tls_mode=none` 的 public tcp：不需要 base_domain 也可（HostSNI `*`）；但展示 public_host 仍尽量用 base_domain（无则 client_hint 用 listen only）。

#### C4. 单测矩阵（`compose_test.go`）

1. 无 expose：仍 join traefik + alias + container_name=`{app}-{comp}`  
2. local tcp：ports loopback，无 tcp labels  
3. public tcp none：HostSNI `*` + entrypoints `tcp6379`  
4. public tcp tls/le：HostSNI host + tls labels（le + certresolver）  
5. public http 回归  
6. 无 gateway + public http → error  
7. 双 local 同 listen 同 version → validate error（C/D 可放 validation 单测）

---

### Step D — Gateway compile + public TCP 端口集

文件：`gateway_compile.go` / `gateway_compile_test.go` / `gateway.go`。

#### D1. API 形状

```go
// CompileGatewayToVersion(ctx, app, cfg, tcpListens []int) (versionId, error)
// tcpListens 已排序去重；可为空
```

- `buildManagedGatewayComponent(cfg, existing, tcpListens)`  
- `buildTraefikStaticConfig(cfg, tcpListens)`：web/websecure + `tcp{p}`；可选 ACME（P11）  
- ports：`80/443/8080` + 每个 tcp listen 的 `"{p}:{p}"`  

#### D2. Create/Update Gateway

- 保存后 compile：S = **当前活跃 public TCP 聚合**（无业务部署时可能空）  
- 去掉 tcp_entrypoint 归一化逻辑  

#### D3. 单测

- static yaml 含 `tcp6379` / address `:6379`  
- ports JSON 含 `6379:6379`  
- letsencrypt 含 certificatesResolvers  
- none 不含 resolver  

---

### Step E — Deploy / Preview / 冲突 / 单运行套

文件：`deployment_execution.go`、version deploy API、preview、store 接口。

#### E1. instance_key + 单运行套

- 解析后非 `default` → `KindValidation`  
- `ServicesByApplication`：存在其它活跃绑定 → 拒绝（文案说明单运行套）  

#### E2. 冲突校验模块（新建小函数文件可 `expose_conflict.go`）

输入：待部署 app/version exposes + 全局活跃占用表。

规则（Spec Validation 表）：

- public 明文 TCP：全局同 effectiveListen 仅一条（排除自身替换）  
- public TLS TCP：同 listen + 同 publicHost 拒绝  
- local：同 `127.0.0.1:listen` 拒绝  
- local 与 public 同 listen：**允许**  

占用表构建：遍历活跃 Service → VersionExposes → 分类。

#### E3. Gateway reconcile 接入 `ExecuteApplicationDeploy`

- standard 且 exposes 需 public 或端口集变化：P12 顺序  
- 需 **deploy gateway app** 复用现有 `ExecuteApplicationDeploy`/`renderAndDeploy`（注意防递归：gateway 部署 **不再** 因自身触发二次业务逻辑；仅 compile+self deploy）  
- Preview：同样跑冲突校验 + 渲染（可选模拟 S 变化提示，**不**真滚动网关）

#### E4. 聚合 exposures for Gateway Get

```go
// ListActiveGatewayExposures(ctx) []GatewayExposureItem
// internal_dns, client_hint:
//   local → "127.0.0.1:{listen}"
//   public http → "http(s)://{host}" 或 host only
//   public tcp → "{host}:{listen}" 或 ":{listen}"
//   始终填 internal_dns = runtimeName
```

---

### Step F — 前端

#### F1. Version 编辑（`ApplicationDetail.vue` 等）

- Expose 行：协议、container_port、**access**（local|public）、可选 listen_port、HTTP path  
- i18n zh/en  

#### F2. Gateway UI

- `GatewayPage` / `GatewayDetail`：**删除** TCP 入口下拉与 `tcp_entrypoint`  
- 详情页增加 **只读** exposures 表：app、component、protocol、access、ports、public_host、internal_dns、client_hint  
- 创建/更新 payload 去掉 tcp 字段  

#### F3. 部署表单

- instance_key 隐藏或固定 default；若仍展示则非 default 时 API 错误可读  

#### F4. 检查

```text
yarn --cwd web lint:fix
yarn --cwd web typecheck
```

---

### Step G — 清理与回归

1. 全库检索 `tcp_entrypoint` / `TCPEntrypoint` / `TcpEntrypoint` → 清零（含 gen 重生成后）。  
2. 旧 compose 测试期望 TCP 走 web → 改写。  
3. guides（若有 TCP 入口说明）一句更新。  
4. `go fmt` / `go vet` / `go test ./cmd/... ./internal/...`；`task check` / `task test`。  

## Files to change（预期）

| 区域 | 路径（代表） |
|------|----------------|
| migration | `sql/migration/{sqlite,mysql}/000013_cd_gateway_tcp_access.{up,down}.sql` |
| model/dto | `internal/model/cd.go`，`internal/application/cd/dto/cd.go` |
| repo | `internal/repository/impl/sqlx/cd/repository.go`，`internal/repository/cd.go`（若增查询） |
| proto | `proto/orbit/v1/version.proto`，`gateway.proto` + gen |
| compose | `compose_renderer.go`，`compose_test.go` |
| gateway | `gateway_compile.go`，`gateway.go`，`gateway_compile_test.go` |
| deploy | `deployment_execution.go`，可能 `expose_conflict.go`，integration tests |
| frontend | `GatewayPage.vue`，`GatewayDetail.vue`，`ApplicationDetail.vue`，i18n |
| tests | `migration_test.go`，e2e version 断言 |

## Verification plan（实现后 / Verification 阶段执行）

| # | 检查 | 方法 |
|---|------|------|
| V1 | migrate=13；expose 有 access；gateway 无 tcp_entrypoint | migration_test + SQL |
| V2 | 无 expose 组件 compose 含 traefik + alias + `{app}-{comp}` | unit |
| V3 | local ports 127.0.0.1 | unit |
| V4 | public tcp labels entrypoints=tcpN，非 web | unit |
| V5 | Gateway static/ports 含动态 TCP | unit |
| V6 | 明文同 listen 冲突 | unit |
| V7 | instance_key≠default 拒绝 | unit/API |
| V8 | Gateway Get 只读 exposures | unit/API |
| V9 | 前端无 TCP 入口；有 access 字段 | typecheck + 目视 |
| V10 | `task check` / `task test` | CI 本地 |

**人工验收建议**（用户环境）：`tls_mode=none` 下 public HTTP `:80` + public TCP redis-cli；集群内第二 app 用 `{app}-redis:6379`。

## Blockers

- 无硬阻塞。  
- 依赖：单 active Gateway（既有假设 A1）。  
- Gateway 滚动需 docker runner 可用（既有部署路径）。

## Assumptions

| ID | 假设 |
|----|------|
| A-P1 | 平台网名固定 `traefik`（与现 `defaultGatewayNetworkName` 一致） |
| A-P2 | 活跃状态集合与 undeploy 清理一致即可，不新增 Service 状态枚举 |
| A-P3 | Gateway 部署可同步调用现有执行路径，超时/异步语义与今日网关部署相同 |
| A-P4 | ACME httpChallenge 仅 web:80，与业务 public TCP 端口无关 |

## Risks

| 风险 | 缓解 |
|------|------|
| 每次新 TCP 端口滚 Gateway 抖入口 | 仅端口集变化时滚；错误信息清晰 |
| 业务失败后 Gateway 已变 | P13 不自动回滚；文档化 |
| 全量 join 扩大攻击面 | 平台网信任边界；不在 public 时不占宿主端口 |
| container_name 从 `app_comp` 改为 `app-comp` | **破坏性**；单运行套+无双轨，接受；已部署容器需重建 |
| 旧 TCP-on-80 行为消失 | 已接受；测试全改 |

## Rollback

1. 代码：revert feature commit（migration 13 已在环境执行时 **禁止**改 000013 内容；新 000014 恢复列仅作应急，默认不需要）。  
2. 运行：停业务 → 停网关 → 按旧版本 redeploy（需旧镜像/旧分支）。  
3. 数据：`access` 默认 public 向前兼容；回滚应用代码后忽略新列（sqlite 保留列通常可容忍）。

## Out of scope（本 Plan 不实现）

- 多环境并行 / 多 instance  
- Gateway TCP CRUD  
- 依赖注入 env  
- UDP / TLS 透传  
- 改已执行 migration  

## User review notes

- 2026-07-23：Spec Accepted → 撰写 Plan Draft。  
- 请用户审查本 Plan；**接受 / 开始实现** 后进入 Implementation（不写产品代码直至明确指令）。
