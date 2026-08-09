# CD 运行时与渲染
最后修改时间: 2026-08-09 18:08:31

Doc role: living SoT  
代码锚点：`internal/application/cd/usecase/compose_renderer.go`、`deployment_execution*.go`、`gateway*.go`、`expose_*.go`、`internal/infrastructure/runner/cd`、`internal/infrastructure/storage/local/cdworkspace`、`internal/infrastructure/external/traefik`。

## 控制流

```text
HTTP Deploy/Stop/Restart
  → usecase 校验 Service/Version 匹配，快照 Service RuntimeConfig 到 Deployment.options_json
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

输入：Application（含 kind）+ Service（含不可变 `code`）+ Version 的 Components/Exposes + Environment/instance +（可选）GatewayConfig。

| kind | 行为 |
|------|------|
| `standard` | 渲染业务 services；按 Expose 注入 labels / local ports |
| `gateway` | 网关 compose（托管 Traefik 组件、静态配置、socket 挂载等） |

输出写入 CD workspace（物理数据根下 `deployment/<service-code>/`），再执行 compose。

预览（Preview）与部署应走同一套渲染语义（测试与 usecase 对齐）。

对每个 `gateway_http` endpoint，Host 统一派生为 `{component_name}.{service.code}.{gateway.base_domain}`；同一 Host、entrypoint 与归一化 path prefix 只能对应一个 endpoint。TLS `gateway_tcp` 使用同一 Host 作为 SNI；无 TLS TCP 仍使用 `HostSNI(*)`。

## Gateway

| 字段（GatewayConfig） | 用途 |
|----------------------|------|
| `rest_api_url` | Traefik providers.rest 控制面 |
| `base_domain` | 业务 public Host 后缀 |
| `default_entrypoint` | HTTP 入口名（如 web） |
| `tls_mode` | `none` / `tls` / `letsencrypt` 等（以实现枚举为准） |
| `image` | 可选覆盖网关镜像 |

- 保存/编译：可生成未发布 Version（compile）。  
- 部署 standard public 暴露前，可能需 reconcile Gateway TCP listen 集合（public TCP）。  
- **单 active gateway**：已有 `running` gateway Service，或另一 gateway Service 存在 `waiting_to_run` / `running` Deployment 时拒绝冲突部署。

## 部署结果

- Deploy/Restart 使用 `docker compose up -d` 的零退出码作为成功条件，并立即将 Service 更新为 `running`、Deployment 更新为 `ran_to_completion`。
- Component Healthcheck 会渲染到 Compose，`depends_on.condition=service_healthy` 可影响 Compose 内启动顺序，但不决定 Deployment 主状态。独立验证接口才读取容器运行态与 Health 状态。
- Pipeline 的 `execution_timeout` 超时属于失败：Run 和仍在运行的 Stage 写为 `faulted`，错误原因包含 timeout；关联 task 进入 `failed`。

## 平台 Route vs 应用 Expose

| | Route | VersionExpose |
|--|-------|----------------|
| 存储 | `route` 表 | `version_expose` |
| 下发 | rest API 全量 PUT | 部署时 Docker labels / ports |
| 域名 | 用户配置 domain | `gateway_http` 为 `{component_name}.{service.code}.{base_domain}` |

证书：平台 Route 可存 PEM；应用 HTTPS/ACME 与 Gateway `tls_mode`、证书目录配置相关——细节见 `docs/guides/routing-and-certificates.md` 与代码（以代码为准）。

## 工作区与物理路径

- 数据根与服务目录由配置 / `physical_data_root` 等决定；Service code 是工作目录唯一键，不使用 Application code 或 instance key。
- 挂载物化：logical mount → 宿主机目录/文件（含 content seed）；逻辑 `directory` 挂载会在 Compose 执行前创建，组件数据目录直接相对服务根组织。

## 相关

- [CD 产品模型](../product/cd-model.md)  
- [后端架构](./backend.md)  
