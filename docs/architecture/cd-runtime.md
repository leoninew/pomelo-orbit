# CD 运行时与渲染
最后修改时间: 2026-07-24 10:47:37

Doc role: living SoT  
代码锚点：`internal/application/cd/usecase/compose_renderer.go`、`deployment_execution*.go`、`gateway*.go`、`expose_*.go`、`internal/infrastructure/runner/cd`、`internal/infrastructure/storage/local/cdworkspace`、`internal/infrastructure/external/traefik`。

## 控制流

```text
HTTP Deploy/Stop/Restart
  → usecase 校验 + 写 Deployment / Service / Task
  → worker 拾取 task
  → deployment_execution：Render → 工作区 → docker compose
  → 更新 Deployment / Service 状态与日志
```

- API **不**在请求路径内直接 `docker compose down/up` 作为同步主路径；由任务队列驱动。  
- 项目名 / compose project 命名以实现代码为准（常见含 app、env、instance）。

## Render

输入：Application（含 kind）+ Version 的 Components/Exposes + Environment/instance +（可选）GatewayConfig。

| kind | 行为 |
|------|------|
| `standard` | 渲染业务 services；按 Expose 注入 labels / local ports |
| `gateway` | 网关 compose（托管 Traefik 组件、静态配置、socket 挂载等） |

输出写入 CD workspace（物理数据根下应用工作目录），再执行 compose。

预览（Preview）与部署应走同一套渲染语义（测试与 usecase 对齐）。

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
- **单 active gateway**：已有 deploying/running 的 gateway Service 时拒绝再部署另一个 gateway。

## 平台 Route vs 应用 Expose

| | Route | VersionExpose |
|--|-------|----------------|
| 存储 | `route` 表 | `version_expose` |
| 下发 | rest API 全量 PUT | 部署时 Docker labels / ports |
| 域名 | 用户配置 domain | public 时由 base_domain 推导 |

证书：平台 Route 可存 PEM；应用 HTTPS/ACME 与 Gateway `tls_mode`、证书目录配置相关——细节见 `docs/guides/routing-and-certificates.md` 与代码（以代码为准）。

## 工作区与物理路径

- 数据根与应用目录由配置 / `physical_data_root` 等决定。  
- 挂载物化：logical mount → 宿主机目录/文件（含 content seed）；历史 init.sh 主路径已弱化，以实现为准。

## 相关

- [CD 产品模型](../product/cd-model.md)  
- [后端架构](./backend.md)  
