# Pomelo Orbit MCP

本目录提供本地 stdio MCP Server。Application、Version、Deployment 与 Gateway 的写操作只调用 Pomelo Orbit HTTP API；Docker / Docker Compose 仅用于受管运行时诊断，以及固定、无正文的组件 HTTP Probe。

## Prerequisites

- Python 3.11+ 与 `uv`
- 已启动且可登录的 Pomelo Orbit HTTP API
- 与 Orbit 使用同一 Docker context 和可见 data root 的 Docker CLI

## Configuration

从 `.env.example` 创建本地 `mcp/.env`，填写既有 Orbit 用户名、密码和 API 地址。默认 data root 是仓库的 `data/` 目录，缺失时会创建；仅在 Docker CLI 可见路径不同于默认值时设置 `POMELO_ORBIT_DATA_ROOT` 覆盖。Server 登录后会将短期 JWT 缓存到 `mcp/.mcp-jwt.env`。这两个文件均被 Git 忽略，且绝不能加入 MCP 配置或日志。

可选变量：

- `POMELO_ORBIT_DOCKER_CONTEXT`
- `POMELO_ORBIT_WAIT_TIMEOUT_SECONDS`，默认 `300`
- `POMELO_ORBIT_STABILITY_WINDOW_SECONDS`，默认 `60`
- `POMELO_ORBIT_STABILITY_POLL_SECONDS`，默认 `2`

## Run

```text
uv --directory mcp run pomelo-orbit-mcp
```

该进程使用 stdio transport。将此命令配置为 MCP client 的 server command；不要通过 HTTP 暴露该本地 Server。

## Codex project registration

受信任项目通过仓库根目录的 `.codex/config.toml` 注册 `pomelo_orbit`。打开新的 Codex 会话后，工具列表中应出现该 Server 的工具和 input schema；已有会话不会热加载新增配置。

manifest 不包含 Orbit URL、用户名、密码、JWT、Docker 参数或用户绝对路径。它只启动本目录的已有 stdio entrypoint；`mcp/.env`、进程环境、Docker context 和 data-root 配置仍由本地操作者提供。

若 Server 无法建立 session，例如 `uv` 不可用或 `mcp/.env` 不完整，调用方必须将其视为 Server unavailable，不能把任何依赖操作记作成功或继续执行。

## Development checks

```text
make -C mcp sync
make -C mcp check
make -C mcp run
```

`make -C mcp help` 列出全部入口。Docker 集成测试需显式配置隔离的 Compose workspace 与 project 后执行 `make -C mcp test-docker`。

`runtime_doctor` 可使用已部署的 Gateway Application target 派生并检查固定的 `traefik` 网络；MCP 不创建 Docker network。`runtime_http_probe` 仅对目标 Compose `ps` 返回的 running component 执行固定 `docker exec ... curl`，不接受任意 Docker 命令、不返回 HTTP body 或容器错误输出。
