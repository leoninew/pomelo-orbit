# Pomelo Orbit 路由和证书
最后修改时间: 2026-08-15 12:48:49

Doc role: living guide  
领域边界见 [CD 模型](../product/cd-model.md)，运行链路见 [CD 运行时](../architecture/cd-runtime.md)。与代码冲突时以代码为准。

## 路由模型

Pomelo Orbit 使用受管 Gateway 的 Traefik `providers.rest` 管理平台 Route。每次同步均对 REST provider 发出完整快照，快照同时包含顶层 `http` 与 `tcp` namespace，因此禁用、删除 Route 或 Gateway 重建后不会遗留旧动态配置。

在覆盖前，Orbit 会检查 Traefik 当前的 `@rest` router 是否都对应 Route 表中的启用 Route。发现未登记的 `@rest` router 时，操作会以 `409/unmanaged_traefik_route` 拒绝，数据库和 Traefik 均不变。Traefik REST provider 只支持全量 PUT，无法保留未知规则；遗留的自定义 HTTP 路由必须先在 Orbit 中创建同等 Route（可使用“高级：自定义下游地址”），或由运维迁移到非 REST provider。

| 类型 | 地址 | Target | Traefik 动态规则 |
|------|------|--------|------------------|
| HTTP Route | `https?://domain/path_prefix` | 默认同项目 Service Component 的 HTTP Endpoint；高级模式可填 HTTP(S) URL | `Host(...)`，可附加 `PathPrefix(...)` |
| TCP Route | `domain:listen_port` | 同项目 Service Component 的 TCP Endpoint | `HostSNI(*)`；所需 entrypoint 与宿主机端口由目标 Gateway Version 显式声明 |

普通 TCP 不携带 HTTP Host。TCP Route 的 `domain` 只用于 DNS 和客户端连接地址，而不是同端口分流条件。因此一个启用 TCP Route 独占一个 `listen_port`；不支持 TLS/SNI 共享端口、任意地址 target 或 UDP。

## HTTP Route

HTTP Route 保持现有域名、路径与证书能力。创建或编辑时默认选择同项目 Service、Component 和已声明 HTTP Endpoint；平台在发布完整 REST 快照时解析目标 Service 当前有效 Endpoint 为 `http://<container-alias>:<container-port>`。只有主动开启“高级：自定义下游地址”时，才填写任意 HTTP(S) URL；受管 Endpoint 与自定义 URL 互斥。

```json
{
  "protocol": "http",
  "name": "my-app",
  "domain": "app.example.com",
  "path_prefix": "/",
  "service_id": "01...",
  "component_name": "api",
  "endpoint_protocol": "http",
  "endpoint_container_port": 8080,
  "enabled": true
}
```

启用 HTTPS 后，Route 使用 `websecure` entrypoint；否则使用 `web`。手工证书与 mkcert 的 PEM 数据储存在 Route 中，文件以 `{route-name}.pem` 和 `{route-name}-key.pem` 写入 Gateway 证书目录。Let's Encrypt 路由在 REST 快照中使用 `tls.certResolver=letsencrypt`。

## TCP Route

先在目标 Component 声明 `internal` TCP Endpoint。它不直接发布宿主机端口或生成 Docker/Traefik label，可由 TCP Route 引用。

```json
{
  "protocol": "tcp",
  "name": "redis-public",
  "domain": "redis.example.com",
  "listen_port": 16379,
  "service_id": "<service-id>",
  "component_name": "redis",
  "endpoint_protocol": "tcp",
  "endpoint_container_port": 6379,
  "enabled": true
}
```

平台校验 target Service 与 Route 属于同一 Project，Component 和 Endpoint 存在且为 TCP Endpoint。TCP Route 提供的是 `domain:listen_port` 接入能力；Endpoint mode 只决定是否生成直接的宿主机端口映射，不影响其作为路由目标的资格。`listen_port` 必须在 `1..65535`，不得使用 Gateway 的 `80`、`443`、`8080`，不得被另一启用 TCP Route、`local` Endpoint 或 `host` Endpoint 占用。

保存、更新、启停、删除或手工同步 Route 时，平台只发布完整 HTTP/TCP REST 动态快照，不会计算或修改 Gateway 静态端口。启用 TCP Route 前，用户必须先在目标 Gateway Version 中显式声明对应 `tcp<listen_port>` entrypoint 与宿主机端口并部署该 Version。Gateway deploy/restart 在 `compose up` 成功后，会先轮询 `rest_api_url`（Traefik API，默认本机 8080）直到控制面就绪，再全量发布动态 Route REST 快照；容器起来但 API 尚未监听时不会立刻 PUT。发布失败会使 Gateway Service 和 Deployment 进入 `faulted`。

## Endpoint Mode

Endpoint mode 仅有：`internal`、`local`、`host`、`gateway`。

- `gateway` 只能用于 HTTP Endpoint，生成组件派生 Host：`{component_name}.{service_code}.{gateway.base_domain}`。
- TCP Route 可引用任意声明的 TCP Endpoint；Endpoint mode 不影响路由资格。
- `local` 与 `host` 是直接的宿主机端口映射，不能与 TCP Route 监听端口重叠。

Endpoint 不保存名称，身份为同一 Component 内的 `(protocol, container_port)`，界面派生显示 `http<container_port>` 或 `tcp<container_port>`。该结构和 mode 枚举已合并到当前迁移基线；开发数据库升级时重建，不提供离线转换脚本或运行时兼容路径。

## 排查

1. 确认 Route 已启用，并在 Route 页面执行同步。
2. 对 TCP Route，确认目标 Gateway Service 绑定的 Version 已显式包含对应 entrypoint 与宿主机端口，然后部署或重启该 Service。
3. 确认 DNS 将 Route 域名解析到 Gateway 主机，且操作系统/防火墙允许对应 TCP 端口。
4. 通过 Gateway 的 Traefik Dashboard 或 `GET /api/http/routers`、`GET /api/tcp/routers` 检查动态 Route 是否已出现。
5. 若 Gateway Deployment faulted，查看该 Deployment 日志中的 Route snapshot 发布错误；Gateway 容器可能已启动，但动态 Route 未被确认发布。

## 证书边界

HTTP Route 可使用 `manual`、`mkcert` 或 `letsencrypt`。TCP Route 本需求不进行 TLS 终止、passthrough 或证书管理；公开 Redis、MySQL 等 TCP 服务仍需要由服务本身提供认证、TLS/ACL 和网络防火墙控制。
