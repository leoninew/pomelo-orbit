# RAGFlow 拆分应用部署计划
最后修改时间: 2026-07-26 20:06:12

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260726-ragflow-split-deployment.md` (`Accepted`)
- Spec: `docs/spec/20260726-ragflow-split-deployment.md` (`Accepted`)
- Flow mode: 严格模式 / `strict`
- Environment baseline: 环境移除任务已完成 MCP 的 Environment-free 契约调整和复核；本任务的实际 MCP 写入与部署必须基于其最终合入版本，且不得恢复 Environment 参数。
- Parallel peer: Task 1 同步实现运行时字段、Credential 引用、env-file 物化、脱敏和 MCP 契约；本任务不修改其拥有的 `mcp/**`、`internal/**`、`proto/**` 或 `sql/**` 文件。
- Live integration gate: Task 1 的 Version / MCP 契约与本任务的非秘密 fixture 已核对。创建实际 Credential、Application、Version 或 Deployment 前，Task 1 必须完成 Verification，证明运行时字段、Credential 引用、env-file 物化和脱敏行为可用。

## Implementation steps

### Step 0: 并行只读预检与目标状态工件

1. 检查当前工作区、Task 1 实际 diff 和 MCP Environment-free 契约；本任务不接管或修改 Task 1 的实现文件。任务一尚未完成 Verification 不阻塞本步骤。
2. 使用 MCP 只读工具读取 Project、Application、Version、Service 和 Gateway；确认不存在相同 application code 的受管对象。任何冲突都停止，不自动收编或删除已有资源。
3. 通过 `orbit_create_gateway` 创建明确配置的 Gateway，并沿用既有 publish / deploy / wait 工具将其部署；以其 Application ID 和 instance key 调用 `runtime_doctor` 检查 Docker context、Compose CLI、Orbit data root 和该 target 派生的共享 bridge `traefik` 网络。其结果必须显式含 `healthy=true` 和 `external_networks` 中的 `traefik`；Gateway 未部署、类型错误或网络不可用时，初始化器以 capability gate 阻断，不调用未受管 Docker CLI。
4. 固定实施参数：`PROJECT_ID`、`INSTANCE_KEY`、RAGFlow HTTP 入口（local 或 public）、监听端口 / Gateway 策略、备份目录、维护窗口。Environment ID 不得出现于任何 MCP 请求、Version 或验证命令。
5. 确认 Elasticsearch 为首版向量引擎，或停止并回到 Task 2 Spec 更新其他引擎的镜像、连接和存储设计。
6. 在仓库中准备脱敏 desired-state payload fixture 和 Runbook：五个 Application 的非秘密字段、符号化 Credential ref、预期 Compose 片段、健康检查和验证断言必须逐项可审阅；不得执行 MCP 写操作或生成实际秘密。
7. 实现 `scripts/ragflow_initialize.py`：默认 `plan` 离线校验和记录操作计划；`check` 通过 stdio MCP 检查工具、参数、资源冲突和任务一能力；`apply` / `verify` 受 Live integration gate 与显式确认约束。运行状态原子持久化，恢复只能使用同一 fixture 摘要和无秘密参数。

### Step 1: 初始化器集成门槛与创建受管 Credential

1. 用初始化器执行 `plan`，校验 fixture 的别名、依赖顺序、非秘密字段、Credential 引用、Expose 变体和禁止项；其 JSON 运行记录是后续 `check` / `apply` / `verify` 的唯一可恢复状态。
2. 使用 Task 1 已提供的组件字段和 MCP 工具执行 `check`；字段、alias、healthcheck、秘密引用结构、同名资源 ID、Gateway target、`runtime_doctor` 网络证明或返回脱敏规则不一致时，初始化器记录 `blocked`，停止 MCP 写入并回到对应任务修正。RAGFlow 候选 healthcheck 随 Version 发布；部署后的实际就绪只能由固定 `runtime_http_probe` 写入无秘密 journal。
3. 待 Task 1 Verification 通过后，以独立、Git 忽略、权限受控的秘密文件作为 `apply` 输入；通过任务一新增的 MCP Credential 工具在同一 Project 创建 `ragflow-mysql`、`ragflow-redis`、`ragflow-minio`、`ragflow-elasticsearch` 四个 `runtime_env` Credential。
4. MCP 响应只检查 ID、名称和类型，不记录 value；不调用导出或明文读取接口验证值。中断后仅可使用同一 fixture 摘要和运行目录 `--resume`。

### Step 2: 创建五个 Application 和 Version

1. 使用 `orbit_bootstrap_application` 或 Application / Version 分步工具，按 Spec 创建 `ragflow-mysql`、`ragflow-redis`、`ragflow-minio`、`ragflow-elasticsearch` 和 `ragflow`，均为 `standard` Application。
2. 为每个 Version 写入钉死镜像、命令、逻辑挂载、healthcheck、`restart_policy=unless-stopped`、秘密引用和非秘密环境变量；不传原始 Compose 的 `include`、`profiles`、`env_file` 或宿主机端口变量。
3. MySQL 使用 `MYSQL_DATABASE=rag_flow` 官方初始化环境变量；Redis command 和所有带秘密的 healthcheck 均使用 `$$VARIABLE` 在容器 shell 中展开。
4. Elasticsearch 必须使用受管 tmpfs、memlock ulimit、逻辑 data mount 和已验证的 memory resource limit。
5. RAGFlow 只使用镜像内置 entrypoint / 配置模板，引用四个依赖 Credential 并设置稳定 alias；不挂载 RAGFlow 源码文件。
6. 对每个 Version 调用 Environment-free preview，检查生成 Compose 不含实际秘密、没有 Environment 字段、workspace / project 名只由 application code 和 instance key 构成；确认后发布。

### Step 3: 按依赖顺序部署

1. 依次调用 `orbit_deploy` 部署 MySQL、Redis、MinIO、Elasticsearch；每次只传 Version ID 和 `INSTANCE_KEY`，不传 Environment ID。
2. 对每次 Deployment 使用 `orbit_wait_deployment` 等待终态；`faulted`、`canceled` 或超时即停止后续部署，不以重试掩盖失败原因。
3. 对成功依赖运行 `runtime_compose_config`、`runtime_compose_ps`、`runtime_container_inspect`、`runtime_compose_logs` 和 `verify_deployment`，确认健康、逻辑 data mount、restart policy、tmpfs / ulimit（Elasticsearch）与共享网络 alias。
4. 仅当四个依赖均为 consistent 且 healthcheck stable 时部署 RAGFlow；使用同一 instance key。
5. 等待 RAGFlow Deployment 和稳定性窗口，核对其 healthcheck、连接日志、脱敏后的 Compose / inspect；在最终 `verify` 中执行 fixture 固定的 `runtime_http_probe`，再检查 RAGFlow HTTP 入口。

### Step 4: 应用层验证与备份基线

1. local 入口仅验证 `127.0.0.1:<RAGFLOW_LOCAL_HTTP_PORT>`；public 入口仅验证已配置的 Gateway host / TLS 路由。验证请求不得在日志中包含 Credential 值。
2. 验证 RAGFlow 已完成数据库初始化，能够建立与 MySQL、Redis、MinIO、Elasticsearch 的连接；失败时收集脱敏的 Deployment 和 Compose logs。
3. 记录五个逻辑数据目录、备份位置、执行者、恢复步骤和首次成功 Deployment ID；备份实现不由 MCP Docker 写工具代替。
4. 运行一次非破坏性的重启恢复验证：重启单个依赖或 RAGFlow 时，确认 `unless-stopped` 和 env-file 引用仍生效；具体动作只在维护窗口且经实施授权后执行。

### Step 5: 实施阶段收尾

1. 汇总 Application、Version、Service、Deployment ID 与验证结论，保证所有输出已脱敏。
2. 确认未创建未选向量数据库、NATS、TEI、DeepDoc、GPU、Sandbox 或原 Compose 的额外端口。
3. 不删除成功部署的数据目录；停止或回滚只经 MCP lifecycle 工具执行，并保留 `remove_volumes=false`。
4. 停在 Implementation / 实现阶段，记录实际写入、命令结果、未运行验证和残余风险，等待人工验收后进入 Verification。

## Files to change

### 必改

- `docs/guides/ragflow-split-deployment.fixture.yaml`：脱敏 desired-state payload fixture，用于与 Task 1 的 Version / MCP 契约进行集成核对；不写入 Credential value、实际 Project ID、资源 ID 或主机路径。
- `docs/guides/ragflow-split-deployment.md`：受管部署 Runbook，定义 fixture 编码、集成门槛、MCP 操作顺序和验证边界。
- `scripts/ragflow_initialize.py` 与 `scripts/test_ragflow_initialize.py`：可重复调用的 stdio MCP 初始化器及其离线/占位能力测试；不修改 MCP Server。
- `mcp/src/pomelo_orbit_mcp/orbit_client.py`、`mcp/src/pomelo_orbit_mcp/tools/orbit.py`：Gateway list/create/get 的受控 Orbit HTTP 映射。
- `mcp/src/pomelo_orbit_mcp/docker_runtime.py`、`mcp/src/pomelo_orbit_mcp/tools/runtime.py`：Gateway-aware `runtime_doctor` 和从 Compose `ps` 派生容器的固定 HTTP Probe。
- `mcp/tests/**`：Gateway mapping、非 Gateway 拒绝、固定 `traefik` network proof、Probe argv 与失败脱敏测试。
- 在 Live integration gate 通过后，才通过 MCP 创建 Orbit 运行数据和受管 workspace。

### 可能触及

- Task 1 的测试 fixture：仅在双方确认后，以非秘密 RAGFlow desired-state case 补充其 MCP / Preview 集成测试；不得改动其实现范围或测试基线。

### 预期不改

- `D:\SourceCodes\opensource\ragflow\**`
- `internal/**`、`proto/**`、`sql/**`、`web/**`，除非前置任务在其自身范围内修改。
- Docker / Compose 生命周期命令、任意宿主机路径或网络操作。

## Verification plan

1. 初始化器 `plan` 对 fixture 可重复产生相同计划摘要，状态文件不包含秘密；`check` 在任务一工具或字段缺失时以 `blocked` 失败，`apply` 不会发出 MCP 写入。
2. 所有 MCP 请求、Application Version、Deployment 和 runtime tools 均不出现 Environment 字段或参数。
3. 五个 Application 的 code、镜像、组件、连接 alias、逻辑挂载、Credential ref、healthcheck、restart、tmpfs / ulimit 与 Task 2 Spec 一致。
4. RAGFlow 仅通过 `ragflow-<dependency>-<component>` alias 访问依赖；没有公开 MySQL、Redis、MinIO 或 Elasticsearch 端口。
5. 每个 Deployment 到达成功终态，`verify_deployment` 返回 `consistent`，容器在稳定性窗口内未退出、unhealthy 或重启。
6. Compose preview、runtime config、inspect、logs、MCP 响应和验证证据不含测试秘密或实际 Credential values。
7. RAGFlow HTTP 入口、数据库初始化和依赖连接通过指定的 local / public 验证；备份目录和恢复责任被记录。

## Blockers

1. 环境移除任务未完成或 MCP 仍接受 / 返回 Environment 参数。
2. Task 1 未完成 Verification，或其 runtime field / Credential / redaction 功能未能通过测试。该项只阻塞实际 MCP 写入与部署，不阻塞本任务的 desired-state 工件、Runbook、镜像调研和集成测试开发。
3. 向量引擎、Project、instance key、入口、healthcheck、资源和备份参数未确认。
4. Docker context 缺少 `traefik` 网络，或 Elasticsearch tmpfs、memlock、逻辑数据目录权限未验证。
5. 初始化器依赖的任务一 MCP 工具或 Component 字段尚未可用；此时 `plan` 可继续，`check` / `apply` / `verify` 必须报告对应 capability。
6. 未创建或未部署 Gateway、`runtime_doctor` 未从其受管 target 明确报告 bridge `traefik` 网络，或 RAGFlow 部署后的固定 HTTP Probe 失败；前两种情况不创建 Version 或 Deployment，Probe 失败时停止后续验收与上线。

## Assumptions

1. Elasticsearch 是首版唯一的向量数据库；其他候选在本 Plan 外。
2. 所有服务在同一 Docker context 和同一 Project 下，以相同 instance key 运行。
3. 环境移除后的 runtime 目标、Compose workspace 和 MCP 接口只以 Application、Service 和 instance key 定位。
4. Credential 轮换通过重新部署依赖服务和 RAGFlow 生效，详细自动化不在本任务范围。

## Risks

1. 前置任务与本任务都涉及 MCP / Deployment 边界，未基于已核对的契约执行会导致运行参数不一致；并行期间以非秘密 fixture 和 Preview 集成测试而非实际部署资源发现该问题。
2. RAGFlow 没有被上游 Compose 固定的 Docker healthcheck；候选健康端点必须先验证，不能将 container running 视为应用成功。
3. Elasticsearch 对内存、tmpfs、ulimit 和宿主机目录权限敏感，Docker Desktop 与 Linux Docker 的结果可能不同。
4. 本机 Docker 高权限用户仍可访问容器环境和受管 workspace，MCP 脱敏不构成宿主机隔离。
5. 秘密文件在本机由初始化器读取，必须在仓库外保存、受操作系统权限保护，并从运行记录和错误文本中排除。

## Rollback

1. 停止 RAGFlow，再按反向依赖顺序停止 Elasticsearch、MinIO、Redis、MySQL；所有 stop 操作使用 MCP，`remove_volumes=false`。
2. 保留 Service workspace 和所有逻辑 data mount，直至备份、恢复或重新部署决策完成。
3. 仅当应用和数据明确为本次隔离部署创建且用户另行授权时，才允许删除 Application、Version 或数据目录；本 Plan 不授权该操作。

## User review notes

用户于 2026-07-26 要求任务一与任务二并行开发。本 Plan 允许立即开展 Task 2 的非秘密 desired-state 工件、Runbook、镜像调研和集成测试开发；实际创建 Credential、Application、Version 和 Deployment 仍须解除 Live integration gate，并由用户明确授权。

用户于 2026-07-26 要求开始任务二 Implementation；视为接受本 Plan。`20260726-orbit-deployment-capability-gaps` 由独立 agent 认领，本任务不修改其文件或任何由其拥有的运行时实现。

用户于 2026-07-26 要求以可重复调用、可日志、可验证的 Python 初始化器承载 RAGFlow 业务初始化；本 Plan 的该部分按 capability gate 支持任务一未完成时的显式占位。

用户于 2026-07-26 要求将 `20260726-runtime-preflight-probe` 合并入本 Plan；其 MCP Gateway 映射、受管网络预检、固定 Probe 与相关测试按本计划的必改范围实施。
