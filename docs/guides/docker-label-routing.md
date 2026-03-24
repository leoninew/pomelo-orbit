# Docker Label 路由指南

## 概述

从 v0.4.0 开始，Pomelo Orbit 支持通过 Docker Labels 配置路由，无需在数据库中手动创建路由记录。这种方式更适合容器化应用，路由配置与应用部署紧密结合。

## 架构

Pomelo Orbit 使用混合路由架构：

- **File Provider**: 传统的基于 YAML 文件的路由（从数据库生成）
- **Docker Provider**: 基于 Docker Labels 的路由（Traefik 自动发现）

两种方式可以共存，互不干扰。

## 路由源类型

系统中的路由分为三种类型：

| 类型 | 说明 | 管理方式 | 证书管理 |
|------|------|---------|---------|
| `file_provider` | 传统路由 | 数据库 + YAML 文件 | 支持 |
| `docker_label` | Docker Label 路由 | Docker Labels（只读） | 支持 |
| `external` | 外部服务路由 | 数据库 + YAML 文件 | 支持 |

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

Traefik 路由（从 Docker Provider 自动发现）本身不存储在数据库中，但可以通过以下方式管理证书：

### 方式1：创建对应的 Route 记录（推荐）

1. 在"路由配置"页面手动创建一个 Route 记录：
   - 名称：与 Traefik 路由相同（如 `my-app`）
   - 域名：与 Traefik 路由相同（如 `app.localhost`）
   - 目标 URL：可以填写任意值（如 `http://localhost:80`）
   - 启用状态：禁用（因为路由规则由 Docker Labels 管理）

2. 为这个 Route 记录上传证书

3. 执行"同步路由"操作，系统会生成集中式 `tls.yml` 配置

4. 证书自动应用到对应域名的 Traefik 路由

### 方式2：直接管理证书文件

如果不想创建 Route 记录，可以直接管理证书文件：

1. 生成证书：
   ```bash
   uv run --project backend/ python scripts/cert.py new -n app.localhost
   ```

2. 手动复制证书文件到 Traefik 证书目录：
   ```bash
   cp scripts/cert/app.localhost.pem backend/data/traefik/certs/my-app.pem
   cp scripts/cert/app.localhost-key.pem backend/data/traefik/certs/my-app-key.pem
   ```

3. 手动创建或更新 `backend/data/traefik/dynamic/tls.yml`：
   ```yaml
   tls:
     certificates:
       - certFile: /certs/my-app.pem
         keyFile: /certs/my-app-key.pem
   ```

4. Traefik 会自动重新加载配置

**注意**: 方式1更推荐，因为证书会存储在数据库中，便于管理和同步。

## 路由发现

### 自动发现（未来功能）

系统将支持自动发现 Docker Label 路由并同步到数据库：

```bash
# 手动触发发现（未来功能）
POST /api/route/discover
```

### 手动创建记录（当前方式）

如果需要为 Docker Label 路由上传证书，需要手动创建路由记录：

```bash
POST /api/route
{
  "name": "my-app",
  "domain": "app.localhost",
  "path_prefix": "/",
  "target_url": "",
  "enabled": true,
  "source_type": "docker_label"
}
```

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

## 与 File Provider 路由的对比

| 特性 | File Provider | Docker Label |
|------|--------------|--------------|
| 配置位置 | 数据库 + YAML 文件 | docker-compose.yml |
| 管理方式 | UI 手动管理 | 代码管理 |
| 动态更新 | 需要同步操作 | 容器启动时自动生效 |
| 证书管理 | 支持 | 支持 |
| 适用场景 | 外部服务、静态路由 | 容器化应用 |
| 可编辑性 | 完全��编辑 | 只读（需修改 docker-compose.yml） |

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

1. 检查证书是否正确上传
2. 检查 `tls.yml` 配置文件是否生成
3. 检查证书文件是否存在于 `certs/` 目录
4. 重启 Traefik 容器

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
