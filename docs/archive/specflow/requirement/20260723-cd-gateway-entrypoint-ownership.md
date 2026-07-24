# CD：default_entrypoint 归属 Gateway
最后修改时间: 2026-07-23 11:30:00

Review status: Accepted

Flow mode: light

Parent / 关联：
- E6 `docs/requirement/20260722-cd-gateway-domain-config-e6.md`（`base_domain` 已迁 Gateway；当时 **保留** Env 上 entrypoint/tls）
- 清理任务 `docs/requirement/20260723-cd-post-e6-cleanup.md`（工程死键清理，**不含**本模型修正）
- Spec 历史：`docs/spec/20260722-cd-environment-ingress-policy.md`（entrypoint 原挂 Environment）

## Background

E6 已把 **域名根**（`base_domain`）与 rest 控制面 URL 归到 **Gateway config**，Host 为 `{app_code}.{gateway.base_domain}`。

当前 **`environment.default_entrypoint` / `tcp_entrypoint` / `tls_mode`** 仍挂在 Environment：compose label 的 `entrypoints=` 与 TLS 相关 labels 从 Env 读，而 Traefik **entryPoints 定义**实际由 Gateway compile 的 static config 声明（目前仅 **`web` / `websecure`**）。

这些字段**直接决定** Traefik Docker labels 如何生成（HTTP/TCP `entrypoints=`、`tls` / `certresolver`），属于 **网关入口面能力**，不是 Environment（部署目标）语义。

这导致：

1. **所有权错位**：entrypoint / TLS 策略与网关 static entryPoints 应对齐，却配在 Environment。  
2. **配置分裂**：运维在 Environment 改入口名，与 Gateway 静态配置无强制对齐。  
3. E6 Spec 当时「仍可留在 Environment」的 entrypoint 裁定与「网关拥有路由/入口面」不一致；用户现裁定 **应属 Gateway**。

项目约束：活跃开发期 **不做历史兼容**；**不修改已执行 migration 文件**（`000010`/`000011` 只读，用新 migration 改 schema）。

## Goal

1. **入口/TLS 策略 SoT 全部迁到 Gateway config**：`default_entrypoint`、`tcp_entrypoint`、`tls_mode`（与 `rest_api_url`、`base_domain` 同表/同 API 面）。  
2. compose / Expose label 渲染与 gateway dashboard labels：**只从 Gateway 读**上述字段，**不再**读 Environment 对应列。  
3. Environment API / UI / proto：**去掉** `default_entrypoint` / `tcp_entrypoint` / `tls_mode`；Environment 仅保留部署目标元数据（code/name/description 等）。  
4. Gateway API / UI / proto：**暴露**上述字段（创建/更新/响应）。  
5. **创建默认**：`default_entrypoint=web`；`tls_mode=none`；`tcp_entrypoint` 可空（空则 TCP labels 回落 `default_entrypoint`，语义与现实现一致，但读源改为 Gateway）。  
6. **枚举约束（当前产品面）**：`default_entrypoint` 与非空的 `tcp_entrypoint` **仅允许** `web` | `websecure`（与 compile static entryPoints 一致）。非法值创建/更新 Gateway 拒绝；渲染侧同样不接受未知名。  
7. 新 migration：`gateway_config` 增列，`environment` **删列**；无 dual-read、无旧列兼容层。  
8. 检查：`task check`；测试：`task test`。

## Non-goal

1. 不实现 F1 自定义 PEM、external Traefik、多 gateway（F3）。  
2. **不**在本任务产品化任意自定义 entryPoints 名或端口矩阵；集合固定为 compile 已声明的 `web`/`websecure`。后续若扩展 entryPoints，再改枚举与 compile 同源。  
3. 不改已执行 migration 文件内容。  
4. 不做旧库数据迁移兼容或双轨读 Env+Gateway。  
5. 不顺带重构无关 CD 模块。

## User scenarios

1. 运维在 **网关** 创建/编辑时配置默认 HTTP entrypoint（默认 `web`，可选 `websecure`）、可选 TCP entrypoint、TLS 模式；部署 standard 应用时 Docker labels 使用这些值。  
2. **环境** 页只管理 code/name/description；**不再**出现入口/TLS 字段。  
3. 无 Gateway、或 Gateway 入口/TLS 字段无效时，需要 labels 的 Expose 部署 **明确失败**（不静默回落到 Env 旧字段）。

## Acceptance

1. `gateway_config` 含 **`default_entrypoint`**、**`tcp_entrypoint`**（可空）、**`tls_mode`**；Gateway create/update/list/get 契约同步。  
2. `environment` 表与 Environment create/update/list/get **无** 上述三列/字段。  
3. `compose_renderer` / 相关校验与单测：labels 的 entrypoint 与 TLS **仅**来自 Gateway；错误文案不再声称 environment ingress policy 字段。  
4. Gateway 校验：`default_entrypoint` 必填且 ∈ {`web`,`websecure`}（创建缺省 `web`）；非空 `tcp_entrypoint` 同集合；`tls_mode` ∈ {`none`,`letsencrypt`,`tls`}（缺省 `none`）。  
5. 前端：Gateway 表单可编辑上述字段（entrypoint 用固定选项，不用自由文本）；Environment 表单与列表移除入口/TLS。  
6. 新 migration（sqlite + mysql 成对）；**不**改 `000010`/`000011` 已执行文件。  
7. **无** 读 Env 列作 fallback 的兼容路径。  
8. `task check` 与 `task test` 通过。

## Open questions

**暂无需要用户确认的未决事项。**

（Q1–Q3 已由用户 2026-07-23 裁定，见 Decisions。）

## Decisions

1. 用户裁定（2026-07-23）：**入口面属 Gateway** → `default_entrypoint`（及整套相关策略）迁 **Gateway config**。  
2. **`tcp_entrypoint` / `tls_mode` 一并迁 Gateway**；Environment 不再持有任何 ingress entrypoint/TLS 列。  
3. **创建默认** `default_entrypoint=web`（与 seed/compile 一致）；`tls_mode` 默认 `none`。  
4. **当前仅 `web` / `websecure`**：二者与 Gateway compile 的 Traefik entryPoints 对齐，并 **决定** Traefik Docker labels 如何生成；保存与渲染均限制在此集合（非任意字符串）。  
5. Flow：**light**（Requirement → Implementation → Verification）。  
6. **不做历史兼容**；开发库可重建。  
7. 验证命令：`task check` / `task test`。  
8. 与 post-e6 cleanup **正交**：cleanup 可独立合并；本任务为模型修正，可另 diff/PR。

## Risk

1. 已有 Environment 上的 entrypoint/tls 配置在删列后丢失 → 符合无兼容；Gateway 默认 `web` + `tls_mode=none` 覆盖常见路径，特殊值需在 Gateway 上重配。  
2. 若未来增加 entryPoints，必须同步改 compile static、枚举校验与 UI 选项（本任务不预留自由文本兼容）。  
3. proto 字段号变更需 `task proto`（或项目等价生成）并前后端同步。  
4. 集成/单测夹具大量写 `Env.DefaultEntrypoint` 等，需改为 Gateway。

## Risk / 假设（实现向）

| 假设 | 说明 |
|------|------|
| A1 | 单 active gateway 解析路径已存在；render 时已有 `GatewayConfig` 入参 |
| A2 | `tcp_entrypoint` 空 → TCP 使用 `default_entrypoint`（逻辑保留，读源改 Gateway） |
| A3 | 默认 `web` / `tls_mode=none` 与现网 seed 行为一致 |
| A4 | `websecure` 作为 HTTP 默认 entrypoint 时，labels 写 `entrypoints=websecure`；TLS 附加 labels 仍由 `tls_mode` 控制（与现 Env 行为同构） |

## User review notes

- 2026-07-23：用户指出 **`environment.default_entrypoint` 应该是网关的能力**。  
- 2026-07-23：用户 **采纳** (1) `tcp_entrypoint`/`tls_mode` 同迁；(2) 默认 `web`；(3) 当前仅 **`web`/`websecure`**，它们决定 Traefik Docker label 生成。  
- 2026-07-23：用户「开始实现」→ Requirement **Accepted**；进入 Implementation。
