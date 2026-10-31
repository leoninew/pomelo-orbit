# 本地开发

从源码修改、调试 Pomelo Orbit，或自行构建镜像。直接运行发布产物见[自托管运行](./installation.md)。以下命令使用 Bash；获取源码后，默认在仓库根目录执行。

## 前置条件

- Bash、Go 1.25、Node.js 22、Corepack 和 Task 3；使用的工具与路径需属于同一开发环境，不混用 Windows 原生与 WSL 工具链。
- Git 用于获取源码和管理版本；调试构建、部署功能时，还需准备 Docker Engine 和 Compose。
- 网络可以访问 Go 模块与 npm 源。`task deps` 会安装锁定的前端依赖和项目固定版本的 Go 工具。

## 获取源码与依赖

```bash
git clone --branch develop --single-branch https://github.com/leoninew/pomelo-orbit.git
cd pomelo-orbit
```

安装项目使用的 Task 版本：

```bash
go install github.com/go-task/task/v3/cmd/task@v3.45.4
```

确保 Go 工具的安装目录位于 `PATH`，然后在仓库根目录运行：

```bash
task deps
```

## 配置开发环境

`task webapi` 使用 `development` profile，因此会读取根目录的 `.env.development`。如果该文件不存在，可以从 `.env.example` 创建；已有文件请保留，不要覆盖。

```bash
test -e .env.development || cp .env.example .env.development
```

在 `.env.development` 中设置 `POMELO_ORBIT_JWT__SECRET_KEY`，格式与生成方法见[实例主密钥](../operations/secrets.md#实例主密钥)。开发实例也应保留原密钥，以便解密已保存的凭据。

配置依次由 `configs/config.yaml`、可选的 `configs/config.development.yaml`、`.env.development`、操作系统环境变量和根目录 `overwrite.env` 覆盖，后者优先级更高。系统设置写入的 `overwrite.env` 可能使环境变量中的值不生效。

## 启动与验证

在第一个终端启动后端：

```bash
task webapi
```

在第二个终端启动前端：

```bash
task web
```

前端开发服务默认使用 `http://localhost:9020`；如果端口被占用，以终端实际输出的地址为准。后端默认监听 `127.0.0.1:9021`，可以访问 `http://127.0.0.1:9021/api/health`。成功时接口返回 HTTP 200，响应中的 `status` 为 `ok`。

健康检查只说明 HTTP 服务响应，不能证明流水线、SSH 或应用部署正常。登录方式和应用部署配置需要按照实际环境准备，不应将测试数据中的账号当作生产凭据。

## 构建镜像

在源码根目录执行：

```bash
git rev-parse HEAD
task build TAG=dev
```

构建产物为 `pomelo-orbit:dev`。Go 和前端编译在 Docker 内完成；使用国内构建源时运行 `task build TAG=dev cn=1`。需要部署自建镜像时，按[自托管运行](./installation.md)准备运行目录，并将 Compose 的 `image` 改为这个标签。

记录源码提交与镜像 ID；升级使用新的镜像标签，并保留旧版本。

## 预览使用手册

使用手册在 `docs/handbook/`，VitePress 配置与主题在 `docs/.vitepress/`。在独立终端运行：

```bash
cd docs
corepack yarn install --immutable
corepack yarn dev
```

打开 `http://127.0.0.1:5173/pomelo-orbit/`，修改 Markdown 后页面自动刷新。操作顺序使用步骤列表；需要说明资源关系、作用域或跨组件交互时，使用 `mermaid` 代码块绘图，避免重复相邻步骤或表格。图示保持窄屏可读。

提交前在 `docs/` 运行 `corepack yarn build`，检查构建和站内链接。

## 常用检查

在仓库根目录运行：

```bash
task check
task test
```

`task check` 检查前端和 Go 的类型、格式与 lint；`task test` 运行前端及 Go 核心单测。更多验证入口见[项目开发约定](https://github.com/leoninew/pomelo-orbit/blob/develop/AGENTS.md)。发布包构建见[打包脚本说明](https://github.com/leoninew/pomelo-orbit/blob/develop/scripts/README.md)。

## 常见问题

- **提示 JWT 密钥缺失或无效**：检查 `.env.development` 的 `POMELO_ORBIT_JWT__SECRET_KEY`，以及是否被环境变量或 `overwrite.env` 覆盖。
- **找不到 Task 或 Go 工具**：检查 Go 安装目录是否在 `PATH`，并确认已经在仓库根目录执行 `task deps`。
- **前端无法访问 API**：检查后端是否启动、健康检查是否成功，以及前端终端输出的地址。前端开发代理默认指向 `127.0.0.1:9021`。
