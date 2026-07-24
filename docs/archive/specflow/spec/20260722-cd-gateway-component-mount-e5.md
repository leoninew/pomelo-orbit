# CD E5：Component 全规格、部署运行时与网关路由
最后修改时间: 2026-07-22 17:38:46

Review status: Accepted

Flow mode: strict

## Scope / 范围

本文件为**实现参考规格**：定义模型、行为与边界。不记录产品背景、历史迭代或仓库叙事。

| 纳入本期 | 不纳入本期（见「后续任务」） |
|----------|------------------------------|
| VersionComponent 结构化挂载 / 环境变量表单与 API | 自定义 PEM 证书推送与挂载对齐 |
| 部署 RuntimeConfig（占位 → 必填/默认） | 同时部署多个 gateway |
| 全 kind 共用挂载解析；gateway 无特殊路由目录树 | rest 管理面 mTLS / BasicAuth |
| gateway top-level network 合并 | HTTP Provider / KV 替换 rest |
| 平台动态路由：`providers.rest` 全量 PUT | dashboard / 容器内访问能力 |
| **全局至多一个** 已部署 gateway（控制面 + 80/443 占用） | 多 gateway 端口编排与 labels 目标选择 |

## Constraints / 硬约束

1. Version 规格用**结构化字段 + 表单**；禁止以巨型自由 JSON 作为 Version 主 SoT。  
2. RuntimeConfig 属**部署面**，不回写 Version；废止以 `env_file: .env` 为部署配置主路径。  
3. 挂载解析与表单能力 **所有 Application.kind 共用**。  
4. top-level network 仅当 `kind=gateway` 时由 **kind × Environment** 合并渲染。  
5. 平台动态路由写入面 = Traefik **`providers.rest`**（`PUT` 全量 JSON）；废除基于 dynamic 目录写文件管理路由。  
6. **不为** gateway 做特殊部署目录结构（不为接平台路由强制 dynamic 子树）。  
7. 路由快照由**平台**结构化维护并 PUT；不进 Version 表单。  
8. Traefik 官方 `api@internal` **只读**；不得与 rest 写路径混淆。  
9. File 与 rest **不得**双轨并行写平台路由。  
10. **不支持同时部署多个 gateway**（见 D9）。

## Overview

```text
[规格]
  Version / VersionComponent
    mounts[] · env[] · ports · command（结构化）

[部署]
  RuntimeConfig + Render(kind)
    standard → services +（有 Expose 时）Docker labels
    gateway  → services + top-level network
    全 kind 同一挂载解析

[路由控制面 · 单一 gateway]
  平台 Route 集合 → route snapshot
  PUT {api_url}/api/providers/rest   # 全量替换 @rest
  GET {api_url}/api/http/routers     # 观测
```

主线（实现可切批，语义同属本期）：

| 线 | 内容 |
|----|------|
| A | 挂载 / 环境变量模型 + Render + UI |
| B | RuntimeConfig + Deploy |
| C | 路由 rest 全量 PUT；废除 File 写路由 |
| D | gateway 网络合并；单 gateway 部署约束 |

## Design decisions

### D1 — 挂载模型

**VersionComponent.mounts**：JSON 数组（或等价 typed 列表），元素固定 schema；UI 表单编辑。

| 字段 | 类型 | 说明 |
|------|------|------|
| `source_type` | enum | `logical` \| `volume` \| `special` |
| `source` | string | `logical`：相对应用数据根；`volume`：命名卷名；`special`：白名单 token |
| `target` | string | 容器内绝对路径，必填 |
| `read_only` | bool | 默认 false |
| `content` | string? | **仅** logical + 文件型；文件正文（UTF-8 文本）；可选 |
| `content_mode` | enum? | `seed`（默认）\| `sync`；仅当允许 content 时有意义 |

**special 白名单（初版）**：

| token | 渲染 |
|-------|------|
| `docker.sock` | 源 `/var/run/docker.sock`（默认 `read_only: true`，可覆盖） |

**content 规则**：

| 项 | 规则 |
|----|------|
| 允许 | `source_type=logical` 且源/目标按文件后缀启发式判定为**文件型** |
| 禁止 | 目录型 logical、`volume`、`special` 带 content → 校验失败 |
| 缺省 content | 物化行为与 E5 初版一致：文件不存在则 touch 空文件 |
| `content_mode=seed` | 文件**不存在**时写入 content（或空文件）；**已存在不覆盖**（acme.json 安全） |
| `content_mode=sync` | 每次 Deploy 用 content **覆盖**写（适合 traefik.yml 规格同步） |
| 体积上限 | 单文件 content ≤ **262144** 字节（256KiB） |
| 与 kind | **无** gateway 自动生成 yml；content 为通用能力 |

**禁止**：

- 以宿主机任意绝对路径作为 Version 唯一真相并保存。  
- 平台强制注入「dynamic 路由目录」挂载项。  
- 平台按 `kind=gateway` 注入 `traefik.yml` 正文（须用户/示例 Version 的 mount content）。

**Render / Materialize**：

- `logical` → 物理路径由应用数据根解析。  
- Preview 与 Deploy **同一解析函数**。  
- 失败明确报错，不静默丢弃。  
- Deploy 前物化：目录 MkdirAll；文件按 content / content_mode 写盘。

存储列可继续使用 `mounts_json`，但内容必须符合上述 schema；UI 不得以整段 JSON 编辑器为唯一入口。

### D2 — 环境变量与 RuntimeConfig

**规格层（Version / VersionComponent）**

- 列表项：`{ key, value }`。  
- `value` 为字面量，或占位：  
  - `${NAME}` — 部署必填  
  - `${NAME:-default}` — 未提供时用 default  

推荐序列化为**数组**（保序）：`[{ "key", "value" }, ...]`。

**部署层（RuntimeConfig）**

- 键 = 扫描 Version 与各 Component env 中的占位名。  
- Deploy 请求携带 `runtime_config: map[string]string`。  
- 缺必填 → 失败，返回缺失键列表。  
- 有默认且未提供 → 用默认。  
- **不**回写 Version。  
- 落库：挂在**本次 Deployment** 记录（options/JSON）；密钥不进 Version。

**Render**：字面量写入 `environment`；占位用 RuntimeConfig 解析结果。  
**废止**：平台部署主路径依赖 compose `env_file: .env` / 工作区固定 `.env`。

### D3 — ports / command

- **ports**：结构化字段 + 表单；容器端口声明。对外暴露意图以 **Expose** 为准（有则写 labels，无则不写）。  
- **command**：字符串或字符串数组；可与挂载/环境变量同期表单。  
- 本期不强制 healthcheck 表单。

### D4 — top-level network（仅 gateway）

```yaml
networks:
  <network_key>:
    name: <resolved_name>
    driver: bridge
```

| 项 | 规则 |
|----|------|
| `resolved_name` | Environment 可配；缺省 **`traefik`** |
| `network_key` | 实现固定一处（`default` 或与 name 相同），避免双 SoT |
| service | gateway 各 service 默认 join 该 network |
| standard | 不注入该 top-level 块 |

网络名与域名后缀（base_domain / domain_suffix）**无关**，禁止混用。

### D5 — 部署目录

| 项 | 规则 |
|----|------|
| 工作区 | gateway 与 standard **同一**策略 |
| 路由目录树 | 平台**不因**动态路由创建/要求 gateway 专用 dynamic 子树 |
| 业务挂载 | 仅来自 VersionComponent.mounts |

### D6 — 平台动态路由（providers.rest）

#### D6.1 契约

| 项 | 值 |
|----|-----|
| 方法 | `PUT` |
| URL | `{api_url}/api/providers/rest`（`api_url` 去尾斜杠） |
| Body | JSON；`Content-Type: application/json` |
| 语义 | **全量替换** rest 命名空间（非 merge 单条） |
| 清空 | `{"http":{"routers":{},"services":{}}}`（`{}` / `{"http":{}}` 不清） |
| 观测 | `GET {api_url}/api/http/routers` 等（只读） |
| Traefik | **≥ 3.6**（3.6.x–3.7.x 已验证 rest 仍可用） |
| 注意 | 配置生效受 `providersThrottleDuration`（默认约 2s）影响；清空后宜再观测确认 |

#### D6.2 快照

- 平台为 **唯一写者**；对 rest PUT **串行化**。  
- 快照 = 全部应生效的平台 Route（结构化字段）组装的 `http.routers` + `http.services`（及本期支持的 tls 字段）。  
- `Sync` / `Revoke` / 对账：重建全集后 **一次 PUT**。  
- 路由/服务名稳定、合法。  
- **禁止**把整包 Traefik 配置当作 Version 字段编辑。

#### D6.3 废除的写入路径

| 废除 |
|------|
| 向 dynamic 目录写 per-route YAML |
| 依赖 File watch / Windows 向容器发 HUP 促路由热更 |
| 以删除 yaml 文件作为撤销语义（改为快照重建 PUT） |
| 配置项「dynamic 路由目录」作为平台写路由 SoT |

#### D6.4 TLS（本期 vs 后续）

| 能力 | 本期 |
|------|------|
| HTTP 路由 | 必须支持 |
| ACME / `tls.certResolver`（如 letsencrypt） | 支持（rest router 字段） |
| **自定义 PEM** 文件写入与容器路径对齐 | **不纳入本期** → 后续任务 **F1** |

#### D6.5 控制面

- `api_url` 来自平台配置（单一地址，与 D9 一致）。  
- gateway 静态配置须启用 `providers.rest`；**E5 期内**平台不在 Version 写 rest 开关魔法（依赖规格/文档或用户 env/command/content）。**一等领域字段与专用渲染 → 扩展 E6**，不在本期静默实现。  
- rest 写入口不得默认暴露公网；管理网 / 本机绑定。  
- rest 鉴权增强 → 后续任务 **F2**。

#### D6.6 与 Docker labels

- **standard + Expose**：Docker labels（provider=docker）。  
- **平台 Route 实体**：rest 快照（provider=rest）。  
- rest 全量替换 **只影响 `@rest`**，不清除 docker labels 路由。

### D7 — UI（行为）

| 能力 | 行为 |
|------|------|
| Version 编辑（未发布） | 表单维护 mounts、env；校验 source_type / 路径 |
| Deploy | 展示占位必填与默认；提交 `runtime_config` |
| Preview | 展示解析后的 volumes / environment；不要求 env_file |

### D8 — 不兼容

- 删除或改写依赖 env_file 主路径、dynamic 目录写路由、gateway 必挂 dynamic 的用例与假设。  
- 无 File + rest 并行写平台路由。

### D9 — 单 gateway（本期强制）

**不支持同时部署多个 gateway。**

原因（产品/运维）：

1. 多个 gateway 争用宿主机 **80/443**（及同类入口端口）会直接冲突。  
2. 若改为「每 gateway 不同端口」，则 **Docker labels 应指向哪一入口 / 哪一实例** 失去稳定答案，standard 暴露与平台路由目标均模糊。

**本期行为**：

| 规则 | 说明 |
|------|------|
| 控制面 | **单一** `api_url` → 唯一 rest 写入口 |
| 部署约束 | 在已有 **运行中的 gateway** Service（或等价）时，拒绝再部署另一个 `kind=gateway`（明确错误） |
| 网络名 | 默认共享名（如 `traefik`）与「单提供者」一致 |
| labels / Route 目标 | 默认指向该唯一 gateway 所在入口与网络语义 |

多 gateway 编排（端口矩阵、labels 目标选择、多 `api_url`）→ 后续任务 **F3**。

## Interfaces

### Mount / EnvVar

```text
Mount {
  source_type: logical | volume | special
  source: string
  target: string
  read_only?: bool
  content?: string              # logical file only
  content_mode?: seed | sync    # default seed
}

EnvVar {
  key: string
  value: string   # literal | ${NAME} | ${NAME:-default}
}
```

### Deploy body（增量）

```text
runtime_config?: map[string]string
```

缺必填占位 → 4xx + 缺失键列表。

### Traefik rest（出站）

```text
PUT {api_url}/api/providers/rest
Content-Type: application/json
Body: { "http": { "routers": {...}, "services": {...}, ... } }
```

## Closed technical choices

| # | 选择 |
|---|------|
| T1 | env 序列化优先 **数组** `[{key,value}]` |
| T2 | RuntimeConfig 挂 **Deployment** |
| T3 | 网络默认名 **`traefik`**（Environment 可覆盖） |
| T4 | 自定义 PEM → **后续 F1**，不本期交付 |
| T5 | rest 失败 → 明确错误，不静默成功 |
| T6 | **单 gateway**；多实例 → **后续 F3** |

## Follow-up tasks / 后续任务

实现与验收跟踪时必须单独开任务（或 cadence 扩展项），**不得静默丢弃**：

| ID | 主题 | 说明 |
|----|------|------|
| **E6** | **网关领域（双形态）** | **Gateway domain**：`managed`（容器 Traefik — 配置面向用户、非 Version UX，部署借 Version 能力）\| `external`（宿主机/自部署 — 只登记 rest）。动态路由仍 rest PUT。Requirement：`docs/requirement/20260722-cd-gateway-domain-config-e6.md`（Draft） |
| **F1** | 自定义证书（PEM） | 平台写证书、容器路径、rest `tls` 引用与挂载策略；与「无强制 gateway 路由目录」兼容 |
| **F2** | rest 管理面鉴权 | mTLS / BasicAuth / 反向代理鉴权；替换仅靠网络隔离 |
| **F3** | 多 gateway | 端口不冲突方案、labels/Route 目标选择、多 `api_url`、部署约束放宽条件 |
| **F4** | 路由 provider 演进 | HTTP Provider / KV 等替换 rest 的适配边界（仍不回流 File 写路由，除非新 ADR） |
| **F5** | dashboard / 访问能力 | gateway dashboard、Service 内容器访问等产品化 |
| **F6** | healthcheck 等规格表单 | Component 健康检查等扩展字段 |
| **F7** | 预期外缺口 | 实现/验收中新发现且超出本 Spec 的项，记入 cadence 后续，不塞进本期静默扩大 |

## Risks

1. rest 全量替换丢路由 → 唯一写者 + 完整快照。  
2. rest 为遗留能力 → 锁定 Traefik 版本下限；F4 预留演进。  
3. 管理面暴露 → 网络隔离；F2 补鉴权。  
4. 单 gateway 约束被绕过 → 部署校验必须硬拦。  
5. 表单范围膨胀 → 实现可切批 A/B/C/D，但不删语义。

## Alternatives rejected

| 方案 | 原因 |
|------|------|
| File + dynamic 目录写路由 | 与挂载/部署耦合；热更脆弱 |
| `PUT/POST /api/http/routers` | 官方 API 只读（405） |
| Version 存整包 Traefik JSON | 违反结构化建模 |
| 本期多 gateway | 80/443 冲突；labels 目标歧义（D9） |

## Implementation order（建议）

1. C — rest 路由 + 废除 File 写路径  
2. D9 — 单 gateway 部署校验  
3. A — 挂载/环境变量 + Render + UI  
4. B — RuntimeConfig  
5. D4 — gateway 网络合并  
6. 清理失效用例  

## User review notes

- Spec 定位为**参考规格**，不写项目背景叙事。  
- 自定义证书、多 gateway、鉴权等 **延后并列入后续任务 F1–F7**。  
- **暂不支持多 gateway**（端口与 labels 目标）。  
- 其余决策 **Accepted**。
