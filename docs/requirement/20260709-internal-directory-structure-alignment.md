# Requirement: internal 后两级目录结构对齐与架构收敛

状态：Accepted

## Scope

本任务对齐 `internal` 的后两级目录结构和真实职责边界。项目处于活跃开发期：不保留兼容层、别名、wrapper、转发包、新旧路径并存或空目录。不得修改 API route、请求/响应契约、数据库 migration 或前端行为，除非获得单独明确批准。

## Architecture constraints

- 依赖方向保持：entrypoint / adapter → application → model / ports。
- `bootstrap` 是组合根，只装配实现和管理资源，不承载业务规则。
- `api/http`、`queue`、`scheduler` 是入站适配器或执行机制；业务动作必须调用 application usecase。
- `application` 负责编排用例、DTO 转换和端口依赖，不依赖协议类型、SQLC 类型、第三方 SDK response 或具体 infrastructure 实现。
- `repository` root 表达持久化 port，具体 SQLC/SQLX 细节只位于 implementation。
- `infrastructure` 承担外部系统、本地存储、日志、配置和数据库等技术适配。
- `common` 不得承载业务语义。
- 不为匹配参考目录而创建未被真实代码消费的空 package。

## Issue status

| 编号 | 关注目录 | 最终结果 | 状态 |
| --- | --- | --- | --- |
| 1 | `internal/common/civariable` | CI variable 规则迁至 `internal/application/ci/rule/civariable`；旧 common 路径已删除。 | 已实施并验证完成 |
| 2 | `internal/application/<domain>` | auth/user/role/project/settings/CI/CD 均采用 domain-first 的 `usecase`、`dto`、`port`、`rule`、`runner` 职责目录。 | 已实施并验证完成 |
| 3 | `internal/repository` | 持久化 port 收敛到 repository root；adapter 按 SQLC/SQLX 实际技术归类。 | 已实施并验证完成 |
| 4 | `internal/queue` 与 `internal/worker` | task model/service、worker runtime、handler dispatch 收拢至 queue boundary。 | 已实施并验证完成 |
| 5 | `internal/infrastructure/{traefik,turnstile}` | concrete provider adapter 归入 `infrastructure/external/<provider>`。 | 已实施并验证完成 |
| 6 | `internal/api/http` | handler 的纯字段转换归入同包 domain-local mapper；HTTP 协议编排保留 handler。 | 已实施并验证完成 |
| 7 | `internal/gen/sqlc` | SQLC output 固定为 `internal/gen/sqlc`，使用受控 sqlc v1.30.0。 | 已实施并验证完成 |
| 8 | `internal/workflow` | CI/CD execution 收敛到 application usecase；queue handler 仅适配 task payload；完整 workflow tree 已删除。 | 已实施并验证完成 |

## Issue 8 implementation result

### Final execution flow

```text
application command use case
  → queue task
  → queue worker handler
  → application execution use case
  → repository / workspace / execution-log / runner port
```

- CI `ExecutePipelineRun`、stage DAG、状态推进、执行日志、artifact、credential rewrite 和 cancellation guard 位于 `internal/application/ci/usecase`。
- CD deploy/restart/stop、配置渲染、workspace materialization、Compose command 执行、deployment log 与状态补偿位于 `internal/application/cd/usecase`。
- CI queue handler 使用 application CI DTO；CI/CD handler 均只做 task payload decode、字段校验和一次 application dispatch。
- Worker bootstrap 构造 application execution services，并注入既有 application Docker/Shell runner 与 local execution-log store。
- `internal/workflow/activity/{ci,cd}` 以及整个空的 `internal/workflow` tree 已删除；未保留 wrapper、alias、forwarding constructor、旧 DTO 或兼容 import。
- 未创建 workflow runtime、definition、trigger、engine、scheduler 或 registry 空目录：当前实际异步运行机制是 queue worker，pipeline/deployment 定义由 application/model/repository 表达。

### Behavior retained

- CI/CD task type、payload key、handler validation/error 文案和 queue lease/retry/concurrency 未变。
- CI snapshot/variable resolution、DAG 并发、stage cancel、log/artifact、credential environment rewrite 和 Docker command 行为未变。
- CD deploy/restart/stop 的差异化状态补偿、Compose 参数、workspace/log 路径、Liquid/route rendering、`init.sh` line ending 与 chmod 行为未变。
- API route、request/response contract、migration、schema 和前端行为未变。

## Independent follow-up work

| 编号 | 内容 | 状态 |
| --- | --- | --- |
| C1 | Docker/Shell runner 与 local workspace 收敛至 infrastructure adapter；application 仅依赖 workspace/command port，CD status/log 不再 direct `os/exec`。 | 已实施并验证完成 |
| C2 | SQLC/SQLX adapter 将 `sql.ErrNoRows` 映射为 `repository.ErrNotFound`；application 仅判断 repository 语义错误。 | 已实施并验证完成 |
| C3 | CI/CD application 使用 typed dispatch port；queue adapter 独占 task type、JSON payload 与 `queue/task` implementation contract。 | 已实施并验证完成 |

## Acceptance

- [x] Issue 1-8 与 C1-C3 均完成 Implementation 与 Verification。
- [x] C1：application 仅依赖 workspace/runner/command-query port；concrete runner 与 local filesystem adapter 仅位于 infrastructure 并由 bootstrap 构造。
- [x] C2：application 不依赖 `database/sql`；repository adapter 将 no-row 统一表达为 `repository.ErrNotFound`。
- [x] C3：application CI/CD command 仅依赖 typed business dispatch port；queue adapter 独占 `queue/task`、task type 与 JSON transport 细节。
- [x] 每项均删除旧职责路径，不保留 wrapper、alias、compatibility forwarding 或新旧 import 并存。
- [x] 未创建空目录或未来预留结构。
- [x] 每项均在 verification 记录最终职责、实际 diff、风险、未完成项和测试结果。
- [x] `go fmt ./cmd/... ./internal/...` 通过。
- [x] `./bin/golangci-lint fmt ./cmd/... ./internal/...` 通过。
- [x] `./bin/golangci-lint run ./cmd/... ./internal/...` 通过，`0 issues.`。
- [x] `go vet ./cmd/... ./internal/...` 通过。
- [x] `go test ./cmd/... ./internal/...` 通过。
- [x] 未修改 API route path、request/response contract、数据库 migration 或前端行为。
