# 开发入门

本指南面向从源码参与开发的读者，对应 GitHub `develop` 分支。命令默认在仓库根目录执行；前后端启动命令需要分别在两个终端中运行。

## 前置条件

- Go 1.25、Node.js 22、Corepack 和 Task 3。
- 网络可以访问 Go 模块与 npm 源。`task deps` 会安装锁定的前端依赖和项目固定版本的 Go 工具。
- Python 3 可用于生成下面的本地密钥；也可以使用其他安全方式生成 Fernet 密钥。

安装项目使用的 Task 版本：

```sh
go install github.com/go-task/task/v3/cmd/task@v3.45.4
```

确保 Go 工具的安装目录位于 `PATH`，然后在仓库根目录运行：

```sh
task deps
```

## 配置开发环境

`task webapi` 使用 `development` profile，因此会读取根目录的 `.env.development`。如果该文件不存在，可以从 `.env.example` 创建；已有文件请保留，不要覆盖。

```sh
test -e .env.development || cp .env.example .env.development
```

PowerShell：

```powershell
if (-not (Test-Path -LiteralPath .env.development)) {
  Copy-Item -LiteralPath .env.example -Destination .env.development
}
```

在 `.env.development` 中设置 `POMELO_ORBIT_JWT__SECRET_KEY`。该值必须是有效的 Fernet 密钥；例如使用 Python 生成一个随机密钥：

```sh
python -c "import base64, os; print(base64.urlsafe_b64encode(os.urandom(32)).decode())"
```

将输出保存为配置值，不要提交、共享或写入文档。该密钥也用于保护已有凭据；已有数据的环境不要随意更换密钥。

配置依次由 `configs/config.yaml`、可选的 `configs/config.development.yaml`、`.env.development`、操作系统环境变量和根目录 `overwrite.env` 覆盖，后者优先级更高。系统设置写入的 `overwrite.env` 可能使环境变量中的值不生效。

## 启动与验证

在第一个终端启动后端：

```sh
task webapi
```

在第二个终端启动前端：

```sh
task web
```

前端开发服务默认使用 `http://localhost:9020`；如果端口被占用，以终端实际输出的地址为准。后端默认监听 `127.0.0.1:9021`，可以访问 `http://127.0.0.1:9021/api/health`。成功时接口返回 HTTP 200，响应中的 `status` 为 `ok`。

健康检查只说明 HTTP 服务响应，不能证明流水线、SSH 或应用部署正常。登录方式和应用部署配置需要按照实际环境准备，不应将测试数据中的账号当作生产凭据。

## 常用检查

在仓库根目录运行：

```sh
task check
task test
```

`task check` 检查前端和 Go 的类型、格式与 lint；`task test` 运行前端及 Go 核心单测。更多验证入口见[项目开发约定](https://github.com/leoninew/pomelo-orbit/blob/develop/AGENTS.md)。

## 常见问题

- **提示 JWT 密钥缺失或无效**：检查 `.env.development` 的 `POMELO_ORBIT_JWT__SECRET_KEY`，以及是否被环境变量或 `overwrite.env` 覆盖。
- **找不到 Task 或 Go 工具**：检查 Go 安装目录是否在 `PATH`，并确认已经在仓库根目录执行 `task deps`。
- **前端无法访问 API**：检查后端是否启动、健康检查是否成功，以及前端终端输出的地址。前端开发代理默认指向 `127.0.0.1:9021`。
