# Pomelo Orbit

自托管 CI/CD 平台，管理流水线、应用部署和 Traefik 网关。后端使用 Go，前端使用 Vue 3，支持本机和 SSH 环境中的 Docker Compose。

开发需要 Go 1.25、Node.js 22（含 Corepack）和 Task 3。前端在 `web/package.json` 固定 Yarn Modern 版本，使用 `node_modules`。先参考[开发入门](docs/handbook/getting-started/development.md)准备 `.env.development` 和密钥，再在仓库根目录执行：

```sh
go install github.com/go-task/task/v3/cmd/task@v3.45.4
task deps
```

在一个终端中启动后端：

```sh
task webapi
```

在另一个终端中启动前端：

```sh
task web
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

## 文档站

文档站目标地址为 [GitHub Pages](https://leoninew.github.io/pomelo-orbit/)，对应 `develop` 分支的开发状态。站点从 `docs/handbook/` 中的 Markdown 构建；研发过程和归档文档不进入站点。

修改文档后，在 `docs/` 目录运行：

```sh
corepack yarn install --immutable
corepack yarn build
```

GitHub PR 检查文档构建，文档变更合入 GitHub `develop` 后由 Actions 发布。首次发布需在仓库 Pages 设置中选择 GitHub Actions；本地推送到 Gitee 不会触发 GitHub Actions。

许可证：MIT。
