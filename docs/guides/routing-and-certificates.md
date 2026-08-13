# Pomelo Orbit 路由和证书
最后修改时间: 2026-08-13 19:10:01

Doc role: living guide  
领域边界见 [CD 模型](../product/cd-model.md)，运行链路见 [CD 运行时](../architecture/cd-runtime.md)。与代码冲突时以代码为准。

## 路由模型

Pomelo Orbit 使用受管 Gateway 的 Traefik `providers.rest` 管理平台 Route。每次同步均对 REST provider 发出完整快照，快照同时包含顶层 `http` 与 `tcp` namespace，因此禁用、删除 Route 或 Gateway 重建后不会遗留旧动态配置。

| 类型 | 地址 | Target | Traefik 动态规则 |
|------|------|--------|------------------|
| HTTP Route | `https?://domain/path_prefix` | 默认同项目 Service Component 的 HTTP Endpoint；高级模式可填 HTTP(S) URL | `Host(...)`，可附加 `PathPrefix(...)` |
| TCP Route | `domain:listen_port` | 同项目 Service Component 的 TCP Endpoint | `HostSNI(*)`，固定 `tcp<listen_port>` entrypoint |

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

平台校验 target Service 与 Route 属于同一 Project，Component 和 Endpoint 存在且为 TCP `internal` Endpoint。`listen_port` 必须在 `1..65535`，不得使用 Gateway 的 `80`、`443`、`8080`，不得被另一启用 TCP Route、`local` Endpoint 或 `host` Endpoint 占用。

保存、更新、启停、删除或手工同步 Route 时，平台从全部启用 TCP Route 计算 Gateway 端口集合，编译 `tcp<listen_port>` 静态 entrypoint 与 `0.0.0.0:<listen_port>:<listen_port>` Compose 映射。该静态变更要在随后部署 Gateway 后才对外生效。Gateway deploy/restart 在 `compose up` 成功后，会先轮询 `rest_api_url`（Traefik API，默认本机 8080）直到控制面就绪，再全量发布动态 Route REST 快照；不再在部署后 compile Gateway Version。容器起来但 API 尚未监听时不会立刻 PUT。发布失败会使 Gateway Service 和 Deployment 进入 `faulted`。

## Endpoint Mode

Endpoint mode 仅有：`internal`、`local`、`host`、`gateway`。

- `gateway` 只能用于 HTTP Endpoint，生成组件派生 Host：`{component_name}.{service_code}.{gateway.base_domain}`。
- TCP Route 仅引用 TCP `internal` Endpoint。
- `local` 与 `host` 是直接的宿主机端口映射，不能与 TCP Route 监听端口重叠。

在部署包含此枚举的新二进制前，先执行同版本交付的离线转换脚本：

- SQLite: `sql/migration/offline/000033_endpoint_mode_rename.sqlite.sql`
- MySQL: `sql/migration/offline/000033_endpoint_mode_rename.mysql.sql`

它将 `gateway_http` 转换为 `gateway`，并将 `gateway_tcp` 与旧的 `tcp` mode 转换为 `internal`。脚本不由应用启动或业务 usecase 调用；请在执行前备份数据库。

Endpoint 不保存名称，身份为同一 Component 内的 `(protocol, container_port)`，界面派生显示 `http<container_port>` 或 `tcp<container_port>`。升级到 `000034_endpoint_identity` 前，先执行对应的 `sql/migration/offline/000034_endpoint_identity.{sqlite,mysql}.sql`：该脚本清空旧 Service Endpoint overlay 与受管 Route，随后再执行结构迁移；不在运行时转换旧数据。

## 排查

1. 确认 Route 已启用，并在 Route 页面执行同步。
2. 对 TCP Route，确认 Gateway 最新未发布 Version 已包含 `tcp<listen_port>`，然后部署或重启 Gateway。
3. 确认 DNS 将 Route 域名解析到 Gateway 主机，且操作系统/防火墙允许对应 TCP 端口。
4. 通过 Gateway 的 Traefik Dashboard 或 `GET /api/http/routers`、`GET /api/tcp/routers` 检查动态 Route 是否已出现。
5. 若 Gateway Deployment faulted，查看该 Deployment 日志中的 Route snapshot 发布错误；Gateway 容器可能已启动，但动态 Route 未被确认发布。

## 证书边界

HTTP Route 可使用 `manual`、`mkcert` 或 `letsencrypt`。TCP Route 本需求不进行 TLS 终止、passthrough 或证书管理；公开 Redis、MySQL 等 TCP 服务仍需要由服务本身提供认证、TLS/ACL 和网络防火墙控制。
