# internal 后两级目录结构对齐与架构收敛

最后修改时间: 2026-07-09 23:25:16

Review status: Accepted

## Background

上一轮 `internal` 分层 follow-up 主要处理职责边界泄漏：application 中的基础设施细节、HTTP server 中的组合根职责、bootstrap provider 膨胀、runtime model 归属等。该轮迁移已可收尾。

用户进一步指出：对照 `D:\SourceCodes\mywork\best-practices\docs\guides\backend-service-architecture.md`，当前项目在 `internal` 后两级目录结构上仍与文档有明显出入。该问题不应继续追加到上一轮 follow-up，也不应混入 runner/sql error 等局部问题中；本任务作为新的 light / 轻量模式任务单独记录。

这里的"后两级目录结构"指以 `internal/<一级>/<二级>` 为主的目录组织，例如 `internal/application/<domain>`、`internal/api/http/<subpkg>`、`internal/infrastructure/<category>`、`internal/gen/<tool>`、`internal/workflow/<subsystem>` 等。

用户已明确补充：本任务列出的所有目录结构问题均需要处理；不做兼容、不做适配、不以"最小修改"为目标。目录调整要拆到正确边界，而不是保留半截旧结构或新旧并存路径。

此外，上一轮迁移目标可以收尾，但若继续向更严格的架构边界演进，仍有几个不属于目录结构对齐但值得跟踪的候选点（runner concrete 归属、SQL sentinel error 边界、queue/task 膨胀）。这些候选项在本文档后半段单独列出，以便统一跟踪，但不属于目录结构对齐的阻塞项。

## Goal

- 建立专门的 requirement 文档，记录 `internal` 后两级目录结构与架构指南之间的差异，并逐条处理到位。
- 将已识别的目录结构差异全部纳入后续处理范围，不再把其中一部分标为"可接受而不处理"或"仅观察"。
- 后续仍按 light / 轻量模式逐条推进，不一次性重排整个 `internal`，但每条被选中的 issue 必须在本 issue 范围内拆到正确边界。
- 每条目录调整都必须服务职责边界、依赖方向和测试归属，不做"为了长得像文档"的空目录或占位文件，也不做只改一点点的半截迁移。
- 对每条 issue 明确处理结论、实现范围、验证结果和是否仍存在未完成项。
- 继续对照 `backend-service-architecture.md` 中的分层职责：`cmd` 薄入口、`bootstrap` 组合根、`api/http` 入站适配器、`application` 用例编排、`model` 领域核心、`repository` 端口与实现隔离、`infrastructure` 外部系统适配器。
- 把更严格架构收敛的候选问题（runner concrete 边界、SQL sentinel error 边界、queue/task 膨胀）统一记录在本文档中，避免文档无限增长的同时保留跟踪入口。

## Non-goal

- 不在 Requirement 阶段修改产品代码。
- 不一次性把所有目录问题在一个实现阶段全部完成；仍然一条一条处理，避免不可 review 的巨型 diff。
- 不把 `backend-service-architecture.md` 当作逐文件模板；但凡本任务列出的目录差异，后续都需要给出明确处理，不再以"项目特化"作为跳过理由。
- 不引入兼容层、别名、wrapper-only 文件、新旧路径并存或默认值兜底。
- 不做"最小修改"式的表面移动；如果某条 issue 进入实现，应拆到符合架构边界的稳定结构。
- 不修改 API route path、request/response contract、数据库迁移文件或前端行为，除非某条 issue 后续明确扩大范围并被用户接受。
- 不主动启动、停止或重启开发服务器。
- 不执行 `git add`、`git commit`、`git push` 等 Git 写操作。
- 不把上一轮已经完成的 10 个 follow-up issue 重新打开。
- 不在本阶段处理"后续架构收敛候选"的 runner / sql sentinel / queue-task —— 仅记录，后续单独开新任务进入实现。

## User scenarios

- 作为维护者，我希望 `internal` 目录结构能表达清楚的架构职责，而不是迁移后只解决依赖泄漏但目录仍显得随意。
- 作为维护者，我希望所有已识别的目录差异都被逐条处理，而不是长期保留"观察项"。
- 作为维护者，我希望每条目录结构调整都拆到正确边界，不保留兼容路径、不保留适配层、不以"最小修改"为目标。
- 作为维护者，我希望目录结构调整仍然一条一条处理，避免大范围移动导致导入路径、测试和 review 成本可控。
- 作为维护者，我希望上一轮迁移明确收尾，后续更严格的架构收敛问题另开避免旧文档无限增长。
- 作为维护者，我希望每条候选项先有清楚的问题描述、架构依据和处理建议，再决定是否实现。

## Current observed structure notes

基于当前仓库初步观察，`internal` 后两级目录与架构指南存在以下差异。这些差异均纳入后续处理范围；后续每条 issue 进入实现前仍需读取真实调用点和测试，但默认目标是处理到位，而不是跳过。

1. `internal/application` 当前按业务域组织：
   - `internal/application/auth`
   - `internal/application/ci`
   - `internal/application/cd`
   - `internal/application/project`
   - `internal/application/role`
   - `internal/application/settings`
   - `internal/application/user`

   架构指南示例更偏向：
   - `application/port`
   - `application/command`
   - `application/query`
   - `application/workflow`
   - `application/transaction`
   - `application/dto`

   当前 application 的端口接口、DTO、command/query/service 混在各 domain 包内。后续需要将 application 的后两级结构调整为更清晰表达端口、用例、DTO、事务或业务域边界的结构；具体采用 domain-first 下的职责子目录，还是转向职责-first 目录，需要在 issue 内根据真实代码确定，但不能继续保持当前混杂状态不处理。

2. `internal/api/http` 当前结构包含：
   - `server.go`
   - `handler/<domain>`
   - `middleware`
   - `codec`
   - `response`

   架构指南示例还包含：
   - `router.go`
   - `routes.go`
   - `binding`
   - `validator`

   当前 `server.go` 已经从组合根职责中拆出，但 HTTP 层目录仍没有把 router/routes、binding DTO、mapper、validator 等职责充分显式化。后续需要处理 HTTP adapter 后两级目录结构，使 Gin router、route registration、binding/mapper、response/error mapping 的归属更清楚。

3. `internal/infrastructure` 当前结构包含：
   - `database`
   - `config/envfile`
   - `logger/logging`
   - `storage/local`
   - `traefik`
   - `turnstile`

   架构指南示例倾向把第三方 HTTP/RPC client 放到 `infrastructure/external/<provider>`，把 storage/cache/mq/logger/metrics/tracing/mail/lock 等归类明确。当前 `traefik`、`turnstile` 直接位于 infrastructure 一级目录下，需要后续统一 external adapter 分类或给出等价的明确 category，不保留散落在 infrastructure 一级的 provider 包。

4. `internal/gen/sqlc` 当前是 flat 生成目录：
   - `internal/gen/sqlc/*.sql.go`

   架构指南示例是：
   - `internal/gen/sqlc/mysql/<entity>.sql.go`
   - `internal/gen/sqlc/sqlite/<entity>.sql.go`

   当前项目同时存在 MySQL / SQLite 迁移和 driver 配置，后续需要结合 `sqlc.yaml` 与 SQL 目录确认 generated code 是否应按 dialect 或其他边界拆分。该项不再只作为观察项，必须形成明确结构处理结论，并在需要时更新生成配置和调用点。

5. `internal/repository` 当前结构较薄：
   - `internal/repository/option.go`
   - `internal/repository/impl/sqlc`

   架构指南示例中 repository 顶层应承载 repository interfaces、tx、option 等端口定义，impl 下承载 sqlc/memory/fake。当前具体 store/repository interface 散在 application domain 或 sqlc impl 周边，后续需要把 repository 端口与实现结构重新对齐。

6. `internal/queue` 与 `internal/worker` 当前拆成两个一级包：
   - `internal/queue/task`
   - `internal/worker`
   - `internal/worker/handler/<domain>`

   架构指南示例把 queue/consumer/dispatcher/worker/retry/codec/task 都放在 `internal/queue` 下。后续需要处理 worker runtime 与 queue task 的目录关系，统一表达 queue 运行机制、worker 调度、handler 分发、task model/service 的边界。

7. `internal/workflow` 当前主要是：
   - `internal/workflow/activity/ci`
   - `internal/workflow/activity/cd`

   架构指南示例中 workflow 还包含 engine、executor、scheduler、context、registry、trigger、definition、runtime 等。后续需要处理 workflow 后两级目录结构，使 activity、execution、runner、definition/runtime 等职责边界清楚；不创建空目录，但已有 workflow 执行代码不能继续全部沉在 activity domain 下。

8. `internal/common` 当前包含：
   - `errors`
   - `util`
   - `crypto`
   - `constant`
   - `template`
   - `civariable`

   架构指南要求 `common` 必须保持无业务语义。`civariable` 明显带 CI 业务语义，上一轮为避免 application/activity 重复而放入 common。该问题必须处理：CI variable 规则需要迁出 common，进入更明确的业务规则或应用共享边界。

## Candidate issue list — 目录结构对齐

| 编号 | 关注目录 | 差异描述 | 处理要求 | 优先级 | 状态 |
|---|---|---|---|---|---|
| 1 | `internal/common/civariable` | `common` 按指南应无业务语义，但 `civariable` 承载 CI runtime/template variable 规则，存在业务名词和业务规则。 | 必须迁出 common，迁到明确的 CI 规则归属位置；application 与 workflow activity 直接依赖新位置；不保留 wrapper、alias、旧包或适配层。 | 高 | 已实施，检查通过 |
| 2 | `internal/application/<domain>` | 当前 application 按 domain 包组织，端口、DTO、command/query/service 混在同一层级；与指南中的 `port`、`command`、`query`、`dto` 分层差异明显。 | 必须重整 application 后两级结构，明确端口、DTO、用例、事务/编排边界；不以"最小修改"保留混杂结构。可分 domain 逐条处理，但最终所有 application domain 都要收敛。 | 高 | 待办已细化，待实施 |
| 3 | `internal/repository` | repository 顶层端口较薄，具体 store/repository interface 散落在 application；与指南中 repository 端口集中表达有差异。 | 必须重新对齐 repository 端口与 impl 结构，明确哪些接口进入 repository，哪些保留 application port；sqlc impl 不承担端口定义或跨层 runtime model。 | 高 | 待处理 |
| 4 | `internal/worker` 与 `internal/queue` | 当前 worker runtime 独立于 queue；指南示例更倾向 queue 下统一 consumer/dispatcher/worker/task。 | 必须处理 queue/worker 目录关系，统一表达 task model/service、worker runtime、handler dispatch、lease/retry/concurrency 边界；保持行为不变但不保留两套含混结构。 | 中 | 待处理 |
| 5 | `internal/infrastructure/traefik`、`internal/infrastructure/turnstile` | 第三方/external adapter 直接放在 infrastructure 一级目录，未归入 `external/<provider>` 或其他明确 category。 | 必须统一 external adapter 分类，优先考虑 `internal/infrastructure/external/<provider>` 或同等清晰结构；迁移时删除旧路径，不留 wrapper/alias。 | 中 | 待处理 |
| 6 | `internal/api/http` | 当前无单独 `binding` / `validator` / `router.go` / `routes.go` 目录或文件，DTO/mapper 分散在 handler。 | 必须重整 HTTP adapter 结构，明确 router/routes、binding DTO、mapper、response/error、middleware 的归属；不保留职责混杂的 handler 文件。 | 中 | 待处理 |
| 7 | `internal/gen/sqlc` | 当前 sqlc generated files 为 flat 结构，与指南中按 dialect 拆分的示例不同。 | 必须读取并调整生成配置或形成等价清晰结构；如果按 dialect 拆分，需要同步更新 sqlc 配置、生成文件和 import；不保留新旧 generated path 并存。 | 中 | 待处理 |
| 8 | `internal/workflow` | 当前主要沉在 `activity/<domain>`，未形成 execution/runtime/definition/trigger 等清晰边界。 | 必须重整已有 workflow 执行代码的目录结构，明确 activity、execution、runner、runtime/definition 等职责；不创建空占位目录，但已有代码要归位。 | 中 | 待处理 |

## 后续架构收敛候选（非目录结构对齐阻塞项）

以下候选项属于"更严格的架构边界收敛"，不属于目录结构对齐的阻塞项，但统一记录在本文档中避免文档膨胀。每条后续单独开新任务进入实现和验证。

| 编号 | 关注文件 | 问题描述 | 处理建议 | 优先级 | 状态 |
|---|---|---|---|---|---|
| C1 | `internal/application/ci/runner/docker.go`、`internal/application/cd/runner/shell.go`、`internal/workflow/activity/ci/runner.go`、`internal/workflow/activity/cd/runner.go` | application / workflow activity 中仍定义 concrete runner，并直接使用 `os/exec` 调用 Docker 或 shell 命令。按架构指南，外部进程执行更接近 infrastructure / execution adapter，application 应依赖端口而不是承载 concrete implementation。 | 先分析 CI/CD runner 的真实消费关系、测试覆盖和与 workflow activity 的职责边界；目标是保留消费侧端口，将 Docker/Shell concrete runner 移到合适的 infrastructure/execution 包，由 bootstrap 或 activity assembly 注入；不改变执行命令、参数、日志写入或退出码语义。 | 中 | 待处理 |
| C2 | `internal/application/*` 中对 `database/sql` 的 `sql.ErrNoRows` 判断 | 多个 application service 直接判断 SQL sentinel error。严格按分层，application 不应感知 SQL 细节；not-found / conflict 等应由 repository 或端口边界转换为领域/应用错误。 | 逐包分析 repository 接口和 sqlc impl 的错误返回约定；优先选择一个业务域试点，将 not-found 语义从 `sql.ErrNoRows` 收敛为稳定错误或 repository-level helper；避免一次性替换所有 service。 | 中 | 待处理 |
| C3 | `internal/queue/task` 与 HTTP/application/worker 的关系 | `internal/queue/task` 当前同时被 HTTP task handler、application CI/CD task service interface、worker runtime 使用。当前可接受，但若继续膨胀，可能成为第二套 application 层。 | 先观察是否存在业务规则进入 queue/task；如需要处理，应明确 queue/task 是执行机制 service 还是 application-facing task port，再决定是否拆 interface 或移动 task use case。 | 低 | 待观察 |

## C1 初步分析：runner concrete implementation 边界

### 参考架构依据

`backend-service-architecture.md` 中明确：

- `application` 负责编排用例、领域模型、Repository、外部服务端口。
- `application` 不负责第三方 API、Redis、SQL、具体 MQ/EventBus 或外部 SDK 调用细节。
- `infrastructure` 放技术实现和外部系统客户端。
- `workflow`、`queue`、`scheduler` 是执行机制，不是业务核心；真正业务动作仍应调用 application 用例。
- Handler / Consumer / Job / Activity 应作为入口适配器，统一调用 application，避免形成第二套业务语义。

当前 runner concrete implementation 调用 Docker 或 shell 命令，本质是外部进程执行能力。它不像纯业务规则，更接近 infrastructure / execution adapter。

### 当前已知现象

初步复核看到：

- `internal/application/ci/runner/docker.go`：
  - 定义 `DockerRunner` concrete implementation；
  - `DockerRunner.Run` 使用 `exec.CommandContext(ctx, "docker", args...)`。
- `internal/application/cd/runner/shell.go`：
  - 定义 `ShellRunner` concrete implementation；
  - `ShellRunner.Run` 使用 `exec.CommandContext(...)`。
- `internal/workflow/activity/ci/runner.go` 与 `internal/workflow/activity/cd/runner.go` 也存在类似 concrete runner。

这说明上一轮迁移虽然已把 workspace、execution log、Traefik、settings envfile 等明显 infrastructure 细节移出，但 runner concrete implementation 仍有继续收敛空间。

### 初步处理建议

本 issue 不应直接机械搬文件，应先确认：

1. CI application runner 与 CI workflow activity runner 是否完全重复，还是执行语义不同。
2. CD application shell runner 与 CD workflow activity shell runner 是否都需要保留。
3. runner 端口应该由消费侧定义，还是已有消费侧接口可保留。
4. concrete implementation 的目标包应表达外部进程执行语义，例如 `internal/infrastructure/runner`、`internal/infrastructure/docker` 或更贴近现有 storage/traefik 的 execution 子包。
5. 是否需要由 `bootstrap` 统一注入，或 workflow activity service 自身作为执行适配器直接组装。

实现时必须保持：

- Docker command args 不变。
- Shell command cwd/stdout/stderr 行为不变。
- exit code 处理语义不变。
- log writer 注入语义不变。
- 不新增兼容 wrapper、类型别名或新旧 runner 并存路径。

## Initial priority

已按建议第一条优先处理 `internal/common/civariable`，该 issue 已完成实施并通过检查。

理由：

- 它直接触及架构指南中很明确的一条：`common` 必须保持无业务语义。
- 当前 `civariable` 的业务语义非常明显，不是单纯命名差异。
- 它是上一轮迁移为了消除 application/activity 重复而引入的位置，适合作为目录结构对齐的第一条继续收敛。
- 相比全量重排 `application` 或 `repository`，它范围更可控，适合作为 light / 轻量模式第一步。

注意：Issue 1 完成不代表本任务整体完成。所有 candidate issue 都必须在后续逐条处理。

## Acceptance

- [x] 新 requirement 文档已创建，且不复用上一轮 follow-up 文档。
- [x] 文档明确本任务关注 `internal` 后两级目录结构，而不是重新验证上一轮代码拆分。
- [x] 文档列出当前实现与 `backend-service-architecture.md` 的主要目录结构差异。
- [x] 文档明确所有列出的 candidate issue 均需要处理，不再保留"仅观察/可不处理"项。
- [x] 文档记录用户约束：不兼容、不适配、不以"最小修改"为目标。
- [x] Requirement 阶段不修改产品代码。
- [x] 后续每次只选择一条 issue 进入 Implementation，但每条 issue 内部必须拆到正确边界。
- [x] Issue 1 已实施并检查通过。
- [ ] Issue 2-8 仍需逐条完成 Implementation 与 Verification。
- [ ] 后续架构收敛候选 C1-C3 在后续单独任务中逐条推进。

## Completion definition

本任务整体完成条件：

- Issue 1-8 全部完成 Implementation / 实现 与 Verification / 验证。
- 每个 issue 均删除旧职责路径，不保留 wrapper、alias、兼容转发、类型别名或新旧 import 并存。
- 每个 issue 均对照 `backend-service-architecture.md` 说明最终归属理由，而不是只做机械目录改名。
- 每个 issue 均记录实际 diff、预期与实际改动对比、验证命令、范围偏差、风险和未完成项。
- 后端检查通过：
  - `go fmt ./cmd/... ./internal/...`
  - `./bin/golangci-lint fmt ./cmd/... ./internal/...`
  - `./bin/golangci-lint run ./cmd/... ./internal/...`
  - `go vet ./cmd/... ./internal/...`
  - `go test ./cmd/... ./internal/...`
- 未修改 API route path、request/response contract、数据库迁移或前端行为，除非对应 issue 后续显式扩大范围并被用户接受。

## Architecture constraints for all issues

所有后续 issue 必须按以下架构约束判断目录归属：

- 依赖方向保持外层到内层：entrypoint / adapter -> application -> model / ports。
- `cmd` 只做进程启动、配置、信号处理和生命周期入口，不承载业务分支。
- `bootstrap` 作为组合根，可以看见所有层，但只负责注入实现和管理资源，不承载业务规则。
- `api/http`、`queue`、`scheduler`、`workflow activity` 是入站适配器或执行机制，必须调用 application 用例，不应沉淀第二套业务流程。
- `application` 负责用例编排、事务边界、DTO 转换、端口依赖；不得依赖 Gin / gRPC 协议类型、SQLC 生成类型、Redis client、具体 MQ client、第三方 SDK response 或具体 infrastructure 实现。
- `model` 保持业务核心，不依赖协议类型、SQLC 类型、Redis key、SDK response、日志指标实现或外层包。
- Repository 接口与实现必须隔离；接口使用 `model` 或基础类型表达持久化能力，SQLC / DB row / driver 细节只出现在 repository impl 或 database infrastructure 边界。
- `infrastructure` 只做外部系统、本地存储、日志、配置、数据库连接等技术适配，不承载跨业务对象的用例流程。
- `workflow`、`queue`、`scheduler` 只表达"什么时候、以什么机制执行"，真正业务动作仍回到 application。
- `gen` 只隔离生成代码；Proto 生成类型属于协议边界，SQLC 生成类型属于持久化实现细节，mock 生成类型属于测试边界，不能因为同在 `gen` 下就跨层扩散。
- `common` 必须保持无业务语义；凡是带业务名词、依赖业务模型或为规避循环依赖而放入 common 的代码，都必须迁回对应边界。

## Issue dependency / sequencing

默认推进顺序：

1. Issue 1：`internal/common/civariable`，已完成。
2. Issue 2：`internal/application/<domain>`，优先处理，因为 application port / DTO / use-case / transaction 边界会影响后续 repository、workflow、queue/worker 和 HTTP mapper 的归属。
3. Issue 3：`internal/repository`，应在 application 边界明确后处理，避免 repository interface 与 application port 反复移动。
4. Issue 8：`internal/workflow`，依赖 application 用例边界，避免 activity 承载业务编排或 workflow runtime 继续沉在 activity domain 下。
5. Issue 4：`internal/queue` 与 `internal/worker`，依赖 application 用例边界，避免 worker handler 成为第二套 application。
6. Issue 6：`internal/api/http`，可与 application DTO / mapper 边界联动，但不得修改 route path 和 request/response contract。
7. Issue 5：`internal/infrastructure/traefik`、`internal/infrastructure/turnstile`，相对独立，可在不影响 application/repository 边界的前提下穿插处理。
8. Issue 7：`internal/gen/sqlc`，需要结合 repository impl 和 sqlc 配置处理，避免生成路径反复变更。

该顺序是默认策略，不是跳过其他 issue 的理由；如某条 issue 的真实代码分析显示依赖关系需要调整，应先更新本 requirement 再进入实现。

## Per-issue acceptance baseline

每条 issue 的通用验收要求：

- 进入实现前读取相关调用点、测试和配置，确认真实职责边界，不按目录名机械搬迁。
- 删除旧职责路径，不保留 wrapper、alias、兼容转发、类型别名或新旧 import 并存。
- 更新所有调用点、测试 import 和必要的生成配置。
- 不改变业务逻辑、API route path、request/response contract、数据库迁移或前端行为，除非该 issue 明确扩大范围并被用户接受。
- 对照 `backend-service-architecture.md` 说明最终目录归属理由。
- 运行后端检查：`go fmt`、`golangci-lint fmt`、`golangci-lint run`、`go vet`、`go test`。
- Verification 文档记录 requirement alignment、actual diff summary、expected vs actual changed files、acceptance checklist、test results、missed or expanded scope、risks、incomplete items 和 conclusion。

## Issue 2 implementation todo

Issue 2 采用 domain-first 下的职责子目录。以下待办均属于 Issue 2；实施完成前保持未勾选，进入 Verification 后再按实际结果更新。

- [ ] 建立迁移基线：确认旧 `internal/application/{auth,user,role,project,settings,ci,cd,civariable}` import 面，重点覆盖 `internal/api/http/server.go`、`internal/bootstrap/http.go`、`internal/api/http/handler/**`、`internal/workflow/activity/ci/execution.go` 和 application 测试。
- [ ] 迁移 `auth` 到 `internal/application/auth/usecase` 与 `internal/application/auth/dto`，保留 `Service`、`New`、用例方法、持久化 `Repository` 接口和错误语义，测试跟随 usecase。
- [ ] 迁移 `user` 到 `internal/application/user/usecase` 与 `internal/application/user/dto`，保留现有用户创建、更新、状态、删除行为和错误语义，测试跟随 usecase。
- [ ] 迁移 `role` 到 `internal/application/role/usecase` 与 `internal/application/role/dto`，保留角色保存、权限校验、排序和去重行为，测试跟随 usecase。
- [ ] 迁移 `project` 到 `internal/application/project/usecase` 与 `internal/application/project/dto`，保留项目成员、废弃校验和项目依赖计数行为，测试跟随 usecase。
- [ ] 迁移 `settings` 到 `internal/application/settings/usecase`、`internal/application/settings/dto`、`internal/application/settings/port`，将 `EnvStore` 作为 application-owned external capability port，保持 envfile 实现由 bootstrap 注入。
- [ ] 迁移 CI usecase：将当前 CI service/use-case 文件归入 `internal/application/ci/usecase`，保留 `RepositoryStore`、`PipelineExecutionStore`、`Service`、`New`、`NewWithRunner`、`NewExecutionService` 和所有用例方法签名。
- [ ] 迁移 CI DTO：将 `Repository*`、`Webhook*`、`Credential*`、`BuildStage*`、`ArtifactConfig`、`StageOrchestration`、`PipelineTemplate*`、`PipelineRun*`、`PipelineSnapshotDetail`、`ArtifactListInput` 等应用 DTO 归入 `internal/application/ci/dto`。
- [ ] 迁移 CI port / runner / rule：将 `TaskService`、`LogReader`、`ContainerRunner`、`RunOptions` 归入 `internal/application/ci/port`，将 `DockerRunner` 归入 `internal/application/ci/runner`，将 `internal/application/civariable` 归入 `internal/application/ci/rule/civariable`。
- [ ] 迁移 CD usecase：将当前 CD service/application_extra/route 用例归入 `internal/application/cd/usecase`，保留 `Store`、`DeploymentExecutionStore`、`Service`、`New`、`NewWithRunner`、`NewExecutionService` 和所有用例方法签名。
- [ ] 迁移 CD DTO：将 `Application*Input`、`Deployment*`、`ApplicationImportInput`、`ConfigFileInput`、`ApplicationServiceConfig*`、`ApplicationRouteInput`、`Route*Input`、`Traefik*Resp` 等应用 DTO 归入 `internal/application/cd/dto`。
- [ ] 迁移 CD port / runner / rule：将 `TaskService`、`LogReader`、`CommandRunner`、`RouteConfigPublisher`、`RouteCertificateGenerator`、`TraefikRouterClient` 归入 `internal/application/cd/port`，将 `ShellRunner` 归入 `internal/application/cd/runner`，将 compose command 支撑逻辑归入 `rule` 或明确保留在 usecase 支撑文件。
- [ ] 更新外层调用点：HTTP server、bootstrap、HTTP handlers 和 workflow CI activity 全部改用新路径；不改 route path、request/response contract、后台 task payload key 或错误文案。
- [ ] 清理旧路径：删除旧 flat package 文件与空目录，确保不保留 wrapper、alias、类型别名、兼容转发或新旧 import 并存。
- [ ] Issue 2 实现完成后停在 Implementation 阶段，汇报实际 diff、已知风险、未运行检查和建议验收点；等待用户明确要求进入 Verification。

## Open questions

- Issue 2 已基于真实代码分析确定采用 domain-first 下的职责子目录；实施时仍需逐包确认 import cycle、测试归属和旧路径清理结果。
- `repository` 端口应集中到 `internal/repository`，还是部分保留为 application-owned outbound port，需要在 Issue 3 中根据"谁消费，谁定义"和持久化端口边界判断。
- `infrastructure` external adapter 目标结构是否统一为 `internal/infrastructure/external/<provider>`，需要在 Issue 5 中结合 Traefik 与 Turnstile 的实际职责确认。
- `gen/sqlc` 是否按 dialect 拆分，需要在 Issue 7 中读取 `sqlc.yaml` 与 SQL 目录后决定；但该项必须处理并形成结构结论。
- C1 runner concrete implementation 的目标包命名需要在进入实现前根据实际调用关系确认，不在当前 Requirement 阶段提前定死。
- 是否优先从 C1：runner concrete implementation 边界开始处理目录结构对齐候选项。

## Decisions

- 本任务使用 light / 轻量模式。
- 本任务作为新任务，不继续追加上一轮 `20260708-internal-layering-followup-split`。
- 用户已明确：所有列出的目录结构问题均需要处理。
- 用户已明确：不兼容、不适配、不以"最小修改"为目标。
- 本 requirement 作为目录结构对齐 Issue 1-8 与后续收敛候选 C1-C3 的总任务范围基准，状态更新为 `Accepted`；单条 issue 仍按自身状态推进。
- 已按建议优先处理 Issue 1：`internal/common/civariable` 迁出 `common`。
- 目录结构对齐不创建空目录、占位文件或未来预留结构；但已有代码必须迁到正确职责边界。
- 每条 issue 仍单独分析、单独实现、单独验证，避免一次性不可控大迁移。
- C1-C3 属于更严格的架构收敛，不在目录结构对齐的当前处理范围内，后续单独开新任务逐条推进。

## Risk / Assumption

- 架构指南是推荐结构，不是逐字目录模板；但本任务列出的差异已被用户确认均需处理，因此后续不能以"只是项目特化"为理由跳过。
- `application`、`repository`、`queue/worker`、`workflow` 重排可能产生较大导入路径变更，应逐条实施并完整验证。
- Issue 1 已将 `common/civariable` 迁入 `internal/application/civariable`；后续 Issue 2 / Issue 8 重整时，需要继续确认该规则包是否仍符合最终 application 与 workflow 边界。
- `gen/sqlc` 结构调整可能涉及生成配置和大量 import，后续实现前必须先确认生成入口和 SQL 方言边界。
- 当前项目处于活跃开发期，不做兼容层或新旧逻辑并存。
- Runner 边界（C1）看似小，但可能牵涉 CI/CD 执行语义和 workflow activity 测试，不能在未分析调用关系前直接搬迁。
- SQL sentinel error 收敛（C2）可能横跨多个 application service 和 repository impl，建议后续按业务域逐步试点，不一次性全量替换。
- `internal/queue/task`（C3）当前只是观察项，不应为了架构纯度提前引入多余抽象。

## User review notes

- 用户要求：light / 轻量模式。
- 用户要求：用新任务完成。
- 用户指出：`internal` 后两级目录结构中，当前实现与文档还是有很大出入。
- 用户补充：所有问题均需要处理，不兼容、不适配、不"最小修改"。
- 用户要求：开始收拾。

## Issue 1 实施结果

- 新增 `internal/application/civariable/runtime.go`，package 名保持 `civariable`，文件内容保持不变。
- 删除 `internal/common/civariable/runtime.go` 与空目录 `internal/common/civariable`。
- 更新以下 6 处 import：
  - `internal/application/ci/repository.go`
  - `internal/application/ci/runtime_variables.go`
  - `internal/application/ci/snapshot.go`
  - `internal/application/ci/template.go`
  - `internal/application/ci/template_test.go`
  - `internal/workflow/activity/ci/execution.go`
- 未新增 wrapper、alias、兼容转发、类型别名或新旧路径并存。
- 未修改 CI variable 规则实现。
- 未修改 API route path、request/response contract、数据库迁移或前端。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：

- `go fmt`：通过。
- `golangci-lint fmt`：通过。
- `golangci-lint run`：通过，输出 `0 issues.`
- `go vet`：通过。
- `go test`：通过。

关键相关包：

```text
ok   gitee.com/leoninew/PomeloOrbit-go/internal/application/ci
ok   gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci
ok   gitee.com/leoninew/PomeloOrbit-go/internal/test/e2e
```