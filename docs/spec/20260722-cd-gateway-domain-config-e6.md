# CD E6：网关领域（Gateway domain）规格
最后修改时间: 2026-07-22 18:51:11

Review status: Accepted

Flow mode: strict

## Requirement basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260722-cd-gateway-domain-config-e6.md` | **Accepted** |
| `docs/requirement/20260721-cd-application-version-cadence.md` | Accepted；本 Spec = **扩展 E6** |
| `docs/requirement/20260722-cd-application-kind-gateway.md` | R3；`kind=gateway` 身份 |
| `docs/spec/20260722-cd-gateway-component-mount-e5.md` | E5；挂载 / rest 机制 / 单 gateway |
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | R2；Environment 接入策略（E6 要**剥离**错挂在 Env 上的网关配置） |

**硬约束（Requirement + 本轮用户裁定）**：

1. **Gateway = `Application(kind=gateway)`**（身份仍在 Application 体系）。  
2. Gateway **自有配置独立管理**：**改表**（独立配置存储）；**禁止**在 `application` 上堆 `kind=standard` 不用的列。  
3. 容器工作负载仍经 **Version** 部署（E5 挂载能力保留）。  
4. **导航增加 Gateway 入口**；应用主列表默认 standard。  
5. **rest 管理面 URL、与网关相关的域名配置从 Gateway 读取**；从 **Environment** 拿掉属于 Gateway 的配置；**全局** `traefik.api_url` / `traefik.domain_suffix` 等 **移除**。  
6. 仅 managed；无 external；不兼容不迁旧数据。  
7. F1–F7 / File 写路由 / `code==traefik` 魔法：不做。

## 「Intent」是什么（术语澄清）

上一版 Spec 里的 **Intent / GatewayIntent** 是草稿用词，指：

> **用户在网关领域 UI 上编辑的那组结构化配置字段**（镜像、端口、rest、docker provider…），  
> 相对于「直接改 Version 挂载行」的另一种编辑面。

**不是**独立运行时对象，也不是第二个 Deploy 引擎。

| 易混说法 | 正确理解 |
|----------|----------|
| Intent | **已废弃作正式名**；改称 **Gateway 自有配置（Gateway config）** |
| Gateway config | 落在 **Gateway 配置表**（见 D2）的产品 SoT |
| Version | 部署规格载体（可含 mounts）；可由配置 **编译生成/更新**，也可人工 E5 编辑 |
| Application | `kind=gateway` 的身份行；**不**塞网关专用列 |

下文统一用 **Gateway config**，不再使用 Intent。

## Scope / 范围

| 纳入本期 | 不纳入本期 |
|----------|------------|
| Gateway = Application(kind=gateway) + **独立配置表** | external |
| 导航 **Gateway** 入口 + 应用列表默认 standard | F1 PEM / F2 鉴权 / F3 多 gateway / F5 dashboard |
| rest `api_url`、网关域名相关配置 **归属 Gateway** | 全局 `traefik.api_url` / `domain_suffix` 保留 |
| Environment **剥离**误属网关的配置 | 旧数据迁移 |
| Version 完整挂载保留；配置→Version 编译（managed） | 在 application 表加网关专用列 |
| 路由 PUT 目标从 **当前 Gateway config** 解析 | 第二套非 Version 部署引擎 |

## Overview

```text
[导航]
  应用     → Application list (kind=standard 默认)
  网关     → Gateway list (Application where kind=gateway)
                + Gateway config 编辑
                + Version / Deploy（工作负载）

[配置分层]
  application          身份：id, project_id, code, kind=gateway, name…
  gateway_config       自有配置：rest_api_url, domain…, 部署意图字段…  (独立表)
  version/component    部署规格：image, mounts, env, ports… (E5)

[读路径]
  平台写 rest  →  api_url  := Gateway config（当前生效网关）
  域名相关     →  从 Gateway config（及 R2 中仍属 Environment 的业务策略）解析
  全局 yaml    →  删除 traefik.api_url / domain_suffix（见 D5）

[部署]
  Gateway config ──compile──► Version
  Deploy → Render(kind=gateway) + materialize + runner
```

## Design decisions

### D1 — Gateway 身份

| 项 | 规则 |
|----|------|
| 定义 | **Gateway ≔ `Application` 且 `kind=gateway`** |
| 创建 | 经 **网关** UI/API 创建 Application，`kind` 固定 `gateway` |
| 禁止 | 无 Application 的纯配置网关；在 standard Application 上挂 Gateway config |
| 数量 | 与 E5 一致：同时 **active** 的 gateway Service 全局至多一个（部署约束）；配置行可按 Project 策略 Spec 钉（推荐 Project 下可有多个 gateway App，但 active 仍 D9） |

### D2 — 改表：Gateway 自有配置（独立于 Application 列）

**问题**：网关有 rest URL、域名、镜像意图等字段；若加在 `application` 上，standard 行全是 NULL，模型脏。

**决策**：

| 存储 | 内容 |
|------|------|
| `application` | **仅**通用身份字段（与 standard 同形）：project、code、name、kind、时间戳… |
| **`gateway_config`（新表，名 Plan 可微调）** | **仅** `kind=gateway` 使用的配置；`application_id` **PK 或 UNIQUE** 1:1 |
| `version` / `version_component` | 部署规格（E5）；**不**把 rest 控制面 URL 只存在 Version 文本里当平台读源 |

**关系**：

```text
application (kind=gateway) 1 ── 1 gateway_config
application               1 ── * version
```

- 删除 gateway Application → 级联删 `gateway_config`（Plan 钉 FK）。  
- **禁止** `application.rest_api_url` 这类 kind 条件列。

#### D2.1 Gateway config 字段分组（逻辑；列名 Plan/migration）

**A. 控制面 / 平台读（必须来自 Gateway，非全局）**

| 逻辑字段 | 说明 |
|----------|------|
| `rest_api_url` | Traefik `providers.rest` 基址；平台 **PUT/GET 唯一来源** |
| （可选后续）鉴权 | F2，本期可无列 |

**B. 域名 / 入口相关（从 Env/全局迁到 Gateway 的部分）**

| 逻辑字段 | 说明 |
|----------|------|
| `base_domain` | **原 Environment.base_domain + 全局 domain_suffix 职责**；Host 推导根域；**Environment 不再存 base_domain** |
| 其它网关自有域名项 | 仅挂 Gateway config |

**C. 部署意图（编译进 Version 的用户向配置）**

| 逻辑字段 | 说明 | 编译去向（示意） |
|----------|------|------------------|
| `image` | Traefik 镜像 | Component.image |
| `http_port` / `https_port` | 数据面端口 | Component.ports |
| `rest_listen` / 管理口策略 | 容器内启用 rest | content/command |
| `docker_provider` | docker provider 开关与参数 | content/command |
| `docker_network` | provider 网络名 | 与 E5 top-level network 对齐 |
| `mount_docker_sock` | special mount | mounts |
| `log_level` | 日志 | env/command |
| `acme_enabled` | acme 文件 seed | logical file mount |

**D. 非配置**

- 运行状态、容器 ID → Service/Container，不进 gateway_config。

### D3 — Version 与编译

| 项 | 规则 |
|----|------|
| 部署载体 | 仍 **Version**；Deploy 只认 Version 规格 + RuntimeConfig |
| Gateway config C 组 | **保存时 compile** 写入/更新 unpublished Version（推荐） |
| E5 挂载 UI | **保留**；可与 compile 并存（upsert 规则：config 管理的 mount 按 target 键覆盖，其余保留） |
| 平台读 rest URL | **只读 gateway_config.rest_api_url**，**不**从 Version 文本解析 |

### D4 — Environment：拿掉属于 Gateway 的配置

E6 用户裁定（2026-07-22）：**`Environment.base_domain` 移除，由 Gateway 负责。**

| 仍可留在 Environment | 必须离开 Environment |
|----------------------|----------------------|
| `default_entrypoint` / `tcp_entrypoint` / `tls_mode` | **`base_domain`、`domain_template` 列删除** |
| name/code/description | 控制面 URL、全局 domain_suffix、任何 rest 目标 |

**实现动作（Plan）**：

1. migration 删除 `environment.base_domain` 与 `domain_template`；API/UI/proto 同步。  
2. Render Host：固定 `{app_code}.{gateway.base_domain}`。  
3. **不迁移**旧数据；开发库重建。  
4. **控制面 URL 绝不在 Env。**

### D5 — 移除全局 Traefik 配置

| 现状 | E6 |
|------|-----|
| `configs` / env：`traefik.api_url` | **删除**；改为 Gateway config `rest_api_url` |
| `traefik.domain_suffix` | **删除**；改为 Gateway config（或仅业务 Env，见 D4 盘点） |
| settings 键 `traefik__domain_suffix` 等 | **删除**产品路径 |
| `traefik.cert_dir` | F1/证书路径：Plan 决定迁 Gateway、保留 data 布局常量、或后续 F1；**不得**再充当 api_url 来源 |
| `RouteManager` 读 `cfg.Traefik.APIURL` | 改为 **解析当前 Gateway** 的 `rest_api_url`（无 gateway / URL 空 → 明确错误，不静默） |

**当前生效 Gateway**：与 E5 单 active gateway 对齐 — 存在 running/deploying 的 gateway Service 时用其 Application 的 config；若无运行实例则用「Project 默认/唯一 gateway」规则（Plan 钉；推荐 **显式 default 标记** 或 **唯一 gateway Application**）。

### D6 — 导航与 UI

| 面 | 行为 |
|----|------|
| 侧栏 | 新增 **网关 / Gateway** 入口（与「应用」并列） |
| 应用 | 列表默认 `kind=standard`；不把 gateway 当常规应用主列表 |
| 网关 | 列表 `kind=gateway`；详情：**Gateway config 表单** + Version/Deploy/挂载（E5） |
| 路由 | 写 rest 失败时错误信息可指向「未配置 Gateway rest_api_url」 |

### D7 — 单 gateway 与 rest 写路径

| 项 | 规则 |
|----|------|
| 全量 PUT | 仍 E5 `providers.rest` |
| URL 来源 | **仅** Gateway config |
| 无有效 URL | Sync/Apply **失败**，不回退全局配置（全局已删） |

### D8 — 兼容

无迁移、无双轨读全局+Gateway、无 application 脏列兼容层。

### D9 — 交付批

| 批 | 内容 |
|----|------|
| **E6-A** | 表 `gateway_config`；导航 Gateway；list 过滤；CRUD config 骨架 |
| **E6-B** | 去掉全局 api_url/domain_suffix；RouteManager/域名解析改读 Gateway；Env 剥离 |
| **E6-C** | Gateway config 部署意图字段 UI + compile → Version；Deploy 闭环 |
| **E6-D** | Version 挂载路径回归（E5 不回归）；单测与文档 |

可 A+B 优先（配置归属正确），C 补领域部署表单。

## Interfaces（逻辑）

```text
GET    /api/cd/gateway?project_id=
POST   /api/cd/gateway                 # 创建 Application(kind=gateway)+gateway_config
GET    /api/cd/gateway/{application_id}
PUT    /api/cd/gateway/{application_id}  # 更新 gateway_config（+ 可选 compile）
# Deploy/Stop/Version 可复用 application 既有路由，或网关命名空间代理同一 usecase

GET    /api/cd/application?kind=standard|gateway
```

- 路由单数风格与项目约定一致（`/api/cd/gateway`）。  
- Body DTO **proto 生成**。

## Data shape（示意 · 非最终 SQL）

```text
gateway_config
  application_id   PK/FK → application.id
  rest_api_url     TEXT NOT NULL  -- 或允许空但禁止作为 PUT 目标直到填写
  domain_suffix    TEXT          -- 若承担原全局后缀
  image            TEXT
  … 其它 C 组字段或 JSON 文档列（若用 JSON：仅 config 表内，不上 application）
  updated_at
```

若 C 组字段多，允许 `spec_json` **仅在 gateway_config**；**禁止** application 级自由 JSON 当 standard/gateway 混用袋。

## Risks

1. R2 Env 字段与 Gateway 域名边界需盘点，避免 standard Host 推导断裂。  
2. 去掉全局 api_url 后本地 dev 必须先建 Gateway config。  
3. compile 与手改 Version 冲突 — upsert 规则 + 保存即 compile。  
4. 「当前 Gateway」解析在多 Project 时要钉死（E5 单 active 全局 vs per-project）。

## Alternatives（否决）

| 方案 | 否决 |
|------|------|
| application 表加 rest_api_url 等 | standard 脏列；用户明确否决 |
| 配置继续全局 yaml | 用户明确移除 |
| rest URL 放 Environment | 属于 Gateway，用户明确拿掉 |
| 不建表、只包一层 UI | 无法承载自有配置 SoT |

## User review notes

- 2026-07-22：用户纠正 — **要改表**；Gateway=Application(kind=gateway)；**自有配置独立**；导航加 Gateway；**rest/域名从 Gateway 取**；Env 剥离；**全局配置移除**。  
- 澄清：**Intent = 旧草稿对「领域配置字段」的称呼**，正式名 **Gateway config**。  
- 2026-07-22：用户钉死 **Environment.base_domain 移除，由 Gateway 负责**；实现步骤 OK；**开始 Plan** → Spec **Accepted**。  
- 2026-07-22：用户钉死 **Env 不留 domain_template**；**开始实现**。
