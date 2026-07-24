# CD 应用版本化：从 Version 渲染运行时产物
最后修改时间: 2026-07-22 12:02:43

Review status: Accepted

Flow mode: standard

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`（P3）

Depends on: P0、P1；与 P2 部署闭环配合

**后续路线（labels / 环境接入 / 应用类型，超出本 P3 原文「本期不做域名」）**：

| 修订 | 文档 | 与本 P3 关系 |
|------|------|----------------|
| R1 | `20260721-cd-environment-expose-service.md` | Render 增加 Expose × EnvironmentBinding → Traefik labels |
| R2 | `20260722-cd-environment-ingress-policy.md` | labels 合路改为 Version.Expose × Environment.IngressPolicy（**standard**） |
| R3 | `20260722-cd-application-kind-gateway.md` | **Application.kind** 分支：`gateway` 与 `standard` 不同 Render 策略（实现完成，待 Verification） |

## Terminology / 规范用词

同 P0。  

- **Version** = 业务规格 SoT  
- **渲染产物** = 引擎需要的文件（如 docker-compose.yml），**由系统生成**  
- **不存在** 产品意义上的「用户维护 raw compose yaml / overlay 优先级」

## Background / 背景

部署链路：

```text
Service（已绑 Version）→ Renderer(Version.spec) → 产物 → apply → Container
```

渲染是 **函数**：同一 Version → 同一产物（同平台解析上下文）。  
完成后，用户只编辑 Version/Component，**不**再编辑 compose 当业务源。

## Goal / 目标

1. 依赖方向：`Version.spec → Renderer → 产物 → apply`。
2. 第一实现：Docker Compose 渲染；K8s **仅预留**、本期不实现。
3. 预览与真实部署 **同一套渲染**。

## Non-goal / 非目标

1. 不实现 K8s。
2. **不提供** raw compose 编辑、overlay 合并、双源优先级。
3. 不在本需求改造域名/路由对外访问。
4. 模板引擎选型属实现细节。

## User scenarios / 用户场景

1. 预览 Version 1.1 将生成的 compose，与点部署时一致。
2. 改 Component 后重新预览/部署，产物随之变化。
3. （远期）K8s renderer 读同一 Version，不改业务字段。

## Design boundary / 设计边界

### Source of truth

- **只有** Version.spec（Component、env、mounts、networks…）。
- 落盘 compose **可丢弃重算**；禁止再当业务编辑入口。

### Renderer

| Renderer | 输入 | 输出 | 本期 |
|----------|------|------|------|
| ComposeRenderer | Version | compose 及附属生成文件 | 要有 |
| K8sRenderer | Version | manifests | 预留 |

### 附属配置文件

若 Component 需要配置文件内容，应作为 **Version 业务数据的一部分**（结构化字段或 Version 内受管文件内容），经渲染写出。  
**不是**「用户旁路丢一个 raw compose 进来覆盖结构化规格」。

### 部署操作选项 vs 渲染

- force recreate 等：**不进 Renderer 输入规格**；只影响 apply 怎么执行（P2）。

## Acceptance / 验收标准

1. 写清 SoT 与「产物非业务源」。
2. 预览 = 部署同一 render 原则。
3. **明确无 raw compose 产品路径**。
4. K8s 仅预留。

## Open questions / 待确认

**业务向：无。**

实现/Plan：生成文件落盘路径约定、是否与旧目录布局对齐。

## Decisions / 决策

1. 输入 = Version；输出 = 引擎产物。
2. **无 raw compose / overlay 优先级** 说法与功能。
3. 预览与 deploy 共享 render。
4. 域名/路由注入若将来需要，不在本 P3 用「子系统」架空；**本期不做**。
5. 本需求不交付代码（至 Plan）。

## Risk / 风险

1. 实现仍读写旧 compose 当编辑源 → 违反 SoT。
2. 预览与部署两套逻辑 → 结果不一致。

## User review notes

- 2026-07-21：用户否定 raw compose 与 overlay 优先级；渲染仅为 Version → 产物。
- 2026-07-21：用户 **接受 P3**，进入 Plan。
- 2026-07-22：文档头挂入 R1/R2 后续路线（环境 labels 策略）；本 P3 正文范围不改写。
