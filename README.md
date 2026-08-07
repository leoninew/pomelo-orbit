# Pomelo Orbit

Pomelo Orbit 是一个面向本地容器环境的轻量级 CI/CD 平台，提供流水线、应用部署和 Traefik 网关管理。

## 技术栈

- 后端：Go、Gin、sqlc、SQLite/MySQL
- 前端：Vue 3、TypeScript、Vite、Tailwind CSS
- 运行环境：Docker、Traefik

## 快速开始

```bash
task install
task dev:backend
task dev:frontend
```

访问 <http://localhost:9020>。

常用命令：

```bash
task check   # 类型检查、格式化和 lint
task test    # 前后端测试
task build   # 构建 Docker 镜像
```

更多产品、架构和操作文档见 [`docs/`](./docs/)。脚本说明见 [`scripts/README.md`](./scripts/README.md)。

## 许可证

MIT License
