# 接口层 Proto 领域路径与生成 DTO 收敛
最后修改时间: 2026-07-24 11:23:35

## Review status

Accepted

## Background

API body 契约已由 `proto` + `buf` 生成前后端类型，但布局与项目领域基线不一致：

1. Proto 源为扁平 `proto/orbit/v1/<entity>.proto`，package 统一为 `orbit.v1`，生成物路径不体现领域。
2. `ci` / `cd` 仅适合历史路由与前端页面分组，不是业务领域名。
3. 前后端 API DTO 只认 proto 生成；与 proto 对不上的手写 API `*Req` / `*Resp` 应删除。
4. 消息归属存在错放与死契约：如 `ServiceResp*` 在 version、Task 业务命令在 task、`config_file` / `service_config` 无生产调用方、application 层重复定义 `Traefik*Resp`。

领域边界以 `docs/analyze/20260724-domain-split-consensus-共识.md` 为准。本需求对应其中的 **Proto 指导与已确认纠正中的契约部分**；SQL migration rename、sqlc / repository 包拆分、application 模块重划不在本需求实施范围，另立任务。

## Goal

- 将 proto 源路径收敛为 `proto/orbit/v1/<domain>/<entity>.proto`，package 为 `orbit.v1.<domain>`。
- 按共识领域地图落放实体文件，并完成共识中的 **消息归属纠正** 与 **死 proto 删除**。
- 重新生成 Go / TypeScript DTO，更新接口层（HTTP handler / mapper / binding / 测试）与前端对 gen 的引用。
- 删除接口语义的手写 API DTO（含 `cddto.Traefik*Resp`）；HTTP 编解码只使用生成类型。
- 保持对外 HTTP 路径、JSON 字段名（snake_case）、状态码与业务行为不变（允许 import / package / 生成路径变化）。

## Non-goal

- 不执行 SQL migration 表/列 rename（如物理表 `build_stage` → `pipeline_stage`）；**Proto message / 生成类型名** 本阶段即对齐共识：`PipelineStage*`、`PipelineStageRun*`，并采用独立 `pipeline_stage_run.proto`。JSON **字段名**默认保持 snake_case 现状；物理表名可暂时落后于契约名，由后续 migration 任务对齐。
- 不拆分 sqlc query 包、repository 包、application / model 模块（除删除手写 API 形 `*Resp` 所必需的最小返回类型调整）。
- 不把 path / query / header / 错误 envelope / RPC service 写入 proto。
- 不引入旧路径别名、双轨生成目录或兼容层。
- 不主动启停开发服务器；未经授权不做 git 写操作。

## User scenarios

- 契约维护者在 `proto/orbit/v1/<domain>/` 下按领域修改 API body。
- 后端 handler 从 `internal/gen/proto/orbit/v1/<domain>/` 引用生成类型，不手写 API DTO。
- 前端从 `@/gen/proto/orbit/v1/<domain>/<entity>` 引用类型。
- 后续 migration / sqlc / application 拆分可直接对齐同一领域地图，无需再争论 ci/cd 桶。

## Domain map（共识锁定）

`ci` / `cd` 不得作为领域目录名。`common` 为共享契约，不是业务领域。

| 领域 | Proto 文件 |
|---|---|
| `common` | `common/common.proto` |
| `auth` | `auth/auth.proto` |
| `user` | `user/user.proto` |
| `role` | `role/role.proto` |
| `project` | `project/project.proto` |
| `settings` | `settings/settings.proto` |
| `task` | `task/task.proto` |
| `credential` | `credential/credential.proto` |
| `repository` | `repository/repository.proto`、`repository/webhook.proto` |
| `pipeline` | `pipeline/template.proto`、`pipeline/pipeline_stage.proto`、`pipeline/snapshot.proto` |
| `pipeline_run` | `pipeline_run/pipeline_run.proto`、`pipeline_run/pipeline_stage_run.proto`、`pipeline_run/artifact.proto` |
| `application` | `application/application.proto`、`application/version.proto`、`application/application_bundle.proto` |
| `environment` | `environment/environment.proto` |
| `service` | `service/service.proto` |
| `deployment` | `deployment/deployment.proto` |
| `gateway` | `gateway/gateway.proto` |
| `route` | `route/route.proto`、`route/traefik.proto` |

## Message ownership corrections（共识锁定）

| 项 | 目标 |
|---|---|
| `ServiceResp` / `ServiceListResp` / `ServicePaginatedResp` | 迁出 version → `service/service.proto` |
| `PipelineRunExecuteTaskReq` | 迁出 task → `pipeline_run` |
| `ApplicationDeployTaskReq` / `ApplicationRestartTaskReq` / `ApplicationStopTaskReq` | 迁出 task → `deployment` |
| `TaskResp` 与通用创建/查询 | 留在 `task` |
| Traefik 只读消息 | `route/traefik.proto` |
| `ArtifactConfig*` | 留在 `pipeline_stage`（构建阶段配置，非产物 artifact） |
| `config_file.proto` | **删除**源与生成物 |
| `service_config.proto` | **删除**源与生成物 |
| 原 `build_stage.proto` | 迁入 `pipeline/pipeline_stage.proto`；message `BuildStage*` → **`PipelineStage*`** |
| 原 `StageRunResp` 等 | 迁入 **`pipeline_run/pipeline_stage_run.proto`**；message → **`PipelineStageRun*`** |
| application 层 `Traefik*Resp` | **删除**；HTTP 使用 route 域生成类型 |

## Acceptance

1. 不存在扁平 `proto/orbit/v1/*.proto`；源文件均在 `proto/orbit/v1/<domain>/`。
2. 每个业务 proto 的 package 为 `orbit.v1.<domain>`；`common` 为 `orbit.v1.common`。
3. 领域目录与上表一致；已删除 `config_file` / `service_config`；`pipeline_stage` / `pipeline_stage_run` / `service` / task 子命令消息归属符合上表；不存在 `BuildStage*` / `StageRunResp` 生成类型名。
4. `task proto` 成功；Go 生成于 `internal/gen/proto/orbit/v1/<domain>/`，TS 生成于 `web/src/gen/proto/orbit/v1/<domain>/`。
5. 接口层与前端 API body 类型只引用生成物；无 `cddto.Traefik*Resp` 等与 proto 双轨的手写 API DTO。
6. 对外 HTTP 行为与 JSON 字段名（除 Spec 明确记载的不可避免变更外）保持不变。
7. Go：`go fmt` / `go vet` / `go test`（项目约定范围）；前端：`yarn --cwd web typecheck` 与 `lint:fix`。

## Open questions

无阻塞项。下列项已由共识或本需求锁定：

1. package = `orbit.v1.<domain>` — 是。
2. version 留在 application 域文件 `application/version.proto` — 是；service 为独立域。
3. Traefik：usecase 返回内部/port 类型，handler 组装 proto；application 不 import gen — 是。
4. `binding.RepositoryWebhookUpdateReq` presence 包装保留 — 是。
5. `ErrorResp` 不进 proto — 是。
6. 本需求是否含 SQL/sqlc 拆分 — **否**（共识全文指导后续任务）。

## Decisions

1. 流程模式：标准模式 / standard；本迭代额外产出 Spec（用户要求），再进入 Plan → Implementation → Verification。
2. 领域地图与消息归属：**锁定**共识文档 §3、§5、§8 中与 Proto 相关的结论。
3. 实施范围：Proto 布局 + 生成 + 接口/前端引用收敛 + 手写 API DTO 删除 + 死 proto 删除 + **`PipelineStage*` / `PipelineStageRun*` 契约更名**；**不含** migration / sqlc / application 包重划。
4. API DTO 唯一来源：proto 生成；对不上的手写直接删。
5. 不使用 `ci` / `cd` 作为 proto 领域名。
6. 废弃「仅拆 application `ci.go`/`cd.go` monofile」旧方案。
7. 用户确认（2026-07-24）：Spec 开放点 1/2 按共识修改——message 更名 + `pipeline_stage_run` 独立文件。

## Risk

- 全仓库 import / 生成物 / 类型名机械替换，diff 大。
- 契约已用 `PipelineStage*` / `PipelineStageRun*`，物理表仍可能是 `build_stage` / `stage_run`，需在后续 migration 对齐。
- 删除 `config_file` / `service_config` 若存在隐蔽引用会导致编译失败（预期无生产调用方）。
- 删除 `Traefik*Resp` 触及 application usecase 返回类型，需最小改动。
- 前端 `api/ci|cd` 目录名与契约领域不一致，本阶段只改 gen import 与类型符号。

## User review notes

- 2026-07-24：用户澄清接口层范围、proto 路径约定、禁止手写 API DTO、Traefik 由 proto 管理。
- 2026-07-24：用户确认领域基线见 `docs/analyze/20260724-domain-split-consensus-共识.md`。
- 2026-07-24：用户要求按共识修正 requirement，标准模式，并产出 Spec。
- 2026-07-24：用户确认 Spec 点 1/2 对齐共识（`PipelineStage*` 更名 + 独立 `pipeline_stage_run.proto`）。
