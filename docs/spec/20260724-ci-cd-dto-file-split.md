# 接口层 Proto 领域路径与生成 DTO 收敛规格
最后修改时间: 2026-07-24 11:31:49

## Review status

Accepted

## Requirement basis

- Requirement: `docs/requirement/20260724-ci-cd-dto-file-split.md`（`Accepted`）
- 领域基线: `docs/analyze/20260724-domain-split-consensus-共识.md`（已确认）
- 流程: 标准模式 / standard；用户要求进入 Plan，Spec 标为 Accepted

## Overview

将扁平 `proto/orbit/v1/*.proto` 迁入 `proto/orbit/v1/<domain>/<entity>.proto`，package 改为 `orbit.v1.<domain>`，按共识完成消息归属纠正、规范命名与死契约删除，再生成 Go/TS DTO 并收敛接口层与前端引用。HTTP 仍为 JSON + protojson；不引入 gRPC。SQL/sqlc/application 包重划不在本规格实施范围（表级 rename migration 另立任务，但 **Proto 侧标识符本迭代即对齐共识逻辑名**）。

## Design decisions

1. **领域地图只读共识**  
   目录、package、实体文件落点严格采用共识 §5.1；本规格不发明第二套领域名。

2. **同域同 package**  
   同一 `<domain>` 下多个 `.proto` 共用 `package orbit.v1.<domain>;`，生成同一 Go 包 / 可多 TS 文件同 package 字符串。跨域通过 `import "orbit/v1/<other>/<file>.proto"` 引用类型。

3. **source_relative 生成**  
   保持根 `buf.gen.yaml`：`out: internal/gen/proto`，`paths=source_relative`，managed `go_package_prefix` 不变。结果路径为 `internal/gen/proto/orbit/v1/<domain>/*.pb.go`。  
   Web：`web/buf.gen.yaml` out 仍为 `src/gen/proto`，结果为 `web/src/gen/proto/orbit/v1/<domain>/*.ts`。

4. **Wire 与命名策略（对齐共识）**  
   - HTTP path（含历史 `/api/ci/*`、`/api/cd/*`）不变。  
   - JSON **字段名**（message 内 field）保持现有 snake_case（`UseProtoNames: true` + ts-proto `snakeToCamel=false`），除非某 field 本身含 `build_stage` 等旧逻辑词且共识要求一并改名（Plan 列清单）。  
   - Proto **message 类型名** 与共识逻辑名对齐（见决策 5），接受 Go/TS 类型标识符 churn。  
   - **不**为本迭代保留 `BuildStage*` / `StageRun*` 双轨别名。

5. **规范命名与文件拆分（对齐共识 §5、§8）**  

   | 共识逻辑名 | proto 文件 | message 标识符（本迭代必须） |
   |---|---|---|
   | `pipeline_stage` | `pipeline/pipeline_stage.proto`（自 `build_stage.proto`） | `BuildStage*` → **`PipelineStage*`**；`ArtifactConfig*` 保留（阶段配置，非产物） |
   | `pipeline_stage_run` | **`pipeline_run/pipeline_stage_run.proto`（必须独立文件）** | `StageRunResp` → **`PipelineStageRunResp`**；相关仅描述 stage 执行实例的消息同文件 |
   | `service` | `service/service.proto` | 保留 `ServiceResp*`（已是规范名，仅迁域） |

   更名对照（契约类型，Plan 展开全量替换）：

   | 旧 message | 新 message |
   |---|---|
   | `BuildStageResp` | `PipelineStageResp` |
   | `BuildStageCreateReq` | `PipelineStageCreateReq` |
   | `BuildStageUpdateReq` | `PipelineStageUpdateReq` |
   | `BuildStageDuplicateReq` | `PipelineStageDuplicateReq` |
   | `BuildStagePaginatedResp` | `PipelineStagePaginatedResp` |
   | `StageRunResp` | `PipelineStageRunResp` |

   `pipeline_run.proto` 中对 stage 执行列表的字段类型改为引用 `PipelineStageRunResp`；`template` / `snapshot` 中引用阶段定义处改为 `PipelineStage*` 或既有 orchestration 结构（以生成后编译为准，禁止残留 `BuildStage` 类型名）。

6. **死契约删除**  
   删除 `config_file.proto`、`service_config.proto` 及全部生成物与引用。部署 compose 物化不属于 API body 契约。

7. **Task 命令消息搬家**  
   - `PipelineRunExecuteTaskReq` → `pipeline_run` 包  
   - `ApplicationDeployTaskReq` / `Restart` / `Stop` → `deployment` 包  
   - `CreateTaskReq`、`TaskResp` 留 `task`  
   Handler/worker 引用改 import，不改队列 wire 若 JSON 字段未变。

8. **Service 消息搬家**  
   `ServiceResp` / `ServiceListResp` / `ServicePaginatedResp` 从 version 文件迁入 `service/service.proto`，package `orbit.v1.service`。application/version 文件不再定义 runtime service 列表类型。

9. **Traefik 只认 route 生成类型**  
   删除 `cddto.TraefikRouterResp` 等。usecase 返回 `port` 或内部 view；`handler` mapper 映射为 `route` 包生成 message。**application 不得 import `internal/gen/proto`**。

10. **Binding 适配器**  
    `binding.RepositoryWebhookUpdateReq`（presence）保留，内嵌 proto message；不算契约双轨。

11. **错误 envelope**  
    `response.ErrorResp` 不进 proto。

12. **前端目录**  
    `web/src/api/ci|cd` 可暂留；仅更新 `@/gen/proto/...` import 路径。前端符号 `BuildStage*` / 文件名 `build_stage.ts` 等改为 `PipelineStage*` / `pipeline_stage` 生成路径。

13. **Go import 别名约定**  
    禁止再使用单一 `pomeloorbit` 指向扁平 `orbit/v1`。按领域：
    - `authv1`、`userv1`、`rolev1`、`projectv1`、`settingsv1`、`taskv1`
    - `credentialv1`、`repositoryv1`、`pipelinev1`、`pipelinerunv1`
    - `applicationv1`、`environmentv1`、`servicev1`、`deploymentv1`、`gatewayv1`、`routev1`
    - `commonv1`（若需）  
    同一文件多域时全部使用带域别名；测试文件同样遵守。

## Affected components

| 区域 | 变更 |
|---|---|
| `proto/orbit/v1/**` | 迁目录、改 package/import、消息搬家、删死文件 |
| `buf.yaml` / `buf.gen.yaml` / `web/buf.gen.yaml` | 通常不改；generate 失败时再查 |
| `Taskfile.yml` `proto` 任务 | 保持；实现后必跑 |
| `internal/gen/proto/**` | 全量再生 |
| `web/src/gen/proto/**` | 全量再生 |
| `internal/api/http/**` | import、类型包前缀、`BuildStage*`/`StageRun*` → 新 message 名 |
| `internal/application/**` 中引用生成 API 类型或旧符号处 | 最小替换为 `PipelineStage*` / `PipelineStageRun*`；不重划包 |
| `internal/application/cd/dto` + route usecase | 删 Traefik*Resp；返回内部类型 |
| `internal/queue/**`、task handler | Task 子命令 message 新包路径 |
| `web/src/api/**`、stores、views、components | gen import 路径与 `PipelineStage*` 类型名 |
| **不改** | `sql/migration/**` 表 rename、sqlc 领域包拆分、application 按域重划包、HTTP path |

## Interfaces

### Proto 目标树

```text
proto/orbit/v1/
  common/common.proto
  auth/auth.proto
  user/user.proto
  role/role.proto
  project/project.proto
  settings/settings.proto
  task/task.proto
  credential/credential.proto
  repository/
    repository.proto
    webhook.proto
  pipeline/
    template.proto
    pipeline_stage.proto       # 自 build_stage.proto；message = PipelineStage*
    snapshot.proto
  pipeline_run/
    pipeline_run.proto
    pipeline_stage_run.proto   # 必须；message = PipelineStageRun*
    artifact.proto
  application/
    application.proto
    version.proto           # 无 Service*
    application_bundle.proto
  environment/environment.proto
  service/service.proto
  deployment/deployment.proto  # 含 Application*TaskReq
  gateway/gateway.proto
  route/
    route.proto
    traefik.proto
```

### Package 与 import 示例

```protobuf
// proto/orbit/v1/pipeline/snapshot.proto
syntax = "proto3";
package orbit.v1.pipeline;
import "orbit/v1/pipeline/pipeline_stage.proto";
import "orbit/v1/common/common.proto";
```

```protobuf
// proto/orbit/v1/application/application_bundle.proto
syntax = "proto3";
package orbit.v1.application;
import "orbit/v1/application/version.proto";
```

```protobuf
// proto/orbit/v1/pipeline_run/pipeline_run.proto
syntax = "proto3";
package orbit.v1.pipeline_run;
import "orbit/v1/pipeline_run/artifact.proto";
import "orbit/v1/pipeline_run/pipeline_stage_run.proto";
import "orbit/v1/common/common.proto";
```

```protobuf
// proto/orbit/v1/pipeline_run/pipeline_stage_run.proto
syntax = "proto3";
package orbit.v1.pipeline_run;
// message PipelineStageRunResp { ... }  // 自原 StageRunResp
```

```protobuf
// proto/orbit/v1/pipeline/pipeline_stage.proto
syntax = "proto3";
package orbit.v1.pipeline;
// message PipelineStageResp / CreateReq / UpdateReq / ...
// message ArtifactConfigResp / ArtifactConfigReq 保留
```

### 跨域类型依赖（契约层）

```text
common ← repository, pipeline, pipeline_run（变量声明等）
pipeline/pipeline_stage ← pipeline/template, pipeline/snapshot
pipeline_run/pipeline_stage_run ← pipeline_run/pipeline_run
artifact ← pipeline_run
version ← application_bundle（同 application 包内 import）
```

gateway / route / service / deployment / application 之间 **proto message 尽量不交叉 import**；关联用 id 字符串字段表达（与现状一致）。

### Go handler 形态

```go
var req applicationv1.ApplicationCreateReq
if err := binding.DecodeJSON(c, &req); err != nil { ... }
// mapper → application dto Input（非 proto）
transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.ApplicationResp{...})
```

### Traefik 收敛

```text
port/internal view ──mapper──→ routev1.TraefikRouteListResp / TraefikConfigResp
                              （protojson 输出）
```

禁止：`application/dto` 定义与 `routev1` 同构的 `*Resp`。

### 前端形态

```ts
import type { ApplicationResp } from '@/gen/proto/orbit/v1/application/application';
import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';
import type { TraefikRouteListResp } from '@/gen/proto/orbit/v1/route/traefik';
```

## Migration of existing files

| 现有 | 目标 |
|---|---|
| `common.proto` | `common/common.proto` |
| `auth.proto` | `auth/auth.proto` |
| `user.proto` | `user/user.proto` |
| `role.proto` | `role/role.proto` |
| `project.proto` | `project/project.proto` |
| `settings.proto` | `settings/settings.proto` |
| `task.proto` | `task/task.proto`（去掉业务 *TaskReq） |
| `credential.proto` | `credential/credential.proto` |
| `repository.proto` | `repository/repository.proto` |
| `webhook.proto` | `repository/webhook.proto` |
| `build_stage.proto` | `pipeline/pipeline_stage.proto`；message `BuildStage*` → `PipelineStage*` |
| `template.proto` | `pipeline/template.proto`；引用类型改为 `PipelineStage*` |
| `snapshot.proto` | `pipeline/snapshot.proto`；引用类型改为 `PipelineStage*` |
| `pipeline_run.proto` | `pipeline_run/pipeline_run.proto`；`StageRunResp` 迁出并更名 |
| （原 `StageRunResp` 等） | `pipeline_run/pipeline_stage_run.proto`；`PipelineStageRunResp` |
| `artifact.proto` | `pipeline_run/artifact.proto` |
| `application.proto` | `application/application.proto` |
| `version.proto` | `application/version.proto`（去掉 Service*） |
| `application_bundle.proto` | `application/application_bundle.proto` |
| `environment.proto` | `environment/environment.proto` |
| （Service* 自 version） | `service/service.proto` |
| `deployment.proto` | `deployment/deployment.proto`（+ Application*TaskReq） |
| `gateway.proto` | `gateway/gateway.proto` |
| `route.proto` | `route/route.proto` |
| `traefik.proto` | `route/traefik.proto` |
| `config_file.proto` | **删除** |
| `service_config.proto` | **删除** |

## Technical questions

无阻塞项。下列已由用户确认对齐共识：

1. **`BuildStage*` → `PipelineStage*`**：本迭代在 Proto / 生成类型 / 接口与前端引用中完成更名。  
2. **`StageRunResp` → `PipelineStageRunResp`**，且必须落在独立文件 `pipeline_run/pipeline_stage_run.proto`。  
3. **表级 / sqlc 的 `build_stage`、`stage_run` rename migration**：仍不在本需求；与 Proto 标识符可暂时不一致，后续 migration 任务对齐物理表名。

## Risks

- `PipelineStage*` / `PipelineStageRun*` 类型全量替换，handler / 前端 / 测试 diff 增大。  
- Proto 逻辑名已更新而 DB 表名仍为 `build_stage` / `stage_run`，阅读时需区分契约名与物理表名（直至后续 migration）。  
- 多包 import 使 handler 文件 import 变长；用统一别名约定缓解。  
- 删除 config_file/service_config 的隐蔽引用。  
- Traefik usecase 返回类型调整可能带动少量测试。  
- 旧 Plan 已过时，须在 Plan 阶段整页重写。

## Alternatives

| 方案 | 为何不采用 |
|---|---|
| 保持 package `orbit.v1` 只改目录 | 生成 Go 仍易挤在一包或路径与 package 不一致，领域感弱 |
| 本迭代连同 sqlc/migration 全拆 | 超出 requirement 范围；共识检查清单留给后续变更 |
| 仅改路径、保留 `BuildStage*` / `StageRun*` message 名 | **违背共识**；用户已要求本迭代对齐 |
| 前端 api 目录按领域重命名 | 非本需求；可后续 |

## Implementation outline（供 Plan 展开，非实现）

1. 按迁移表移动/改写 proto（package、import、消息搬家与 **PipelineStage* / PipelineStageRun* 更名**、独立 `pipeline_stage_run.proto`、删除死文件）。  
2. `task proto` 再生 Go + TS。  
3. 全局替换后端 gen import、类型限定与旧符号名。  
4. 删 Traefik 手写 Resp，改 usecase/mapper。  
5. 前端 gen import 与 `PipelineStage*` 类型名替换。  
6. fmt / vet / test / typecheck / lint:fix。  
7. 停在 Implementation 待人工验收后再 Verification。

## User review notes

- 规格对齐共识 §5 Proto 指导与 §8 契约相关纠正。  
- 本迭代 **明确排除** migration/sqlc 包拆分；检查清单中对应项记为 N/A 或 follow-up。  
- 2026-07-24 用户确认：  
  1. message **必须** `BuildStage*` → `PipelineStage*`；  
  2. **必须** 独立 `pipeline_stage_run.proto`，且 `StageRun*` → `PipelineStageRun*`。  
- 待用户 Accept 后重写 Plan 并进入实现。
