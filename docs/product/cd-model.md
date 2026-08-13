# CD 领域模型（现行）
最后修改时间: 2026-08-13 19:10:01

Doc role: living SoT  
代码锚点：`internal/model/cd.go`、`internal/common/constant/status.go`、`internal/application/cd/usecase/*`、`sql/migration/*/…_cd_schema*.sql`（以仓库当前迁移文件为准）。

## 术语

| 中文 | 英文 / 类型 | 说明 |
|------|-------------|------|
| 应用 | `Application` | 长期身份；`code`/`name`；**kind** 决定渲染策略 |
| 应用类型 | `Application.kind` | `standard`（默认）\| `gateway`；**创建后不可改** |
| 版本 | `Version` | 结构化静态规格元数据；**不是** compose 全文存储 |
| 组件 | `VersionComponent` | Version 内可部署单元（镜像、挂载、env 等 JSON 规格） |
| 端点 | `VersionComponentEndpoint` | `(protocol, container_port)` + mode；名称由 `http<port>` / `tcp<port>` 派生，不持久化 |
| 环境 | `Environment` | **Project 下**部署目标元数据；无 ingress 字段 |
| 网关配置 | `GatewayConfig` | 与 `kind=gateway` 应用 1:1：`rest_api_url`、`base_domain`、入口/TLS |
| 服务 | `Service` | 运行绑定与运行时配置 SoT：Application + `instance_key` + Version + 不可变 `code` + 明文 K/V |
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

- `running` / `stopped` / `faulted`
- 身份：`(application_id, environment_id, instance_key)`，默认 `instance_key=default`  
- `code` 是全局唯一、创建后只读且不可变的 DNS-label 路由身份；创建窗以 `<application_code>-<instance_key>` 预填，用户提交后不随 Application code、instance key 或 Version 变更。
- stop 不删除 Service 行（绑定保留）
- `runtime_config_json` 是普通明文 `key/value` 配置；修改只更新待部署配置，不改变运行中的容器。
- Credential 不属于 Service 运行时配置或部署链路。

### Component Mount

- Version 挂载声明类型、默认源路径和容器目标；普通 Version 编辑不设置宿主机路径模式。
- Service 组件的稀疏覆盖拥有实际源路径及其宿主机路径模式。目录和文件在该模式下必须使用绝对路径；逻辑源路径仍须为相对路径。
- 服务工作目录固定为 `data/deployment/<service-code>/`；逻辑目录源直接相对此根解析。组件应以自身 code 开始组织数据，例如 `mysql`、`redis`、`tei/cache`，不使用 `data/` 或 `instance_key` 作为目录层级。
- Gateway compile 生成的托管挂载属于内部实现，不构成 Version 编辑页的用户配置项。

### 异步操作（Deployment / Pipeline Run / Pipeline Stage Run）

- `waiting_to_run` → `running` → `ran_to_completion` \| `faulted` \| `canceled`，也可从 `waiting_to_run` 直接进入 `canceled`。
- `ran_to_completion`、`faulted`、`canceled` 是不可逆终态。状态写入必须验证当前源状态，worker 不能覆盖已取消的记录。
- Cancel API 立即写入 `canceled`；worker 以此作为停止 Docker 或 Stage 容器的持久化信号。`canceled` 表示取消已接受，不承诺外部副作用已经回滚。
- Pipeline 上游失败时，尚未开始的下游 Stage 保持 `waiting_to_run`。Pipeline Retry 是创建新 Run 的显式命令，使用 `retry_of` 记录谱系，不改变原 Run。
- `background_task` 是调度记录。创建时将 `worker.max_attempts` 冻结到记录；失败时只依据持久化的 `attempts/max_attempts` 决定重投或 `failed`。开发默认预算为 `1`，因此常规业务任务首次失败即终结。

### CI 阶段模型

- `PipelineStage(kind=template)` 是项目内可复用的执行定义，包含名称、镜像、脚本、制品声明、说明和版本；不保存 DAG、排序或 Application/Component 绑定。模板制品声明不得带 `component_name`。
- Template Pipeline 用 `PipelineStageReference` 保存引入时的来源快照、制品声明及本地 DAG/排序；Application Pipeline 由该引用快照物化自己的 `PipelineStage(kind=application)`，并保留非空来源阶段 ID、名称、版本与说明快照。
- Application Pipeline 在实例化时选择 Application、来源 Version 策略及 Docker 制品到 Component 的绑定。Snapshot/Run 只读取应用私有阶段的冻结定义，绝不在运行时回读可变阶段库。
- 阶段库的修改不会自动同步引用或应用阶段。用户可在节点编辑器预览差异后显式更新来源版本；模板删除不影响既有引用、应用阶段、Snapshot、Run 或 Artifact。

### Application

- **无** `undeployed/deployed` 产品状态机；运行态看 Service。

## 部署顺序（标准路径）

1. 选定 **Version**（部署前置，不可省略）。  
2. 确保 **Service** 绑定（app + env + instance）。  
3. 创建 Deployment 时快照完整 Service 运行时配置；worker 只读取该快照。
4. **Render(Version)** → 以快照解析 `${KEY}` / `${KEY:-default}`，生成 compose 等工作区文件（kind 分支）。
5. **apply**（`docker compose …`）。
6. 更新 Service / Deployment / 观测。

Gateway 部署与 standard 共用 Version/Deploy 管线；保存 Gateway 时可 compile 未发布 Version（托管 traefik 组件与静态配置）。同一时刻仅允许一个实际 `running` 的 gateway Service，或另一 gateway Service 的活跃 Deployment。

## 暴露与接入

| 概念 | 现行规则 |
|------|----------|
| Endpoint mode | `internal`、`local`、`host`、`gateway`。`gateway` 仅用于组件派生 HTTP Host；TCP Route 引用 `internal` TCP Endpoint |
| local / host | 直接生成宿主机端口映射；其监听端口不能与 TCP Route 冲突 |
| 平台 HTTP Route | 独立 `route` 表，默认按 domain/path → 项目内 Service Component 的 HTTP Endpoint，经 active Gateway 的 `rest_api_url` 全量 PUT `http` rest namespace；直接 HTTP(S) URL 仅为高级自定义下游 |
| 平台 TCP Route | 独立 `route` 表，按 `domain:listen_port` → 项目内 Service Component 的 `internal` TCP Endpoint；每个启用监听端口唯一，经 Gateway entrypoint 与 `tcp` rest namespace 转发 |
| Endpoint identity | Version Endpoint、Service Endpoint overlay 与 Route 受管 target 都以 `(protocol, container_port)` 关联；旧 overlay/受管 Route 只允许离线清理后迁移 |
| 存量转换 | `gateway_http → gateway`、`gateway_tcp → internal`、`tcp → internal` 由发布前离线 SQL 处理；运行时不兼容旧值 |
| 废止 | `attach_ingress` / `service.is_ingress` / EnvironmentBinding 用户 SoT / 全局 `traefik.api_url`·`domain_suffix` 产品路径 / Application 级 route 当应用暴露 SoT |

## kind 与渲染

- **standard**：按 Component + Endpoint 渲染业务 compose；HTTP labels 或直接 ports 由 Endpoint mode 驱动。
- **gateway**：`renderGatewayCompose` 等分支；仍用 Version/Component 管线。  
- **禁止** 以 `code==traefik` 魔法替代 kind。

## 可用性目标（吸收自历史 analyze）

历史分析指出：旧模型接近「配置文件包管理」，新建应用易成空壳。现行方向为：

1. **Version 结构化规格** + Render，而不是用户手写 compose 作为唯一 SoT。  
2. **Gateway / Endpoint / Route** 显式建模接入，而不是散落 compose labels 与全局 env。
3. 创建与详情 UI 应对齐 Version/Component/Endpoint/Service 概念（细节以 `web/` 为准）。

（原文 `docs/analyze/application-management-usability.md` 已归档，描述的是旧 compose 文件包模型，**不可**再当现行产品说明。）

## 相关活文档

- [CD 运行时与渲染](../architecture/cd-runtime.md)  
- [决策账本](../decisions/ledger.md)  
- [路由与证书（how-to）](../guides/routing-and-certificates.md)  
