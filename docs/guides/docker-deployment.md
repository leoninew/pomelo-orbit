# Docker 部署指南

本文档介绍如何通过 Docker 部署 Pomelo Orbit 系统。

## 架构概览

```
用户浏览器 ──→ Traefik ──→ Pomelo Orbit (Web UI)
                    │
                    └──→ 应用容器 (用户部署的应用)
```

核心组件：
- Traefik：反向代理和负载均衡器，负责动态路由
- Pomelo Orbit：持续部署系统，通过 Docker Socket 管理容器

## 1. 准备环境

确保 Docker 服务运行中：

```bash
docker --version
docker compose version
```

## 2. 部署步骤

### 2.1 关键配置

首次运行时，系统会自动创建 SQLite 数据库并初始化默认应用（Traefik 和 Pomelo Orbit 自身），其他配置可以使用 .env 管理，参考 `backend/.env.example` 

- POMELO_ORBIT_JWT__SECRET_KEY： JWT 密钥，用于 JWT 认证和凭据加密，必须使用 Fernet 格式
- POMELO_ORBIT_TRAEFIK__DOMAIN_SUFFIX，域名后缀，默认使用 `lvh.me`（本地无需配置 DNS）

### 2.2 docker-compose 启动 Pomelo Orbit

```yaml
services:
  pomelo-orbit:
    image: pomelo-orbit:latest
    container_name: pomelo-orbit
    restart: unless-stopped
    networks:
      - traefik
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data:/app/data
    ports:
      - "9003:80"
    environment:
      - POMELO_ORBIT_JWT__SECRET_KEY=${POMELO_ORBIT_JWT__SECRET_KEY}
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.pomelo-orbit.rule=Host(`pomelo-orbit.lvh.me`)"
      - "traefik.http.routers.pomelo-orbit.entrypoints=web"
      - "traefik.http.services.pomelo-orbit.loadbalancer.server.port=80"

networks:
  traefik:
    external: true
```

### 2.3 Pomelo Orbit 部署 Traefik

在 Pomelo Orbit Web UI 的应用管理页面部署 Traefik，需要先部署此组件。

### 2.4 Pomelo Orbit 管理和部署其他应用

通过 Pomelo Orbit Web UI 管理其他应用、配置文件和路由。

### 2.5 路由管理

Pomelo Orbit 提供两种路由管理方式：

- Docker Label：在 docker-compose.yml 中声明路由规则，Traefik 自动发现
- 手动配置：通过 Web UI 手动创建路由规则

### 2.6 Let's Encrypt

Let's Encrypt 自动证书功能代码已实现并经过测试，你需要修改 .env 启用配置

```ini
POMELO_ORBIT_CERT__LETSENCRYPT__ENABLED=true
POMELO_ORBIT_CERT__LETSENCRYPT__EMAIL=your-email@example.com
```

## 3. 已知问题

### 3.1 mkcert

本地开发推荐使用 [mkcert](https://github.com/FiloSottile/mkcert) 生成受信任的本地证书，容器还没有集成。

### 3.2 自举部署

理论上支持通过 Web UI 重新部署 Pomelo Orbit 自身实现零停机更新，但尚未经过充分测试。
