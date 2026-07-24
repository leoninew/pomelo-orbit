# CD：Gateway TCP 对外访问（三层可达 · 部署期执行）规格
最后修改时间: 2026-07-23 13:22:00

Review status: Accepted

Flow mode: strict

## Requirement basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260723-cd-gateway-tcp-access.md` | **Accepted**（进入 Spec） |
| `docs/requirement/20260723-cd-gateway-entrypoint-ownership.md` | Accepted；入口/TLS SoT 在 Gateway |
| `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | Accepted；`base_domain`、Host 推导 |
| `docs/requirement/20260721-cd-environment-expose-service.md` | Accepted；Expose 模型基础 |
| `docs/spec/20260722-cd-environment-ingress-policy.md` | 历史；TCP 挂 web/websecure **被本 Spec 取代** |

**硬约束（来自 Requirement，不得偏离）**：

1. 出口意图在 **Version Expose**（`protocol` + `access`）；**非** Gateway TCP 路由 CRUD。  
2. 三层可达：**集群内** / **local** / **public**；集群内 **不依赖** Expose。  
3. local ≈ 平台生成 **`127.0.0.1:listen:container`**；public 域名 **`{app_code}.{base_domain}`**；**不强制 HTTPS**。  
4. public TCP 监听在 **Gateway**；业务不写 `0.0.0.0` 业务 ports。  
5. TLS **终结**到容器明文；LE 复用 certResolver（可选）；开发 `tls_mode=none` 可测。  
6. 删除产品面 TCP→`web`/`websecure`；无历史双轨；不改已执行 migration 文件内容。  
7. `task check` / `task test`。  
8. **简化运行模型（用户 2026-07-23）**：不允许同一 app **同时多套**部署；**不做多环境并发**产品目标（现有 Environment 表可保留兼容，但命名/网关/互访按 **单运行上下文** 设计）。

## Overview / 概览

```text
Version.Expose[]
  protocol: http | tcp
  access:   local | public     ← 新增；无 expose = 无 local/public 出口
  container_port
  listen_port?                 ← 可选，默认 = container_port
  path_prefix?                 ← 仅 http

Deploy / Preview / Validate
  ├─ 集群内：所有 standard 组件加入平台网 + network alias（与是否 expose 无关）
  ├─ access=local  → 业务 compose ports: "127.0.0.1:{listen}:{container}"；端口冲突校验
  ├─ access=public + http → 现有 Traefik HTTP labels（Host / web|websecure / tls_mode）
  └─ access=public + tcp  → 同步 Gateway TCP entryPoints + gateway publish + 业务 TCP labels

Gateway UI（只读）
  聚合「当前已部署 Service × Version.Expose」→ 展示出口与内网名；无 CRUD 建路由
```

```text
可达矩阵
  app2 Spring → app1 redis     集群内: {app}-{component}:{container_port}
  本机 redis-cli               local ports 或 public 明文 TCP
  浏览器                        public HTTP http://{app}.{base_domain} （tls_mode=none 时 :80）
```

## Design decisions / 设计决策

| ID | 决策 | 说明 |
|----|------|------|
| D-SimpleRuntime | **单 app 单运行套** | 同一 `application` **不允许**同时存在多套活跃部署（多 `instance_key` / 多 env 并发跑同一 app **不支持**）；再部署 = 替换该 app 当前运行，而非并行第二套 |
| D-EnvScope | **弱化多环境** | 本期 **无**「local+prod 同机双开 + 一套网关」产品目标；Environment 实体可保留（部署仍可选 env 行），但 **内网名/容器名/互访不编码 env**；多环境并行与单 Gateway 的端口/域名冲突由产品明确 **不做** |
| D-Name | 内网 DNS 与容器名 **`{app_code}-{component}`** | 规范化小写；与 compose service 可再设 container_name/alias 同此串；**不**含 env、instance |
| D-SoT | Expose 为出口 SoT | Gateway 不维护可写路由表 |
| D-Access | `access ∈ {local, public}` | 有 expose 行时必填；migration 默认 `public` |
| D-Listen | `listen_port` 可选 | 空/0 → `container_port`；冲突失败 |
| D-Local | local = loopback **ports 注入业务组件** | 不经 Traefik；HTTP/TCP 同一模型 |
| D-PublicHTTP | 保持现有 HTTP labels | Host=`{app_code}.{base_domain}`；entrypoint=`default_entrypoint`；tls 跟 `gateway.tls_mode` |
| D-PublicTCP | Gateway 动态 TCP entryPoint | 名 `tcp{listen}`，`address: ":{listen}"`；gateway publish；业务 **仅 labels** |
| D-TCPPlain | 明文 TCP：`HostSNI(\`*\`)` | 同 listen 全局仅一条 public 明文 TCP |
| D-TCPTLS | TLS 开：`HostSNI(\`{app}.{base}\`)` + 终结 | 同 listen 不同 app Host 允许（SNI） |
| D-TLSSource | 跟 Gateway.tls_mode | 一期无 expose 级 tls 开关 |
| D-NoTCPEntrypointField | **删除** `gateway_config.tcp_entrypoint` | HTTP 保留 `default_entrypoint` |
| D-Net | **所有** standard 组件 join 平台网 `traefik` | 无 expose 也 join |
| D-Domain | public 域名 app 级 | `{app_code}.{base_domain}` |
| D-GWSync | public TCP 部署前 reconcile Gateway | 端口集变化则重编译/滚动 Gateway |
| D-Display | 网关页只读聚合 | 见 Interfaces |
| D-DevHTTP | `tls_mode=none` 可测 | 不强制 HTTPS |
| D-Casing | acronym 规范 | `RestApiUrl`、`TLSMode` 等 |

### Requirement Open questions 闭合

| # | 结论 |
|---|------|
| Q1 HTTP local | 与 TCP local 相同：组件 loopback ports |
| Q2 集群内 DNS / 容器名 | **统一 `{app_code}-{component}`**；不含量 env/instance（见 D-SimpleRuntime / D-Name） |
| Q3 public 域名 | **`{app_code}.{base_domain}`**；与内网名两套 |
| Q4 TCP TLS | 跟 Gateway.tls_mode |
| Q5 listen_port | 可选，默认 container_port |
| Q6 展示来源 | DB 活跃 Service × Version × Expose |
| Q7 网络 | 平台网 `traefik`；全组件 join + alias=`{app}-{component}` |
| Q8 多套/多环境 | **不做**并行多 instance、不做多环境同机双开产品路径 |

## Data model / 数据

### `version_expose`（表名以现网为准，如 `expose`）

| 列 | 类型 | 说明 |
|----|------|------|
| …既有 | | component_name, protocol, container_port, path_prefix |
| **access** | TEXT NOT NULL DEFAULT `'public'` | `local` \| `public` |
| **listen_port** | INT NULL | 空 = 使用 container_port |

新 migration（sqlite+mysql 成对）；**不**改已执行文件。默认 `public`：现有「有 expose 即生成出口」行为平滑到 access 模型。

### `gateway_config`

| 变更 | 说明 |
|------|------|
| **DROP** `tcp_entrypoint` | 不再使用 |
| 保留 | rest_api_url, base_domain, image, default_entrypoint, tls_mode |

### 运行时不落库（由部署推导）

- Gateway TCP entryPoint 集合 = 所有 **status 可用/运行中** Service 的 Version 上 `access=public & protocol=tcp` 的 listen 端口并集（再加本次部署将产生的端口）。  
- 网关展示 DTO 同聚合，不另建 CRUD 表。

## Render 规则

### 公共函数

- `effectiveListen(e) = e.listen_port > 0 ? e.listen_port : e.container_port`  
- `publicHost(app, gw) = "{app.code}.{gw.base_domain}"`（小写）  
- `runtimeName(app, component) = normalize("{app.code}-{component}")`  // 内网 DNS 与 container_name  
- `platformNetwork = traefik`（external: true，name 与现常量一致）

### 0) 单运行套约束（D-SimpleRuntime）

- 部署 standard app：若该 `application_id` 已有非终态/运行中 Service，**替换**该绑定（更新 version / 原地滚动），**禁止**再开第二 `instance_key` 并行。  
- API 仍可接受 `instance_key`，一期 **固定/强制 `default`**（非 default 拒绝或忽略并文档化——Plan 选 **拒绝非 default**）。  
- Environment：UI/文档按 **当前唯一工作环境** 使用；不承诺多 env 下同一 app 双活。  
- 工作目录/compose project 名可仍用现实现字符串（含 env）作隔离目录，**不影响** 集群内解析名。

### 1) 网络（所有 standard 组件）

```yaml
# 每个 component service
container_name: <runtimeName>   # {app}-{component}
networks:
  default: {}
  traefik:
    aliases:
      - <runtimeName>
```

顶层：

```yaml
networks:
  traefik:
    external: true
    name: traefik
```

**无 expose 也 join**（D-Net）。Spring 连 Redis：`host={app}-redis`，`port=容器端口`。

### 2) access=local（http 或 tcp）

对 **expose.component** 对应 service 追加：

```yaml
ports:
  - "127.0.0.1:{effectiveListen}:{container_port}"
```

- **不**写 Traefik labels（该 expose）。  
- 部署前冲突检测：宿主机视角逻辑冲突表（见 Validation）——同 `127.0.0.1:listen` 已被其它 Service 的 local expose 占用 → 失败。  
- 同一 Version 内重复 local listen → 校验失败。

### 3) access=public + http

保持现逻辑（迁移后读 Gateway）：

- labels：`Host(\`{publicHost}\`)`、path、`entrypoints={default_entrypoint}`、`loadbalancer.server.port=container_port`  
- `tls_mode`：none / tls / letsencrypt 同现  
- 组件 join 平台网；**不**因 public http 写 host ports  

### 4) access=public + tcp

**业务 compose**（该 component）：

```text
traefik.enable=true
traefik.tcp.routers.{router}.rule=HostSNI(`*`)           # tls_mode=none
# 或 HostSNI(`{publicHost}`)                             # tls_mode=tls|letsencrypt
traefik.tcp.routers.{router}.entrypoints=tcp{listen}
traefik.tcp.services.{router}.loadbalancer.server.port={container_port}
traefik.tcp.routers.{router}.service={router}
# tls_mode≠none:
traefik.tcp.routers.{router}.tls=true
# letsencrypt:
traefik.tcp.routers.{router}.tls.certresolver=letsencrypt
```

- **禁止** 业务 `ports` 映射该 public TCP。  
- `router` 命名沿用现规范（含 app/env/instance/component/protocol 防撞）。

**Gateway static（compile 生成 traefik.yml 片段）**：

```yaml
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
  tcp6379:                    # 示例：由聚合 listen 生成
    address: ":6379"
  # …每个所需 listen 一个 entryPoint
```

- Gateway 组件 `ports`：至少 `80:80`、`443:443`，以及每个 public TCP listen 的 `{p}:{p}`（一期 public TCP **不**绑 127.0.0.1，与 local 区分）。  
- **ACME**：若 `tls_mode=letsencrypt`，static 含 certResolver（与 HTTP 共用）；无则可不写。具体 yaml 与现 HTTP LE 路径对齐（Plan 可引用现有片段）。

### 5) Gateway reconcile（D-GWSync）

当 Deploy **standard** 且存在 `public+tcp` expose，或 Undeploy/变更导致 public TCP 端口集变化：

1. 计算目标 listen 集合 S。  
2. 若与当前 Gateway 已编译 entryPoints 的 TCP 集合不同 → `CompileGatewayToVersion` 写入新 static + ports → **部署 Gateway**（失败则中止本次业务部署或标记错误，Plan 定事务边界）。  
3. 再渲染/部署业务应用。

Gateway 自身 Deploy 也按当前聚合 S 编译（避免手工丢端口）。

## Validation / 冲突

| 条件 | 结果 |
|------|------|
| access 非法 | 拒绝保存 Version |
| public http/tcp 无 Gateway / 无 base_domain | 预览与部署失败 |
| public 明文 TCP：全局（单机平台）已有其它 Service 占用同 `effectiveListen` | 失败 |
| public TLS TCP：同 listen + 同 publicHost 重复 | 失败；同 listen 不同 app Host 允许 |
| local：同 `127.0.0.1:listen` 已被占用 | 失败 |
| local 与 public TCP 同 listen | **允许**（bind 地址不同：loopback vs 0.0.0.0）——仍建议文档提示；若实现困难可改为冲突，Plan 二选一，**默认允许** |
| protocol=tcp 且 path_prefix 非空 | 拒绝 |

冲突数据来源：DB 中非终态失败的活跃 Service + 其 Version exposes（与展示同源）。

## Interfaces / 契约

### Proto / API

**VersionExpose**（及 create/update 嵌入结构）：

```text
string access = N;              // "local" | "public"
optional int32 listen_port = M; // 0/omit = default container_port
```

**Gateway**：移除 `tcp_entrypoint` 字段（create/update/resp）。

**Gateway 只读展示**（新 RPC 或挂在 Gateway get 扩展，Plan 选一）：

```text
message GatewayExposureItem {
  string application_id;
  string application_code;
  string component_name;
  string protocol;       // http|tcp
  string access;         // local|public
  int32 container_port;
  int32 listen_port;
  string public_host;    // 空 if local；{app}.{base_domain}
  string internal_dns;   // {app}-{component}
  string client_hint;    // "127.0.0.1:6379" / "myapp.lvh.me:6379" / "myapp-redis:6379"
}
repeated GatewayExposureItem exposures = …;
```

无 create/update exposure API。

### 前端

- Version 编辑 Expose：协议、端口、**access**、可选 listen_port、HTTP path。  
- Gateway：去掉 TCP 入口下拉；增加 **只读** 暴露列表（及内网名）。  
- 应用/部署结果：可展示 `client_hint` / internal_dns（最小：Gateway 页必达，应用详情可选）。

## Affected components / 影响面

| 区域 | 变更 |
|------|------|
| migration | expose + access/listen_port；gateway drop tcp_entrypoint |
| model/dto/proto/mapper | 同上 |
| `compose_renderer` | 网络全量 join+alias；local ports；public TCP labels；去掉 tcp_entrypoint |
| `gateway_compile` | 动态 TCP entryPoints + ports；可选 ACME 块 |
| deploy usecase | Gateway reconcile；冲突校验 |
| Gateway/Version UI + i18n | 见上 |
| 测试 | compose/gateway/deploy 单测与集成 |

## Alternatives / 备选（未采纳）

| 方案 | 未采纳原因 |
|------|------------|
| Gateway CRUD 路由表 | 用户否定 |
| public TCP 也写业务 0.0.0.0 ports | 与网关出口模型冲突 |
| 集群内必须经 Traefik | 多余跳数；Redis/MySQL 不适合 |
| TCP 继续挂 web/websecure | 端口语义错误 |
| 开发强制 HTTPS/LE | 与 lvh.me 开发路径冲突 |

## Technical questions（留给 Plan，不阻 Spec 接受）

1. Gateway reconcile 与业务 Deploy 的事务/顺序失败回滚策略。  
2. certResolver 静态片段是否已在 compile 中存在，缺则最小补齐。  
3. `normalize` 字符集（仅 `[a-z0-9-]`）。  
4. local 与 public 同 listen「默认允许」的实现成本若过高，Plan 可收紧为冲突。

## Risks

1. 每次新 public TCP 端口触发 Gateway 滚动 → 短暂影响入口。  
2. 全量 join 平台网扩大攻击面（文档：平台网信任边界）。  
3. 单运行套与历史多 instance API 字段并存时的提示/拒绝策略。  
4. 删除 tcp_entrypoint 与旧 TCP-on-80 行为不兼容（已接受）。  
5. 日后真·多环境需另开需求（多 Gateway 或隔离），不在本 Spec。

## User review notes

- 2026-07-23：Requirement Accepted → 撰写 Spec Draft。  
- 2026-07-23：用户要求 **简化**——禁止 app 同时多套部署；弱化/不做多环境；内网名与容器名统一 **`{app}-{component}`**（去掉 env/instance）。  
- 2026-07-23：用户「接受，进入 plan」→ Spec **Accepted**，进入 Plan。
