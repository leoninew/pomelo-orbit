# Docker Label 路由指南
最后修改时间: 2026-07-24 10:47:37

Doc role: living guide。与代码冲突时以代码为准。

## 概述

从 v0.4.0 开始，Pomelo Orbit 支持通过 Docker Labels 配置路由，无需在数据库中手动创建路由记录。这种方式更适合容器化应用，路由配置与应用部署紧密结合。

## 架构

Pomelo Orbit 使用两种独立的 Traefik provider：

- **REST Provider**: Orbit 管理的 Platform Route，通过 `PUT /api/providers/rest` 发布完整 HTTP/TCP/TLS 快照
- **Docker Provider**: 基于 Docker Labels 的路由（Traefik 自动发现）

两种方式可以共存，互不干扰。

## 使用 Docker Labels 配置路由

### 基本示例

在应用的 `docker-compose.yml` 中添加 Traefik labels：

```yaml
services:
  app:
    image: nginx:alpine
    container_name: my-app
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.my-app.rule=Host(`app.localhost`)"
      - "traefik.http.routers.my-app.entrypoints=web"
      - "traefik.http.services.my-app.loadbalancer.server.port=80"

networks:
  traefik:
    external: true
```

### HTTPS 路由

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.my-app.rule=Host(`app.localhost`)"
  - "traefik.http.routers.my-app.entrypoints=websecure"
  - "traefik.http.routers.my-app.tls=true"
  - "traefik.http.services.my-app.loadbalancer.server.port=80"
```

### 路径前缀路由

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.my-app.rule=Host(`app.localhost`) && PathPrefix(`/api`)"
  - "traefik.http.routers.my-app.entrypoints=web"
  - "traefik.http.services.my-app.loadbalancer.server.port=8080"
```

### 多端口服务

如果服务暴露多个端口，需要明确指定：

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.my-app-web.rule=Host(`app.localhost`)"
  - "traefik.http.routers.my-app-web.entrypoints=web"
  - "traefik.http.routers.my-app-web.service=my-app-web"
  - "traefik.http.services.my-app-web.loadbalancer.server.port=80"

  - "traefik.http.routers.my-app-api.rule=Host(`api.localhost`)"
  - "traefik.http.routers.my-app-api.entrypoints=web"
  - "traefik.http.routers.my-app-api.service=my-app-api"
  - "traefik.http.services.my-app-api.loadbalancer.server.port=8080"
```

## 证书管理

Docker Label 只声明 Router 和 Service。Orbit 的手工和 mkcert 证书仅支持存储在已启用的 Platform HTTP Route，并通过 REST provider 的 `tls.certificates` 发布；不要为 Label 路由创建禁用占位 Route，也不要创建 `tls.yml` 或手工复制 PEM。需要独立管理 Label Router 的 TLS 时，请在该 Router 所属的 Traefik 配置中完成，而不是将其混入 Orbit 的 Route workspace。

## 路由发现

Docker Label Router 由 Traefik Dashboard 和 Docker labels 观察；Orbit 当前不会把它们同步为可编辑的 Route 记录。

## 最佳实践

### 1. 命名规范

- Router 名称使用应用名称（如 `my-app`）
- Service 名称与 Router 名称一致
- 容器名称与应用名称一致

### 2. 网络配置

所有需要通过 Traefik 访问的容器必须连接到 `traefik` 网络：

```yaml
networks:
  traefik:
    external: true
```

### 3. 端口配置

明确指定服务端口，避免 Traefik 自动检测错误：

```yaml
labels:
  - "traefik.http.services.my-app.loadbalancer.server.port=80"
```

### 4. 入口点选择

- HTTP: `entrypoints=web`
- HTTPS: `entrypoints=websecure`
- 同时支持: `entrypoints=web,websecure`

### 5. 证书管理

- 为生产环境使用有效证书
- 为开发环境使用 mkcert 生成的本地证书
- 定期更新证书

## 与 Platform Route 的对比

| 特性 | Platform Route（REST provider） | Docker Label |
|------|--------------|--------------|
| 配置位置 | 数据库 | `docker-compose.yml` |
| 管理方式 | UI/MCP 后同步完整 REST snapshot | 代码管理 |
| 动态更新 | Route 同步立即 PUT 到 Traefik REST API | 容器标签变化后由 Docker provider 发现 |
| 证书管理 | 手工 PEM、mkcert 或 Let's Encrypt | 由 Label Router 所属 Traefik 配置负责 |
| 适用场景 | 外部服务、静态路由 | 容器化应用 |
| 可编辑性 | 通过 Orbit 管理 | 修改 docker-compose.yml |

## 故障排查

### 路由不生效

1. 检查容器是否连接到 `traefik` 网络：
   ```bash
   docker network inspect traefik
   ```

2. 检查 Traefik 是否发现了路由：
   ```bash
   docker logs traefik
   ```

3. 访问 Traefik Dashboard 查看路由状态：
   ```
   http://localhost:8080
   ```

### 证书不生效

1. 检查 Router 的 `tls` label 和所选证书方案
2. 检查 Gateway Version 的 Traefik 配置与日志
3. 不要排查或创建 Orbit 的 `tls.yml`，它不属于当前实现

### 端口冲突

如果服务暴露多个端口，Traefik 可能选择错误的端口。解决方法：

```yaml
labels:
  - "traefik.http.services.my-app.loadbalancer.server.port=80"
```

## 示例应用

### Nginx 静态站点

```yaml
services:
  nginx:
    image: nginx:alpine
    container_name: nginx
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.nginx.rule=Host(`nginx.localhost`)"
      - "traefik.http.routers.nginx.entrypoints=web"
      - "traefik.http.services.nginx.loadbalancer.server.port=80"

networks:
  traefik:
    external: true
```

### Pomelo Orbit 自身

```yaml
services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    container_name: pomelo-orbit
    networks:
      - traefik
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.pomelo-orbit.rule=Host(`pomelo-orbit.localhost`)"
      - "traefik.http.routers.pomelo-orbit.entrypoints=websecure"
      - "traefik.http.routers.pomelo-orbit.tls=true"
      - "traefik.http.services.pomelo-orbit.loadbalancer.server.port=8000"

networks:
  traefik:
    external: true
```

## 参考资料

- [Traefik Docker Provider 文档](https://doc.traefik.io/traefik/providers/docker/)
- [Traefik 路由规则](https://doc.traefik.io/traefik/routing/routers/)
- [证书管理指南](./certificate-management.md)
