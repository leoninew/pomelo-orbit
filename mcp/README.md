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

若 Server 无法建立 session，例如 `uv` 不可用或 `mcp/.env` 不完整，调用方必须将其视为 Server unavailable，不能把任何依赖操作记作成功或继续执行。Server instructions 中会包含当前 MCP 源码/schema 指纹；修改工具后必须建立新的 stdio MCP session，以便工具表和指纹同时更新。

## Development checks

```text
make -C mcp sync
make -C mcp check
make -C mcp run
```

`make -C mcp help` 列出全部入口。Docker 集成测试需显式配置隔离的 Compose workspace 与 project 后执行 `make -C mcp test-docker`。

`runtime_doctor(network_name="traefik")` 可在尚无受管 target 时检查固定外部网络的存在性、driver 和 scope；也保留从已部署 Gateway Application target 派生网络的诊断路径。MCP 不创建 Docker network，也不枚举任意网络。

`orbit_create_gateway` 只创建 Gateway 配置和 Application。`orbit_provision_gateway` 是高层确保型工具：它会唯一复用或创建 `traefik` Gateway、准备 `default` Service、发布当前 Version、部署、等待终态并确认 `traefik` bridge 网络。复用已有 Service 时，不会覆盖 runtime configuration；此时传入非空 `runtime_config` 会失败。

Service 是运行时配置和部署的唯一目标。`orbit_create_service` 必须提交 Version、instance key、runtime configuration 与 exposes；`orbit_update_service_basic` 只修改 Version 和 instance key，`orbit_update_service_configuration` 原子替换 runtime configuration 与 exposes。`orbit_preview_service` 和 `orbit_deploy` 都只接受已保存的 `service_id`。public TCP expose 缺少 Gateway entrypoint 时，`orbit_deploy` 仍会创建 Service deployment 并在响应中返回 warning；它绝不修改或部署 Gateway。必须由操作者显式配置 Gateway 端口并部署 Gateway Service。

`orbit_create_version_component` 向未发布 Version 添加一个仅含名称、镜像、命令和拉取/重启策略的 Component；其余配置通过专用更新工具设置。Version Component 写操作按当前控制面拆分为 `basic`、`runtime`、`ports`、`env`、`mounts`、`dependencies`、`devices` 与 `advanced`。`orbit_update_version_component_devices` 独立替换设备请求，严格校验 driver、count 和 capabilities，且不会修改资源、tmpfs 或 ulimit。`orbit_update_version_component_advanced` 保留资源、tmpfs 与 ulimit 的原子全量替换；`orbit_update_version_component_resources`、`orbit_update_version_component_tmpfs`、`orbit_update_version_component_ulimits` 分别只替换一个高级分组，并保留其余已保存配置。`orbit_update_version_component_connectivity` 已移除，Component 请求也不接受 `networks` 字段；修改工具后需重启 stdio MCP session。

`runtime_compose_ps` 和 `verify_deployment` 默认只返回容器/验证摘要。显式传入 `detail=true` 才会返回原始 Compose 或 inspect 诊断数据。`runtime_http_probe` 仅对目标 Compose `ps` 返回的 running component 执行固定 `docker exec ... curl`，不接受任意 Docker 命令、不返回 HTTP body 或容器错误输出。
