# CD 领域模型（现行）
最后修改时间: 2026-07-27 13:41:16

Doc role: living SoT  
代码锚点：`internal/model/cd.go`、`internal/common/constant/status.go`、`internal/application/cd/usecase/*`、`sql/migration/*/…_cd_schema*.sql`（以仓库当前迁移文件为准）。

## 术语

| 中文 | 英文 / 类型 | 说明 |
|------|-------------|------|
| 应用 | `Application` | 长期身份；`code`/`name`；**kind** 决定渲染策略 |
| 应用类型 | `Application.kind` | `standard`（默认）\| `gateway`；**创建后不可改** |
| 版本 | `Version` | 结构化静态规格元数据；**不是** compose 全文存储 |
| 组件 | `VersionComponent` | Version 内可部署单元（镜像、挂载、env 等 JSON 规格） |
| 暴露 | `VersionExpose` | 协议 + 容器端口 + `access`；**不含域名** |
| 环境 | `Environment` | **Project 下**部署目标元数据；无 ingress 字段 |
| 网关配置 | `GatewayConfig` | 与 `kind=gateway` 应用 1:1：`rest_api_url`、`base_domain`、入口/TLS |
| 服务 | `Service` | 运行绑定与运行时配置 SoT：Application + `instance_key` + Version + 明文 K/V |
| 部署 | `Deployment` | 操作流水（deploy/stop/restart 等）与日志/状态 |
| 平台路由 | `Route` | 控制面配置的 Traefik rest 路由；**不是** Version 暴露 SoT |
| 容器 | Container | 物化观测概念；以 Docker 运行时为准 |

## 对象关系（逻辑）

```text
Project
  ├── Environment (1..*)
  ├── Application (1..*)
  │     ├── kind = standard | gateway
  │     ├── Version (1..*)
  │     │     ├── VersionComponent (1..*)
  │     │     └── VersionExpose (0..*)
  │     └── GatewayConfig (0..1，仅 gateway)
  ├── Service：Application × Environment × instance_key → Version
  ├── Deployment：操作记录（可关联 Service / Version）
  └── Route：平台级动态路由（rest provider）
```

## 生命周期与状态

### Version

- `unpublished`：可改规格  
- `published`：已发布（产品约束以实现/API 为准）

### Service（运行态 SoT，**不是** Application 状态机）

- `deploying` / `running` / `stopped` / `faulted`  
- 身份：`(application_id, environment_id, instance_key)`，默认 `instance_key=default`  
- stop 不删除 Service 行（绑定保留）
- `runtime_config_json` 是普通明文 `key/value` 配置；修改只更新待部署配置，不改变运行中的容器。
- Credential 不属于 Service 运行时配置或部署链路。

### Deployment（任务态）

- `waiting_to_run` → `running` → `ran_to_completion` \| `faulted` \| `canceled`  
- 与后台 task 队列配合；API 创建记录，worker 执行 Docker 操作

### Application

- **无** `undeployed/deployed` 产品状态机；运行态看 Service。

## 部署顺序（标准路径）

1. 选定 **Version**（部署前置，不可省略）。  
2. 确保 **Service** 绑定（app + env + instance）。  
3. 创建 Deployment 时快照完整 Service 运行时配置；worker 只读取该快照。
4. **Render(Version)** → 以快照解析 `${KEY}` / `${KEY:-default}`，生成 compose 等工作区文件（kind 分支）。
5. **apply**（`docker compose …`）。
6. 更新 Service / Deployment / 观测。

Gateway 部署与 standard 共用 Version/Deploy 管线；保存 Gateway 时可 compile 未发布 Version（托管 traefik 组件与静态配置）。同一时刻仅允许 **一个** gateway Service 处于 deploying/running。

## 暴露与接入

| 概念 | 现行规则 |
|------|----------|
| 出口 SoT | `VersionExpose`：`protocol`（http\|tcp 等）+ `access`（`local`\|`public`）+ 端口 |
| 域名 | public HTTP：`{app_code}.{gateway.base_domain}`（由 Gateway 提供 base_domain） |
| local | loopback host ports；不写该 expose 的 Traefik public labels |
| public TCP | 经 Gateway TCP entrypoint / labels；业务侧不另起 Gateway TCP 路由 CRUD |
| 平台 Route | 独立 `route` 表 + active Gateway 的 `rest_api_url` 全量 PUT rest provider |
| 废止 | `attach_ingress` / `service.is_ingress` / EnvironmentBinding 用户 SoT / 全局 `traefik.api_url`·`domain_suffix` 产品路径 / Application 级 route 当应用暴露 SoT |

## kind 与渲染

- **standard**：按 Component + Expose 渲染业务 compose；labels 由 Expose 驱动。  
- **gateway**：`renderGatewayCompose` 等分支；仍用 Version/Component 管线。  
- **禁止** 以 `code==traefik` 魔法替代 kind。

## 可用性目标（吸收自历史 analyze）

历史分析指出：旧模型接近「配置文件包管理」，新建应用易成空壳。现行方向为：

1. **Version 结构化规格** + Render，而不是用户手写 compose 作为唯一 SoT。  
2. **Gateway / Expose** 显式建模接入，而不是散落 compose labels 与全局 env。  
3. 创建与详情 UI 应对齐 Version/Component/Expose/Service 概念（细节以 `web/` 为准）。

（原文 `docs/analyze/application-management-usability.md` 已归档，描述的是旧 compose 文件包模型，**不可**再当现行产品说明。）

## 相关活文档

- [CD 运行时与渲染](../architecture/cd-runtime.md)  
- [决策账本](../decisions/ledger.md)  
- [路由与证书（how-to）](../guides/routing-and-certificates.md)  
