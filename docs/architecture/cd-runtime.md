# CD 运行时与渲染
最后修改时间: 2026-08-18

Doc role: living SoT  
代码锚点：`internal/application/cd/usecase/compose_renderer.go`、`deployment_execution*.go`、`gateway*.go`、`expose_*.go`、`internal/infrastructure/runner/cd`、`internal/infrastructure/storage/local/cdworkspace`、`internal/infrastructure/external/traefik`。

## 控制流

```text
HTTP Deploy/Stop/Restart
  → Deploy 对话框必要时先更新 Service.Version；部署请求只指定 Service
  → usecase 读取 Service 当前 Version，快照 Service RuntimeConfig 到 Deployment.options_json
  → 写 Deployment / Task（Service 不表达操作进行态）
  → worker 拾取 task
  → deployment_execution：只读 Deployment 快照，解析占位符 → Render → 工作区 → docker compose
  → 更新 Deployment / Service 状态与日志
```

- API **不**在请求路径内直接 `docker compose down/up` 作为同步主路径；由任务队列驱动。  
- 项目名 / compose project 命名以实现代码为准（常见含 app、env、instance）。
- worker 领取后先条件转换 `waiting_to_run -> running`，然后才执行 Render、工作区和 Docker 操作；完成写入只允许从 `running` 进入终态。
- Cancel API 立即条件写入 `canceled`。worker 按 `worker.poll_interval` 读取该状态并取消外部命令 Context，后续完成写入不能覆盖取消终态。

## RuntimeConfig

- Service 的 `runtime_config_json` 是运行时 K/V 的唯一业务 SoT，普通明文保存并通过 Service 项目权限读取和更新。
- 保存配置不触发 Docker 操作；只有后续 Deploy 或 Restart 的任务会使用新值。
- Deploy/Restart 创建时复制完整 K/V 到 `Deployment.options_json.runtime_config`。任务排队后修改 Service 配置不能改变该任务的 render 结果。
- Component 的 `environment` 值可用 `${KEY}` 表示必需配置，`${KEY:-default}` 表示带默认值的可选配置。修改关联 Version 的组件或发起部署时缺少必需 key 会被拒绝；多余 key 被保留，worker 在 Deployment 日志写 warning 并忽略它们。
- Credential、解密、组件级 `secret_env_refs`、`.runtime/*.env` 和 compose `env_file` 均不参与此路径。

## Render

输入：Application（含 kind）+ Service（含不可变 `code`）+ Version 的 Components/Endpoints +（可选）GatewayConfig。Endpoint 以 `(protocol, container_port)` 定位，`http<port>` / `tcp<port>` 仅为派生显示。

| kind | 行为 |
|------|------|
| `standard` | 渲染业务 services；按 Endpoint mode 注入 HTTP labels 或直接端口映射 |
| `gateway` | 使用选中 Version/Service 的通用 compose 渲染；Gateway 不在 Render 时注入或恢复 Traefik Component、静态配置或挂载 |

输出写入 `workspace.deployment/<service-code>/`，再执行 compose。逻辑目录由 Orbit 进程写入；原生运行时，组件相对 bind source 保持相对并由 Compose 相对该目录的 `docker-compose.yml` 解析。DooD 运行时才改为 Docker daemon 可见的宿主路径。

预览（Preview）与部署应走同一套渲染语义（测试与 usecase 对齐）。

对每个 `gateway` HTTP Endpoint，Host 统一派生为 `{component_name}.{service.code}.{gateway.base_domain}`；同一 Host、entrypoint 与归一化 path prefix 只能对应一个 Endpoint。TCP Route 引用 `internal` TCP Endpoint，不生成业务 labels 或端口映射。

## Gateway

| 字段（GatewayConfig） | 用途 |
|----------------------|------|
| `rest_api_url` | Traefik providers.rest 控制面 |
| `base_domain` | 业务 public Host 后缀 |
| `default_entrypoint` | HTTP 入口名（如 web） |
| `tls_mode` | `none` / `tls` / `letsencrypt` 等（以实现枚举为准） |
进程配置 `traefik.*` 与 `cert.letsencrypt.*` 只生成初始 Traefik Version 模板：镜像、REST 地址、域名、就绪等待、ACME email/challenge/provider 均从既有配置取得。证书/ACME 目录固定派生为 `workspace.deployment/traefik/data/certs`，并以 Docker daemon 可见宿主路径写入初始 host-path mount。拉取策略 `missing`、入口 `web`、TLS `none` 和共享网络 `traefik` 保持当前默认，不新增配置项。创建后的镜像、端点、挂载、静态文件和拉取策略均由 Application/Version 编辑保存；`GatewayConfig` 只保存 REST、域名、默认入口和 TLS 路由语义。

- 创建：同一事务写入 Application、GatewayConfig、初始 `unpublished` Version/Component、默认停止态 Service 和 Service Component mappings；默认 Service code 为 `<application-code>-default`。
- Provision：`orbit_provision_gateway` 只幂等准备上述资源或指定实例的 Service，不发布 Version、不创建 Deployment、不等待运行态，也不重绑既有 Service。
- Route 保存、启停、删除和同步只发布 HTTP/TCP 动态 REST 快照，不修改 Gateway Version。新增 TCP `entrypoint` 或宿主机端口时，用户先在目标 Gateway Version 中显式配置并部署。
- 同 Application 的 Gateway Service instance 与普通 Application 一样遵从单运行实例规则；没有 Gateway 专用 active/readiness 前置检查。

## 部署结果

- Deploy/Restart 使用 `docker compose up -d` 的零退出码作为成功条件。Gateway Service 在写入 `running` 后、Deployment 标记成功前，先等待 Traefik REST 控制面（`rest_api_url`，默认本机 8080）就绪，再发布完整 HTTP/TCP Route REST 动态快照；不在部署后编译或重写 Gateway Version。`compose up` 成功不代表 API 已监听。发布失败时 Service 与 Deployment 均标记为 `faulted`；缺失 TCP 静态 entrypoint 走用户保存的 Version/Compose 或 Route 发布错误路径。
- Component Healthcheck 会渲染到 Compose，`depends_on.condition=service_healthy` 可影响 Compose 内启动顺序，但不决定 Deployment 主状态。独立验证接口才读取容器运行态与 Health 状态。
- Pipeline 的 `execution_timeout` 超时属于失败：Run 和仍在运行的 Stage 写为 `faulted`，错误原因包含 timeout；关联 task 进入 `failed`。

## 平台 Route vs Component Endpoint

| | Route | VersionComponentEndpoint |
|--|-------|----------------|
| 存储 | `route` 表 | `version_component_endpoint`（Service 可稀疏覆盖） |
| 下发 | REST API 全量 PUT `http` + `tcp` namespaces | 部署时 Docker HTTP labels / direct ports |
| 域名 | HTTP 使用用户 domain/path；TCP 使用 `domain:listen_port` | `gateway` 为 `{component_name}.{service.code}.{gateway.base_domain}` |
| TCP | 仅一个启用 Route 独占一个监听端口，动态 rule 为 `HostSNI(*)` | `internal` TCP Endpoint 可作为 Route target；`local`/`host` 是直接映射 |

证书：平台 Route 可存 PEM；同步时写入 `workspace.deployment/traefik/data/certs`，并在 REST snapshot 的 `tls.certificates` 中以 Gateway 容器内 `/etc/traefik/certs` 路径引用。应用 HTTPS/ACME 与 Gateway `tls_mode` 及该目录相关。

## 工作区与物理路径

- `workspace.pipeline` 与 `workspace.deployment` 是后端拥有、会写入且会作为 Docker source 使用的唯一 CI/CD 根目录；Service code 是 Deployment 工作目录唯一键，不使用 Application code 或 instance key。SQLite、日志、导出和外部 Repository 不属于此配置组。
- 配置加载时，相对 workspace 值仅相对 `orbit.root` 解析一次；绝对值原样保留，可位于项目外。两个根目录不得相同或相互嵌套。
- Orbit 原生运行时，逻辑 workspace 路径用于文件读写，不调用 Docker inspect 或进行路径转换；平台相对的目录、文件和 controlled-file 挂载源保持相对，Compose 以生成的 `docker-compose.yml` 目录解析它们。DooD 下，两个 workspace root 必须分别 bind mount 进 Orbit 容器；启动会以 Docker inspect 验证映射。逻辑目录/controlled-file 挂载在 Orbit 可见的 Service 目录物化，Compose 接收对应的宿主机 source。显式 `source_is_host_path=true` 的绝对源在两种模式中均保持绝对路径。

## 相关

- [CD 产品模型](../product/cd-model.md)  
- [后端架构](./backend.md)  
