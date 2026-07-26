# 本地 Docker MCP 部署控制与产品验证规格
最后修改时间: 2026-07-26 14:48:37

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-mcp-local-deployment-control.md`（`Accepted`）
- CD 领域模型: `docs/product/cd-model.md`
- CD 运行时: `docs/architecture/cd-runtime.md`
- 流程: 严格模式 / strict

## Overview

新增顶层 `mcp/` uv + Python 包，提供本地 stdio MCP Server。MCP 是 Pomelo Orbit 的控制面客户端和运行时验证器，不是另一个 Docker 部署器：所有 Application、Version、发布、部署、停止、重启和删除操作通过 Orbit HTTP API；Docker / Docker Compose 只读取与受管 Application + Environment + instance 对应的本机运行时事实。

MCP 复用现有用户名密码登录和 JWT。它通过 Orbit 领域数据、Version Preview、Deployment / Service 记录和 Docker 实态验证产品行为，并在部署成功后观察 60 秒的容器稳定性。开发环境 Turnstile 关闭。

## Design decisions

1. **独立 uv/Python MCP**

   `mcp/` 是独立包，使用 Python MCP SDK 的 stdio transport。依赖由 uv 锁定；首版不把 Python 嵌入 Go 服务，也不提供远程 MCP transport。

2. **Dynaconf 配置和 JWT 缓存**

   Dynaconf 合并进程环境、`mcp/.env` 和仅用于缓存的 `mcp/.mcp-jwt.env`。两份本地文件均被 Git 忽略；`mcp/.env.example` 只提供变量名。

   `OrbitClient` 每次请求前解析缓存 JWT 的 `exp`。Token 有效时直接发送 Bearer header；缺失或过期时，调用既有 `GET /api/auth/csrf-token` 和 `POST /api/auth/login`，将新 JWT 原子写回 `.mcp-jwt.env`。收到 `401` 时删除缓存、重新登录并仅重试原请求一次。用户名、密码、JWT、CSRF Token 和 Cookie 不得出现在日志、异常或 MCP 返回值中。

3. **优先复用并必要时修正 Orbit API**

   MCP 先调用既有 HTTP API；若 API 无法表达已确认能力，则补充或修正 Go 端 HTTP 契约和 usecase，仍不得绕过数据库、Go internal package 或 Deployment 任务队列。

   当前必须修正的 Version 行为：

   - `UpdateVersion` 删除“published version is immutable”限制；Version status 不得阻断编辑。
   - `DeleteVersion` 删除“only unpublished versions can be deleted”限制；仍保留 Version 被 Service 或 Deployment 引用时不可删除的完整性检查。
   - `PublishVersion` 保留为显式状态标记和规格校验入口，但 MCP 的 preview、deploy、edit、delete 不以 `published` 作为前置条件。

4. **应用与 Version 工具形态**

   MCP 同时暴露既有低层资源操作和完整首版规格创建能力：

   - `orbit_create_application` 调用 `POST /api/application`，得到 Orbit 自动创建的初始 draft Version。
   - `orbit_bootstrap_application` 调用既有 `POST /api/application/import`，一次提交 Application、首个 Version、Components 和 Exposes。
   - `orbit_create_version` / `orbit_update_version` 使用 Version 的完整 Components / Exposes 集合；未提供集合保持既有 API 语义，显式空数组表示替换为空。
   - `orbit_publish_version` 仅写入状态，不是其他工具的权限或生命周期门槛。

5. **所有生命周期写操作走 Orbit**

   `orbit_deploy`、`orbit_stop`、`orbit_restart`、`orbit_delete_application`、`orbit_delete_version` 都调用 Orbit API。`stop` 暴露 `remove_volumes`；Application 删除暴露既有 `remove_dir`。MCP 不执行 `docker compose up/down/restart`，从而保留 Deployment、Service、任务和产品日志的一致记录。

6. **即时展示与异步观察分离**

   写工具不要求二次确认，立即返回 `steps`、HTTP method/path、资源 ID 和命令/请求摘要。对于 deploy/stop/restart，MCP 在创建 Deployment 后立即读取 `GET /api/deployment/:deployment_id`，展示 Orbit 已持久化的 `command_text`，不等待 worker 日志。

   `orbit_wait_deployment` 显式轮询 Deployment 至 `ran_to_completion`、`faulted` 或 `canceled`，默认超时 300 秒，可由本地配置调整。它是等待工具，不会无限阻塞。

7. **Docker CLI 仅作受管只读查询**

   Docker 适配器使用 `asyncio.create_subprocess_exec`，不通过 shell。每个工具先通过 Orbit 解析 Application、Environment、instance 和用户成员资格，再按产品规则构造工作目录 `dataRoot/deployment/<app-code>/<env-code>/<instance-key>` 与规范化 Compose project `<app-code>-<env-code>-<instance-key>`。

   允许的命令仅为：

   - `docker context show`、`docker compose version`；
   - `docker compose -p <project> -f docker-compose.yml config`；
   - `docker compose -p <project> -f docker-compose.yml ps --format json`；
   - `docker compose -p <project> -f docker-compose.yml logs --tail <n> [--since <time>] [service...]`；
   - 从上述 `ps` 结果派生的 `docker inspect <container-id>` 与 `docker network inspect <network>`。

   不提供任意 Docker 参数、任意容器 ID 或 Shell 命令工具。运行时数据是本地验证数据，Version Preview、Compose config、inspect 和日志按原始内容返回；唯一例外是 MCP 自身认证凭据。

8. **部署验证与短时稳定性**

   `verify_deployment` 读取 Deployment、Application、Version、Environment、Service、Preview Compose、落盘 Compose config、`ps`、inspect、网络和日志，检查服务名、镜像、端口、网络、Labels 与领域记录的一致性。

   当 Deployment 为 `ran_to_completion` 且操作类型为 deploy 或 restart，验证器以 2 秒间隔观察 60 秒：所有目标容器必须持续 running，不能出现 exited / dead / unhealthy，且 `RestartCount` 相对观察起点不能增加。窗口和轮询间隔为本地配置项。首版不主动发起 HTTP/TCP 探活，也不要求 healthcheck 存在。

   结论值：

   - `consistent`：预期 Compose、领域记录和 Docker 实态匹配，且通过稳定性观察。
   - `drift`：领域记录与 Compose/Docker 实态不匹配。
   - `failed`：Deployment faulted/canceled，或稳定性观察发现退出、失败或重启。
   - `inconclusive`：Docker/Compose 查询不可用、等待超时或数据不足，无法作出上述结论。

## Affected components

| 区域 | 变更 |
|---|---|
| `mcp/pyproject.toml` / `mcp/uv.lock` | uv 包、依赖和可执行入口 |
| `mcp/.env.example` / `mcp/.env` / `mcp/.mcp-jwt.env` / `.gitignore` | Dynaconf 配置样例、私密配置与 JWT 缓存规则 |
| `mcp/src/pomelo_orbit_mcp/server.py` | stdio MCP Server 和工具注册 |
| `mcp/src/pomelo_orbit_mcp/settings.py` | Dynaconf 设置、JWT 缓存校验与原子写入 |
| `mcp/src/pomelo_orbit_mcp/orbit_client.py` | CSRF 登录、JWT 重试和 Orbit HTTP 客户端 |
| `mcp/src/pomelo_orbit_mcp/docker_runtime.py` | 受管的 Docker/Compose 只读命令 |
| `mcp/src/pomelo_orbit_mcp/workspace.py` | 目标解析与 Compose project 命名 |
| `mcp/src/pomelo_orbit_mcp/verification.py` | 部署对账、稳定性观察与结论 |
| `mcp/src/pomelo_orbit_mcp/tools/*.py` | Orbit、运行时、验证工具 |
| `mcp/tests/**` | HTTP、JWT、命令、稳定性、工具结果测试 |
| `internal/application/application/usecase/version.go` | 移除 published 状态对 update/delete 的阻断 |
| `internal/application/deployment/usecase/command.go` | 命令阶段创建或更新 `deploying` Service，并在创建 Deployment 前写入 `service_id` |
| `internal/application/deployment/usecase/deployment_execution.go` | 仅通过 Deployment 已持久化的 `service_id` 读取 Service；缺失关联即标记失败，不创建或补写 Service |
| 对应 Go handler/usecase 测试 | 覆盖 published Version 可修改/删除及引用保护 |

## Interfaces

### 配置

| 变量 | 必需 | 说明 |
|---|---|---|
| `POMELO_ORBIT_URL` | 是 | Orbit HTTP 基地址 |
| `POMELO_ORBIT_USERNAME` | 是 | 既有 Orbit 用户名 |
| `POMELO_ORBIT_PASSWORD` | 是 | 既有 Orbit 密码 |
| `POMELO_ORBIT_JWT` | 否 | 缓存 JWT，由 `.mcp-jwt.env` 持久化 |
| `POMELO_ORBIT_DATA_ROOT` | 否 | Docker CLI 可访问的 Orbit `data` 根目录；默认仓库 `data/`，缺失时创建 |
| `POMELO_ORBIT_DOCKER_CONTEXT` | 否 | 预期 Docker context |
| `POMELO_ORBIT_WAIT_TIMEOUT_SECONDS` | 否 | 等待 Deployment 的默认超时，默认 `300` |
| `POMELO_ORBIT_STABILITY_WINDOW_SECONDS` | 否 | 部署成功后的稳定性窗口，默认 `60` |
| `POMELO_ORBIT_STABILITY_POLL_SECONDS` | 否 | 稳定性轮询间隔，默认 `2` |

### MCP 工具

| 分组 | 工具 |
|---|---|
| Orbit 资源 | `orbit_list_projects`、`orbit_list_environments`、`orbit_list_applications`、`orbit_create_application`、`orbit_bootstrap_application`、`orbit_get_application`、`orbit_delete_application` |
| Version | `orbit_list_versions`、`orbit_get_version`、`orbit_create_version`、`orbit_update_version`、`orbit_publish_version`、`orbit_delete_version`、`orbit_preview_version` |
| Deployment | `orbit_deploy`、`orbit_stop`、`orbit_restart`、`orbit_deployment_status`、`orbit_deployment_logs`、`orbit_wait_deployment` |
| 运行时 | `runtime_doctor`、`runtime_compose_config`、`runtime_compose_ps`、`runtime_compose_logs`、`runtime_container_inspect`、`runtime_network_inspect` |
| 验证 | `verify_deployment` |

写工具统一返回：`operation`、`resource_ids`、`steps`、`request_summary`，并在 Deployment 操作中返回 `deployment_id` 和 `command_text`。运行时工具返回 `working_directory`、`command`、原始本地数据与结构化摘要。认证字段永不返回。

## Technical questions

无阻塞项。Plan 阶段只需根据 Dynaconf 与 MCP SDK 的实际 API 确定缓存文件的原子写入细节、错误类型和测试 fixture，不得改变本规格中的认证、工具、生命周期或验证语义。

## Risks

- Docker Socket 仍是宿主机高权限能力；目标解析与无 shell 执行只缩小 MCP 工具范围，不能替代本机用户权限。
- MCP、Orbit 和 Docker CLI 的 data root 或 Docker context 不一致会造成错误诊断；`runtime_doctor` 必须在其他运行时工具前报告阻断原因。
- 本地数据不脱敏会暴露容器环境变量、Compose 和日志；该行为仅限本地单用户验证场景，不得扩展到远程或多租户模式。
- 允许修改或删除 published Version 可能改变正在运行 Service 所引用的规格；MCP 仍保留引用完整性检查，但产品不再将 status 作为门禁。
- 60 秒稳定性只证明没有短时退出/重启，不等价于业务 HTTP/TCP 可用性。
- JWT 过期后依赖用户名密码重新登录；若登录失败，MCP 必须明确失败而不能使用失效缓存继续请求。

## Alternatives

| 方案 | 为何不采用 |
|---|---|
| 新增 MCP API Token / PAT | 用户明确要求沿用现有认证 |
| MCP 直连数据库或 Go internal package | 绕过 HTTP 契约、认证和产品校验 |
| Docker Python SDK | 与产品实际使用的 Compose CLI 语义不一致 |
| 任意 Docker / Shell 命令工具 | 目标不可审计，容易越出受管应用范围 |
| MCP 直接执行 Compose 生命周期命令 | 绕开 Deployment / Service / task 记录 |
| 将 Version status 作为 MCP 状态机门禁 | 用户明确状态仅为标记，不得阻断操作 |

## User review notes

- 2026-07-26：用户确认沿用 Dynaconf + 环境文件、用户名密码登录和 JWT 缓存；开发环境 Turnstile 关闭。
- 2026-07-26：用户确认优先复用现有接口，能力不足时修正产品实现。
- 2026-07-26：用户确认 Version status 不阻止操作，部署成功后只需短时无失败/无重启验证，本地部署数据不脱敏。
- 2026-07-26：用户确认即时展示操作，暴露删除与 volume 删除能力；所有生命周期写操作仍通过 Orbit。
- 2026-07-26：实际通过 MCP 重建 nginx 时补充 `orbit_list_applications` 只读工具，用于在 Project 范围定位受管 Application 后执行已授权的删除；不改变生命周期写操作边界。
- 2026-07-26：首次部署实测发现 Deployment 缺失 `service_id`；命令阶段先创建带生成 ID 的 Service，再创建引用它的 Deployment。worker 不创建或补写关联，验证器不以运行时推断掩盖该字段缺失。开发阶段不新增历史数据迁移。
- 2026-07-26：`POMELO_ORBIT_DATA_ROOT` 未配置时固定解析为仓库相对 `data/`，并在 MCP 配置加载时创建；显式配置继续作为 Docker CLI 可见路径的覆盖值。
