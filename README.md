# Pomelo Orbit

自托管 CI/CD 平台，管理流水线、应用部署和 Traefik 网关。后端使用 Go，前端使用 Vue 3，支持本机和 SSH 环境中的 Docker Compose。

需要 Go、Node.js 22（含 Corepack）和 Task 3。前端在 `web/package.json` 固定 Yarn Modern 版本，使用 `node_modules`。

```sh
go install github.com/go-task/task/v3/cmd/task@v3.45.4
task deps
task web:dev
task webapi:dev
```

常用命令：

| 命令 | 用途 |
| --- | --- |
| `task check` | 类型、格式和 lint 检查 |
| `task test` | 前端与 Go 核心单测 |
| `task build` | 构建 Docker 镜像 |
| `task build cn=1` | 使用国内镜像源构建 |
| `task cli -- env --help` | 环境 SSH 与远程命令 |

单独安装前端依赖：在 `web/` 执行 `corepack yarn install --immutable`。默认使用 npm 官方源；本地需使用国内源时，设置 `COREPACK_NPM_REGISTRY` 和 `YARN_NPM_REGISTRY_SERVER` 为 `https://registry.npmmirror.com`。两套 Docker 构建共用同一份锁文件。

[开发约定](AGENTS.md) · [业务模型与文档](docs/README.md) · [打包与迁移脚本](scripts/README.md)

许可证：MIT。
