# 接口层 Proto 领域路径与生成 DTO 收敛计划
最后修改时间: 2026-07-24 11:31:49

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260724-ci-cd-dto-file-split.md`（`Accepted`）
- Spec: `docs/spec/20260724-ci-cd-dto-file-split.md`（`Accepted`）
- 领域基线: `docs/analyze/20260724-domain-split-consensus-共识.md`
- 流程: 标准模式 / standard

本计划整页替换旧版（拆 application monofile / 旧领域表）方案。

## Overview

1. 将 `proto/orbit/v1/*.proto` 迁入 `proto/orbit/v1/<domain>/<entity>.proto`，package = `orbit.v1.<domain>`。  
2. 按共识完成消息搬家、规范更名、死文件删除。  
3. `task proto` 再生 Go/TS。  
4. 收敛接口层与前端对生成类型的引用；删除手写 API `*Resp`（含 Traefik）。  
5. **不做** SQL migration 表 rename、sqlc/repository 领域包拆、application 模块重划。

## Symbol rename table（契约类型）

| 旧 | 新 | 所在域 / 文件 |
|---|---|---|
| `BuildStageResp` | `PipelineStageResp` | `pipeline/pipeline_stage.proto` |
| `BuildStageCreateReq` | `PipelineStageCreateReq` | 同上 |
| `BuildStageUpdateReq` | `PipelineStageUpdateReq` | 同上 |
| `BuildStageDuplicateReq` | `PipelineStageDuplicateReq` | 同上 |
| `BuildStagePaginatedResp` | `PipelineStagePaginatedResp` | 同上 |
| `StageRunResp` | `PipelineStageRunResp` | `pipeline_run/pipeline_stage_run.proto` |
| `ServiceResp` / `ServiceListResp` / `ServicePaginatedResp` | 同名 | 迁至 `service/service.proto` |
| `PipelineRunExecuteTaskReq` | 同名 | 迁至 `pipeline_run`（从 task） |
| `ApplicationDeployTaskReq` / `Restart` / `Stop` | 同名 | 迁至 `deployment`（从 task） |
| `ArtifactConfig*` | 同名保留 | `pipeline/pipeline_stage.proto` |
| `config_file` / `service_config` 全部 message | — | **删除** |
| `cddto.Traefik*Resp` | — | **删除**；用 `route` 生成类型 |

搜索替换完成后，仓库中不得残留 `BuildStage`、`StageRunResp`（类型名）、扁平 `gen/proto/orbit/v1/*.pb.go` 引用。

## Proto file migration table

| 现有 | 目标动作 |
|---|---|
| `common.proto` | → `common/common.proto`，`package orbit.v1.common` |
| `auth.proto` | → `auth/auth.proto`，`orbit.v1.auth` |
| `user.proto` | → `user/user.proto`，`orbit.v1.user` |
| `role.proto` | → `role/role.proto`，`orbit.v1.role` |
| `project.proto` | → `project/project.proto`，`orbit.v1.project` |
| `settings.proto` | → `settings/settings.proto`，`orbit.v1.settings` |
| `task.proto` | → `task/task.proto`；**移出** 四个业务 *TaskReq |
| `credential.proto` | → `credential/credential.proto` |
| `repository.proto` | → `repository/repository.proto`；import `orbit/v1/common/common.proto` |
| `webhook.proto` | → `repository/webhook.proto`，同 package `orbit.v1.repository` |
| `build_stage.proto` | → `pipeline/pipeline_stage.proto`；message 更名 `PipelineStage*` |
| `template.proto` | → `pipeline/template.proto`；import pipeline_stage + common |
| `snapshot.proto` | → `pipeline/snapshot.proto`；import pipeline_stage + common |
| `pipeline_run.proto` | → `pipeline_run/pipeline_run.proto`；import artifact、pipeline_stage_run、common；列表字段类型改 `PipelineStageRunResp` |
| （StageRun 消息） | **新建** `pipeline_run/pipeline_stage_run.proto` |
| `artifact.proto` | → `pipeline_run/artifact.proto` |
| `application.proto` | → `application/application.proto` |
| `version.proto` | → `application/version.proto`；**移出** Service* |
| `application_bundle.proto` | → `application/application_bundle.proto`；import version |
| （Service*） | **新建** `service/service.proto`，`orbit.v1.service` |
| `environment.proto` | → `environment/environment.proto` |
| `deployment.proto` | → `deployment/deployment.proto`；**迁入** Application*TaskReq |
| `gateway.proto` | → `gateway/gateway.proto` |
| `route.proto` | → `route/route.proto` |
| `traefik.proto` | → `route/traefik.proto`，同 package `orbit.v1.route` |
| `config_file.proto` | **删除** |
| `service_config.proto` | **删除** |

目标树见 Spec；实现后 `proto/orbit/v1/` 下不得再有扁平 `*.proto`。

## Implementation steps

### Step 0: 基线

1. 确认 `task deps` 后 `bin/buf`、`bin/protoc-gen-go` 可用。
2. 记录当前 `rg` 命中：`BuildStage`、`StageRunResp`、`config_file`、`service_config`、`TraefikConfigResp`（application dto）、`internal/gen/proto/orbit/v1` 扁平 import。  
3. 不改 `buf.yaml` module path（仍为 `proto`）。

### Step 1: 重排 proto 源

1. 创建领域目录；`git mv` 或等价移动文件到目标路径。  
2. 每个文件设置 `package orbit.v1.<domain>;`。  
3. 更新全部 `import` 为 `orbit/v1/<domain>/<entity>.proto`。  
4. 执行消息搬家与更名（见上两表）。  
5. 删除 `config_file.proto`、`service_config.proto`。  
6. `buf lint`（若有）/ `task proto` 前先保证目录无扁平残留。

### Step 2: 生成

1. 运行 `task proto`（`buf dep update` + Go generate + web template generate）。  
2. 确认：  
   - `internal/gen/proto/orbit/v1/<domain>/*.pb.go`  
   - `web/src/gen/proto/orbit/v1/<domain>/*.ts`  
   - 无旧扁平 gen 文件（`clean: true`）。  
3. 确认存在 `pipeline_stage.pb.go` / `pipeline_stage.ts`、`pipeline_stage_run.*`、`service.*`；不存在 `build_stage`、`config_file`、`service_config` 生成物。

### Step 3: 后端接口层 import 与类型

1. 将所有  
   `pomeloorbit "…/internal/gen/proto/orbit/v1"`  
   改为按域 import + Spec 别名（`authv1`、`pipelinev1`、`pipelinerunv1`、`applicationv1`、`servicev1`、`routev1`…）。  
2. 替换类型限定：`pomeloorbit.X` → `<domainv1>.X`，并应用 Symbol rename table。  
3. 范围：  
   - `internal/api/http/handler/**`  
   - `internal/api/http/binding/**`  
   - `internal/api/http/routes/**`（若引用类型）  
   - `internal/api/http/response/**`、`codec/**` 测试  
   - `internal/queue/**`、task 相关 handler  
4. `binding.RepositoryWebhookUpdateReq` 内嵌类型改为 `repositoryv1.RepositoryWebhookUpdateReq`（或同包生成名）。  
5. **禁止** `internal/application` import `internal/gen/proto`（除当前已违规处清零）。

### Step 4: Traefik 手写 DTO 删除

1. 从 `internal/application/cd/dto/cd.go` 删除 `TraefikRouterResp`、`TraefikConfigResp`、`TraefikRouteListResp`。  
2. `Service.TraefikRouteConfig` / `ListTraefikRoutes` 改为返回 port/内部 view（优先 `cdport.TraefikRouter` 或私有 struct）。  
3. `handler/cd/route_mapper.go`、`route.go`、测试：组装 `routev1.Traefik*Resp`。  
4. 更新 `route_test.go` 等断言。

### Step 5: 后端符号与路径残留清理

1. `rg BuildStage|StageRunResp|build_stage\.pb|config_file|service_config|gen/proto/orbit/v1"`（排除 migration/sql 物理表名处）。  
2. 接口与测试中的 `BuildStage` 类型全部改为 `PipelineStage`。  
3. application 层若仅有字符串/注释含 build_stage 表名可保留；**生成 API 类型名**不得残留。  
4. model/repository **不**强制 rename 表结构（non-goal）。

### Step 6: 前端

1. 全局替换 `@/gen/proto/orbit/v1/<entity>` → `@/gen/proto/orbit/v1/<domain>/<entity>`。  
2. 类型名：`BuildStage*` → `PipelineStage*`；`StageRun*` → `PipelineStageRun*`（若前端有引用）。  
3. 范围：`web/src/api/**`、`stores/**`、`views/**`、`components/**`。  
4. `web/src/api/ci/build_stage.ts`：更新 import 与类型名；**文件名**可暂留 `build_stage.ts`（非 gen）或顺手改为 `pipeline_stage.ts` 并改 `index` 导出——**推荐**改为 `pipeline_stage.ts` 与类型一致，仍放在 `api/ci/` 目录下。  
5. 不强制重命名 `api/ci|cd` 目录。

### Step 7: 检查命令

后端：

```text
go fmt ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

前端：

```text
yarn --cwd web typecheck
yarn --cwd web lint:fix
```

生成：

```text
task proto
```

（实现中若已生成，Verification 再跑一遍确认可复现。）

### Step 8: 实现收尾

1. 汇报：proto 树、删除文件、更名表执行情况、Traefik 收敛 diff。  
2. 对照本 plan「预期不改」列表。  
3. 停在 Implementation，待用户要求再 Verification。

## Files to change（预期）

### 必改

- `proto/orbit/v1/**`（整树）  
- `internal/gen/proto/**`（生成）  
- `web/src/gen/proto/**`（生成）  
- `internal/api/http/**`  
- `internal/application/cd/dto/cd.go`  
- `internal/application/cd/usecase/route.go`（及直接依赖 Traefik DTO 的测试）  
- `internal/queue/**`、task handler（TaskReq 包路径）  
- `web/src/api/**` 及所有 `@/gen/proto` / `BuildStage` 引用  

### 可能触及

- `internal/api/http/handler/ci/build_stage*.go` 等文件名可暂留，类型改名  
- 少量 application 测试构造 API 响应处  

### 预期不改

- `sql/migration/**`  
- `sql/query/**` 领域拆分、sqlc 包路径  
- `internal/repository` / `internal/model` monofile 拆分  
- `internal/application` 按域重划包  
- HTTP 路由 path 字符串  
- `buf.gen.yaml` / `web/buf.gen.yaml`（除非 generate 失败）  

## Go import 别名速查

| 域 | 别名 | 路径后缀 |
|---|---|---|
| common | `commonv1` | `orbit/v1/common` |
| auth | `authv1` | `orbit/v1/auth` |
| user | `userv1` | `orbit/v1/user` |
| role | `rolev1` | `orbit/v1/role` |
| project | `projectv1` | `orbit/v1/project` |
| settings | `settingsv1` | `orbit/v1/settings` |
| task | `taskv1` | `orbit/v1/task` |
| credential | `credentialv1` | `orbit/v1/credential` |
| repository | `repositoryv1` | `orbit/v1/repository` |
| pipeline | `pipelinev1` | `orbit/v1/pipeline` |
| pipeline_run | `pipelinerunv1` | `orbit/v1/pipeline_run` |
| application | `applicationv1` | `orbit/v1/application` |
| environment | `environmentv1` | `orbit/v1/environment` |
| service | `servicev1` | `orbit/v1/service` |
| deployment | `deploymentv1` | `orbit/v1/deployment` |
| gateway | `gatewayv1` | `orbit/v1/gateway` |
| route | `routev1` | `orbit/v1/route` |

## Verification plan

1. **目录**：`proto/orbit/v1` 仅领域子目录；无扁平 proto；无 config_file/service_config。  
2. **命名**：`rg BuildStage|StageRunResp` 在 proto/gen/api 层无命中（允许 sql/model 物理名）。  
3. **生成路径**：Go/TS 均含 `<domain>` 中间段。  
4. **Traefik**：`application/cd/dto` 无 Traefik*Resp；handler 使用 `routev1`。  
5. **application 不 import gen/proto**。  
6. **命令**：上述 fmt/vet/test/typecheck/lint:fix 通过。  
7. **抽样 wire**：login、pipeline stage list、pipeline run detail、traefik list 的 JSON 字段名仍为 snake_case。  
8. 写 `docs/verification/20260724-ci-cd-dto-file-split.md`。

## Blockers

无。工具链依赖本地 `task deps`。

## Assumptions

1. product = `orbit`；领域表以共识为准不再改。  
2. message 更名导致的类型 churn 可接受；JSON field 名默认不动。  
3. 物理表仍名 `build_stage` / `stage_run` 直至后续 migration 任务。  
4. 前端 `api/ci|cd` 目录保留可接受。  
5. `pipeline_stage_run.proto` 至少包含原 `StageRunResp`；若仅此一个 message 仍保持独立文件（共识要求）。

## Risks

1. 全仓机械替换遗漏 → 编译/typecheck 失败。  
2. Proto 名与 DB 表名暂时不一致 → 文档/注释混淆；Verification 注明。  
3. Traefik 返回类型调整波及 integration test。  
4. 生成文件大 diff → 整组提交 gen。  
5. 删除死 proto 若存在隐藏引用 → 编译期暴露。

## Rollback

1. 还原 `proto/orbit/v1` 与全部引用改动。  
2. 重新 `task proto`。  
3. 还原 Traefik application DTO。  
4. 不可半套生成物与半套源并存。

## Out of scope follow-ups

| 项 | 说明 |
|---|---|
| `build_stage` / `stage_run` 表 rename migration | 共识 §8；另立任务 |
| sqlc / repository 按域拆包 | 共识 §7 |
| application / model monofile 拆分 | 延后 |
| HTTP path 去 ci/cd | 延后 |
| 前端 api 目录按域重命名 | 可选后续 |

## User review notes

- 用户要求「开始 plan」：Spec 标为 Accepted；本 Plan 为 Draft 待确认。  
- 实现前请确认本 Plan；确认后回复「开始实现」。
