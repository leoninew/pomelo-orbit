# Pomelo Orbit

为本地容器环境打造的轻量级持续部署系统 + 动态路由网关管理，适用于 Windows WSL2/macOS 及 Linux 服务器等环境。

通过 Docker Socket 管理容器，支持 Web UI 管理应用、配置文件和路由规则，支持 GitHub 代码推送自动部署，可自举部署实现零停机更新。

## 功能

- Web UI 管理 - 直观的界面管理应用和部署
- 动态路由 - 基于 Traefik 的动态路由管理
- Webhook 部署 - GitHub Webhook 触发自动部署
- 自举部署 - 通过 UI 部署自己，实现零停机更新


## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Python 3.12 + FastAPI + SQLAlchemy + SQLite/MySQL |
| 前端 | Vue 3 + Vite + Ant Design Vue + TypeScript |
| 网关 | Traefik (动态路由) |
| 容器 | Docker + docker-compose |

## 架构

### 持续部署

![持续部署流程](docs/deployment-flow.drawio.png)

### 动态路由

![动态路由原理](docs/routing-system.drawio.png)

## 快速开始

### 本地开发

```bash

# 安装依赖
make install

# 启动后端 (端口 9001)
make dev-backend

# 启动前端 (端口 9002)
make dev-frontend
```

访问 [localhost:9002](http://localhost:9002)，用户名/密码：admin/admin

### 使用 Make 命令

```bash
make help          # 查看帮助
make lint          # 代码检查
make test          # 单元测试
make build         # 镜像构建
```

## 部署

详细部署指南请参考 [Docker 部署指南](docs/deployment.md)。

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 了解详情。
