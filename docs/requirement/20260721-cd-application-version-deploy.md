# CD 应用版本化：按版本部署到本机
最后修改时间: 2026-07-22 08:35:55

Review status: Accepted

Flow mode: standard

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`（P2）

Depends on: P0、P1

**后续路线（本 P2 之后，不改写本需求 Accepted 正文范围）**：

| 修订 | 文档 | 与本 P2 关系 |
|------|------|----------------|
| R1 | `20260721-cd-environment-expose-service.md` | deploy 增加 `environment_id`；Render = Version × Environment |
| R2 | `20260722-cd-environment-ingress-policy.md`（Draft） | 环境侧 labels 输入改为 IngressPolicy，非 per-component Binding |

## Terminology / 规范用词（Accepted）

同 P0。核心句式：

```text
deploy(Application, Version)   // Version 必选，无「未指定」场景

  ① Service upsert（绑 version_id，status=deploying）+ Deployment
  ② Render(Service → Version.spec) → 引擎产物（如 compose 文件，非业务源）
  ③ Runtime apply（可带「本次操作选项」，如 force recreate）
  ④ 补齐 Container.component_name；Service = running | faulted
```

## Background / 背景

现状「读 Application 当前文件 → compose up」与 Version 模型冲突。  
本期：**大版本直接切换**，以 Version 为唯一规格前置；**不做**旧路径兼容。

## Goal / 目标

1. **仅**支持 `deploy(Application, Version)`；Version 为**硬前置**。
2. Deployment 记录 Version、操作选项、日志、结果。
3. Service 可查询意图 Version 与状态；stop 不删。
4. Component → Container 可观测。

## Non-goal / 非目标

1. 不实现 Environment / 多机 / K8s。
2. **不做**「部署当前未版本化文件」兼容或双轨。
3. 不在本需求做**域名/端口对外访问（原 application_route / Traefik 路由配置）**产品改造。
4. 不把 force recreate 等写进 Version。
5. 不引入用户可编辑的 raw compose yaml 概念。

## User scenarios / 用户场景

1. 选定 Version 1.1（未发布或已发布）→ 部署 → Service 绑定 1.1。
2. 编辑未部署的 1.2 → 不影响当前 Service 的 1.1。
3. 部署时可勾选**本次选项**（如 force recreate）→ 记在 Deployment，不改 Version。
4. 失败：Service=faulted；Container / Deployment 可下钻。

## Functional requirements / 功能需求

1. **deploy 必须带 Version id**；无 Version 则拒绝（无默认 latest 业务）。
2. 规格只来自该 Version（published 冻结；unpublished 为部署瞬间内容）。
3. ①～④ 顺序同 P0。
4. **操作选项**（属 Deployment / 调用参数，**非 Version**），例如：
   - force recreate
   - remove volumes（若需要）
   - 其他与「这一次怎么执行」相关的开关  
   选项清单实现期可增，但**不得**写回 Version.spec。
5. stop/restart：针对 Service；新 Deployment；restart 用当前 `Service.version_id` 再走 ②③④。

## Acceptance / 验收标准

1. 无 Version 不能部署。
2. 不存在「部署 Application 当前脏文件」入口（或已删除）。
3. unpublished / published 均可在指定 Version 后部署。
4. 操作选项进 Deployment，不进 Version。
5. 渲染产物可落盘执行，SoT 仍是 Version。

## Open questions / 待确认

**业务向：无。**

仅实现/Plan 向（可选）：

1. Container 派生是否落库缓存。
2. 具体有哪些操作选项进第一期 UI（force recreate 是否一期就做）。

## Decisions / 决策

1. **Version 是部署前置**；不存在「未指定 Version」的合法业务场景。
2. **放弃历史兼容**；不做旧「部署当前文件」/双轨；大版本直接 Version 路径。
3. force recreate 等 = **部署时选项** → Deployment/请求参数。
4. 无 raw compose 产品路径。
5. 域名/路由对外访问 **本期不做**（见下「说明」）。
6. 依赖 **P0/P1 Accepted**。

## 说明：曾用词「暴露子系统」指什么

不是新造的平台模块名。指现状能力：

- 表/概念：`application_route` 等  
- 行为：给应用配 **域名 + 端口**，经 Traefik 等把 HTTP 流量引进容器  

曾有人把「这套域名/路由要不要塞进 Version」写成「暴露子系统」。  
**本期结论**：版本化闭环 **不包含** 改这套对外访问配置；Version 最小 schema 也不收 expose。以后若做，用「对外访问 / 路由」表述即可。

## Risk / 风险

1. 迁移时旧应用无 Version：上线前需 **数据迁移生成 Version**，不能保留旧部署入口。
2. 部署期若仍偷偷注入与 Version 不一致的内容，会破坏可复现性。

## User review notes

- 2026-07-21：顺序 Service → Render → apply → Container。
- 2026-07-21：用户澄清——Version 必选；无旧部署兼容；操作选项属部署；无 raw；「暴露」= 域名/路由且本期不做。
- 2026-07-21：用户接受 P0/P1，并确认 **放弃历史兼容**。
- 2026-07-21：用户 **接受 P2**，进入 Plan。
- 2026-07-22：文档头挂入 R1/R2 后续路线指针（cadence 索引）。
