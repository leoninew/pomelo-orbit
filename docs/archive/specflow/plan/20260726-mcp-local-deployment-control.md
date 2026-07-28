# 本地 Docker MCP 部署控制与产品验证计划
最后修改时间: 2026-07-26 15:13:13

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-mcp-local-deployment-control.md`（`Accepted`）
- Spec: `docs/spec/20260726-mcp-local-deployment-control.md`（`Accepted`）
- CD 领域模型: `docs/product/cd-model.md`
- CD 运行时: `docs/architecture/cd-runtime.md`
- 流程: 严格模式 / strict

## Overview

在顶层新增独立的 `mcp/` uv + Python 包，以本地 stdio MCP Server 方式运行。该 Server 只通过 Orbit HTTP API 执行 Application、Version 和 Deployment 生命周期写操作；它以受限的 `docker` / `docker compose` 只读命令读取运行时事实，并对比 Version Preview、Compose、Service、Deployment 与容器实态。

实现同时修正 Orbit 的 Version 状态语义：`published` 仅为标记，不能阻止更新或删除；Version 被 Service 或 Deployment 引用时仍不可删除。MCP 复用现有用户名密码登录与 JWT，不增加 Token/PAT 鉴权模型。

## Implementation steps

### Step 0: 基线与 HTTP 契约核对

1. 记录当前工具链和本机运行时基线：

   ```text
   uv --version
   docker context show
   docker compose version
   go test ./cmd/... ./internal/... ./sql
   ```

2. 逐项核对现有 HTTP handler、binding、response 与路由，确认下列操作的请求字段、响应 ID、权限错误及异步状态：Project / Environment 列表、Application 创建与导入、Version CRUD / publish / preview、Application deploy / stop / restart / delete、Deployment 状态与日志、Service 列表。
3. 记录 API 缺口。只有既有 API 不能表达已接受的 MCP 工具时，才在 Go HTTP 层补充契约；不得以数据库访问、Go `internal` 直接调用或 Docker 生命周期命令替代。
4. 核对 Deployment `command_text` 的持久化和读取路径，确保 deploy / stop / restart 创建后可立即查询并展示命令摘要。

### Step 1: 修正 Version 状态门禁

1. 修改 `internal/application/application/usecase/version.go`：移除 `UpdateVersion` 对 `published` Version 的不可变校验。
2. 同文件移除 `DeleteVersion` 对 `unpublished` 状态的限制；保留 `CountVersionRuntimeRefs` 检查及原有授权、输入校验和仓储错误转换。
3. 在 `internal/application/application/usecase/version_test.go`（或项目现有等价测试位置）覆盖：
   - 已发布 Version 可修改标签、规格、Components 和 Exposes；
   - 未被引用的已发布 Version 可删除；
   - 被 Service 或 Deployment 引用的 Version 无论状态为何均被拒绝删除；
   - publish 仍执行规格校验并仅更新状态标记。
4. 若 HTTP 层存在把 Version status 当作门禁的重复逻辑，一并删除并补充对应 handler 测试；不改变已有响应格式和权限边界。

### Step 2: 初始化独立 MCP uv 包

1. 创建 `mcp/pyproject.toml` 和 `mcp/uv.lock`，包名为 `pomelo-orbit-mcp`，源码根为 `mcp/src/pomelo_orbit_mcp/`；以 Python MCP SDK 的 stdio transport 提供可执行入口。
2. 固定运行依赖：MCP SDK、Dynaconf、`httpx`；固定开发依赖：`pytest`、`pytest-asyncio`、Ruff、mypy 与 PyYAML 类型桩。依赖版本和 Python 版本约束以 `uv lock` 实际结果为准，提交锁文件。
3. 提供 `mcp/README.md`，仅说明本地启动前提、必需配置变量和如下启动命令，不写入真实凭据：

   ```text
   uv --directory mcp run pomelo-orbit-mcp
   ```

4. 新建 `mcp/.env.example`，只包含变量名与非秘密示例。以 `mcp/.gitignore` 忽略 `.env`、`.mcp-jwt.env`、Python 虚拟环境和测试缓存；不创建真实配置文件。
5. 新建 `mcp/Makefile`，以 `uv` 为唯一执行器，提供 `sync`、`lock`、`lint`、`format-check`、`typecheck`、`test`、显式 Docker 测试、`check` 与 `run` 入口。

### Step 3: 配置、JWT 缓存和认证客户端

1. 在 `mcp/src/pomelo_orbit_mcp/settings.py` 用 Dynaconf 合并进程环境、`mcp/.env` 与 `mcp/.mcp-jwt.env`，集中校验：Orbit URL、用户名、密码、data root、Docker context（可选）、等待超时、稳定性窗口和轮询间隔。
2. 实现 JWT 缓存读取和不验签的 `exp` 时间判断。缓存可用时直接作为 Bearer Token；缺失、格式无效或过期时才登录。
3. 使用既有 `GET /api/auth/csrf-token`、`POST /api/auth/login` 流程取得 JWT，以同目录临时文件再原子替换的方式写入 `mcp/.mcp-jwt.env`。
4. Orbit 请求收到 `401` 时删除缓存、重新登录，并只重试原始请求一次。登录、缓存和所有异常路径均不得输出用户名、密码、JWT、CSRF Token 或 Cookie。
5. 对应测试覆盖有效/失效 JWT、缓存写入、登录成功/失败、一次性 `401` 刷新重试与禁止泄露认证字段。

### Step 4: 实现强类型 Orbit HTTP API 客户端

1. 在 `mcp/src/pomelo_orbit_mcp/orbit_client.py` 定义输入、响应和 API 错误模型，将现有 HTTP JSON 契约收敛为异步客户端方法；保留原始 API 错误的安全摘要、HTTP method/path 和资源 ID。
2. 覆盖 Project / Environment 查询、Application 创建/导入/读取/删除、Version 列表/读取/创建/更新/publish/delete/preview、Deployment 创建/读取/日志/等待所需查询，以及 Service 查询。
3. `create_application` 映射到 `POST /api/application`，并返回 Orbit 自动生成的 initial draft Version；`bootstrap_application` 映射到 `POST /api/application/import`，一次提交 Application、首 Version、完整 Components 和 Exposes。
4. Version 创建/更新以完整 Components / Exposes 集合工作：字段省略时保留现有 HTTP API 语义，显式空数组代表替换为空。所有写操作的请求摘要必须剔除认证字段。
5. deploy / stop / restart 创建 Deployment 后立即读取该 Deployment，返回持久化的 `command_text`；不在写工具中等待 worker 完成。

### Step 5: 实现受管范围的 Docker / Compose 只读适配器

1. 在 `mcp/src/pomelo_orbit_mcp/workspace.py` 通过 Orbit 数据与用户权限解析 Application、Environment、instance，生成唯一工作目录：`<data_root>/deployment/<app-code>/<env-code>/<instance-key>`，并生成 Compose project：`<app-code>-<env-code>-<instance-key>`。
2. 在 `mcp/src/pomelo_orbit_mcp/docker_runtime.py` 使用 `asyncio.create_subprocess_exec`，禁止 shell 和任意参数透传。支持且只支持：
   - `docker context show`、`docker compose version`；
   - `docker compose -p <project> -f docker-compose.yml config`；
   - `docker compose -p <project> -f docker-compose.yml ps --format json`；
   - `docker compose -p <project> -f docker-compose.yml logs --tail <n> [--since <time>] [service...]`；
   - 从前述 `ps` 结果得到的容器 ID 所对应 `docker inspect`，以及受管 Compose 网络的 `docker network inspect`。
3. `runtime_doctor` 检查 Docker context、Compose 可用性、data root 和工作目录；任何运行时工具都返回工作目录、已执行的无秘密命令和结构化摘要。
4. 拒绝未解析的 Application / Environment / instance、任意容器 ID、任意网络名、路径穿越、超出限制的日志行数和不受管的 service 名。

### Step 6: 注册 MCP 工具和统一响应协议

1. 在 `mcp/src/pomelo_orbit_mcp/server.py` 创建 stdio Server，在 `mcp/src/pomelo_orbit_mcp/tools/` 按 Orbit、Version、Deployment、runtime、verification 分组注册以下工具：
   - `orbit_list_projects`、`orbit_list_environments`、`orbit_list_applications`、`orbit_create_application`、`orbit_bootstrap_application`、`orbit_get_application`、`orbit_delete_application`；
   - `orbit_list_versions`、`orbit_get_version`、`orbit_create_version`、`orbit_update_version`、`orbit_publish_version`、`orbit_delete_version`、`orbit_preview_version`；
   - `orbit_deploy`、`orbit_stop`、`orbit_restart`、`orbit_deployment_status`、`orbit_deployment_logs`、`orbit_wait_deployment`；
   - `runtime_doctor`、`runtime_compose_config`、`runtime_compose_ps`、`runtime_compose_logs`、`runtime_container_inspect`、`runtime_network_inspect`；
   - `verify_deployment`。
2. 写工具统一返回 `operation`、`resource_ids`、按发生顺序的 `steps`、无秘密的 `request_summary`；Deployment 写工具额外返回 `deployment_id` 和 `command_text`。不添加确认步骤。
3. 只读运行时工具返回目标范围、工作目录、实际命令、原始本地数据及结构化摘要。认证字段、Cookie 和 CSRF 值在成功、错误与日志结果中均不能出现。
4. 对非 `standard` Application、缺失 Environment、目标不属于登录用户、无法确定 instance 的情况返回清晰的范围或参数错误，禁止降级为任意 Docker 查询。

### Step 7: 实现 Deployment 等待与产品一致性验证

1. `orbit_wait_deployment` 以可配置的默认 300 秒总超时轮询 Deployment，直到 `ran_to_completion`、`faulted` 或 `canceled`；超时返回带最后状态的明确结果，绝不无限阻塞。
2. 在 `mcp/src/pomelo_orbit_mcp/verification.py` 汇集 Application、Version、Environment、Service、Deployment、Preview Compose、落盘 Compose config、`ps`、inspect、网络和日志，比较服务名、镜像、端口、网络、Labels 与领域记录。
3. 当 deploy/restart 的 Deployment 为 `ran_to_completion` 时，以可配置的默认 60 秒窗口、2 秒间隔观察目标容器。容器退出、`dead`、`unhealthy` 或 `RestartCount` 相比观察起点增加，立即判为失败；首版不添加 HTTP/TCP 探活。
4. 仅返回四种结论：`consistent`、`drift`、`failed`、`inconclusive`，同时提供逐层证据和差异原因。Docker 不可用、数据不足或等待超时必须为 `inconclusive`，不能伪报成功。
5. 首次 deploy 命令必须先创建带生成 ID、`deploying` 状态和目标 Version 的 Service，再创建引用该 `service_id` 的 Deployment 并入队；worker 只能读取该关联，缺失时标记 Deployment 失败。验证器不得以 Service 列表推断替代缺失关联。开发阶段不新增历史数据迁移。

### Step 8: 测试与本机集成验证

1. Python 单元测试覆盖设置校验、JWT、登录刷新、HTTP 请求映射和错误、安全响应序列化、工作目录/Compose project 解析、白名单命令构造、目标边界、Deployment 状态机与稳定性结论。时间相关逻辑使用可控 clock / sleep，不让单元测试等待 60 秒。
2. 增加显式 Docker 标记的集成测试，验证 `runtime_doctor`、Compose `config` / `ps` / logs、inspect 与网络读取；默认测试命令不得运行这些测试。测试只使用临时的、带唯一前缀的 Compose fixture，不调用 `up` / `down` 等运行时生命周期写操作。
3. 实现阶段依次运行并记录结果：

   ```text
   go fmt ./cmd/... ./internal/... ./sql
   go vet ./cmd/... ./internal/... ./sql
   go test ./cmd/... ./internal/... ./sql
   make -C mcp check
   POMELO_ORBIT_RUN_DOCKER_TESTS=1 POMELO_ORBIT_DOCKER_TEST_WORKSPACE=<workspace> POMELO_ORBIT_DOCKER_TEST_PROJECT=<project> make -C mcp test-docker
   ```

4. 最后一条只在 Docker 可访问、Orbit data root / context 一致且本机已明确启用 Docker 集成测试时执行；否则记录为未运行的环境依赖，不以模拟结果替代。

### Step 9: 实现收尾

1. 抽样验证一条完整流程：创建或导入 `standard` Application、维护 Version、preview、deploy、读取 Deployment command / logs、等待、运行时查询和 `verify_deployment`；删除类工具只在隔离测试数据上验证。
2. 检查 `git diff`，确认不触及用户已有的 `web/` 工作区改动和本需求外的领域重构。
3. 汇报实际改动、命令结果、未运行的 Docker 集成检查、风险与任何 API 契约补充，并停在 Implementation / 实现阶段等待用户指示进入 Verification / 验证。

## Files to change（预期）

### 必改

- `.gitignore`
- `internal/application/application/usecase/version.go`
- `internal/application/application/usecase/version_test.go` 或现有同类测试位置
- `internal/application/deployment/port/port.go`
- `internal/application/deployment/usecase/command.go`
- `internal/application/deployment/usecase/command_test.go`
- `internal/application/deployment/usecase/deployment_execution.go`
- `internal/application/deployment/usecase/deployment_execution_test.go`
- `internal/repository/deployment.go`
- `internal/repository/impl/sqlc/deployment/repository.go`
- `sql/query/deployment/deployment.sql`
- `internal/gen/sqlc/deployment/deployment.sql.go`
- `mcp/pyproject.toml`
- `mcp/uv.lock`
- `mcp/Makefile`
- `mcp/.env.example`
- `mcp/README.md`
- `mcp/src/pomelo_orbit_mcp/__init__.py`
- `mcp/src/pomelo_orbit_mcp/server.py`
- `mcp/src/pomelo_orbit_mcp/settings.py`
- `mcp/src/pomelo_orbit_mcp/orbit_client.py`
- `mcp/src/pomelo_orbit_mcp/workspace.py`
- `mcp/src/pomelo_orbit_mcp/docker_runtime.py`
- `mcp/src/pomelo_orbit_mcp/verification.py`
- `mcp/src/pomelo_orbit_mcp/tools/**`
- `mcp/tests/**`

### 可能触及

- `internal/api/http/handler/application/**`
- `internal/api/http/handler/deployment/**`
- `internal/api/http/binding/**`
- `internal/api/http/response/**`
- `internal/api/http/routes/**`
- 上述 API 相关测试

仅当 Step 0 证明既有 HTTP 契约无法表达已接受的 MCP 能力时修改这些 Go HTTP 文件；新增接口必须保持既有认证与 Project 成员资格校验。

### 预期不改

- `web/**`
- 数据库 schema、migration
- Compose renderer 和队列执行机制
- Docker / Docker Compose 生命周期写操作
- 新的 MCP Token、PAT、refresh token 或远程 Docker transport

## Verification plan

1. 已发布且未引用的 Version 可以通过 HTTP 更新和删除；被引用 Version 仍被拒绝。
2. `uv --directory mcp lock --check` 可复现依赖，stdio MCP Server 可由入口命令启动。
3. 有效 JWT 不触发登录；过期或首个 `401` 仅登录一次并重试一次；所有工具响应、日志与异常中不含认证凭据。
4. 所有生命周期写工具只出现 Orbit HTTP 请求摘要；运行时适配器不含 `up`、`down`、`restart`、`rm` 或任意 shell 参数入口。
5. 运行时查询必须先解析 Orbit 管理目标，且返回的 Compose 目录和 project 符合 `<data_root>/deployment/<app-code>/<env-code>/<instance-key>` 和 `<app-code>-<env-code>-<instance-key>`。
6. Deployment 写工具即时返回 Deployment ID 与 `command_text`；等待工具正确处理 terminal 成功、失败、取消和超时。
7. `verify_deployment` 对一致、漂移、失败、数据不足或 Docker 不可用分别返回规定结论；稳定性观察能识别退出、unhealthy 和 restart count 增长。
8. Go 检查、Python 单元测试、依赖锁定检查通过；Docker 标记集成测试的实际运行状态在 Verification / 验证阶段明确记录。

## Blockers

无。Docker 标记集成测试依赖本机 Docker 可访问，并要求 MCP、Orbit 与 Docker CLI 使用相同 context 和 data root；该条件不足时不会阻塞 Python 单元测试或 Go 改动，但会阻止对真实 Docker 运行时的最终验证。

## Assumptions

1. 现有 Orbit HTTP API 的身份校验和 Project 成员资格检查适用于 MCP 的 Bearer JWT 请求。
2. 开发环境 Turnstile 保持关闭，用户名密码登录流程无需额外挑战。
3. `POMELO_ORBIT_DATA_ROOT` 可覆盖 Docker CLI 可见的逻辑 data root；未配置时使用仓库 `data/` 并创建。覆盖值不得指向宿主机或容器内不一致的路径。
4. 本地部署数据允许原样返回；仅 MCP 自身认证材料必须隐藏。
5. 默认稳定性窗口只验证容器短时运行稳定，不等同应用层可用性。

## Risks

1. Docker Socket/CLI 的本机权限高于 Orbit JWT 权限；目标范围限制只能约束 MCP 工具设计，不能隔离本机用户。
2. Version 可编辑或删除的状态语义放宽后，运行中的服务仍可能引用该 Version；引用检查仍需作为删除边界，验证输出应显示可能的规格漂移。
3. 现有 API 若缺少 instance、workspace 或 Deployment command 的读取字段，补充契约会带来后端测试和兼容性工作。
4. 60 秒观察会使实际验证工具耗时；单元测试必须模拟时间，实际 MCP 调用应清楚展示正在观察的步骤。
5. 本地未脱敏的 Compose、环境变量和日志仅适合单机开发验证，不能作为远程或多租户 MCP 的默认行为。

## Rollback

1. 删除独立 `mcp/` 包和对应 `.gitignore` 条目，即可撤回 MCP 功能；不会产生数据库迁移或 Docker 生命周期副作用。
2. 还原 `version.go` 的两处 status 门禁，并同步还原相关 Go 测试，即可恢复 published Version 的旧限制。
3. 若新增 HTTP API 契约，按独立、可审查的 handler/usecase 测试回退；不得留下 MCP 依赖但无实现的路由。
4. 测试用 Compose fixture 必须隔离命名和临时目录，清理失败时只允许清理已验证属于该 fixture 的资源。

## Out of scope follow-ups

| 项 | 说明 |
|---|---|
| 远程 Docker / SSH / Kubernetes | 另立需求，重新设计 host 权限与凭据边界 |
| Docker Compose 生命周期写操作 | 需定义 Deployment / Service 漂移记录和恢复语义后再评估 |
| MCP Token / PAT / refresh token | 用户明确沿用现有用户名密码 + JWT 认证 |
| Gateway、证书、public TCP、高级挂载 | 首版仅 `standard` Application，后续按模型补充 |
| HTTP/TCP 应用探活 | 当前只做容器状态与重启稳定性观察 |

## User review notes

- 2026-07-26：用户要求进入 Plan / 计划，Spec 标记为 `Accepted`。
- 本 Plan 为 `Draft`；确认后才进入 Implementation / 实现，不在计划阶段修改产品代码。
- 2026-07-26：实际 MCP 操作需要在 Project 范围定位旧 nginx Application，补充只读 `orbit_list_applications`；仍不增加 Docker 生命周期写操作。
- 2026-07-26：首次部署关联的 `service_id` 由命令阶段确定；移除 worker 对 Deployment 的关联补写，不增加历史数据迁移。
- 2026-07-26：数据根未配置时以 `mcp/` 的父目录为基准解析 `data/`，并由 `Settings.load` 创建；覆盖配置保持原有优先级。
