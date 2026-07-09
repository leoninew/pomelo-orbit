# internal 分层后续逐条拆分
最后修改时间: 2026-07-09 10:12:07

Review status: Accepted

## Background

`internal/` 已完成第一轮分层迁移与包路径修复，后端格式化、lint、vet、test 已通过。迁移记录中仍有若干备注项属于“当前能工作，但架构边界仍值得继续审视”的后续问题。

本任务不一次性重构所有问题，而是按用户要求：过一条、拆一条；每次先记录问题和处理建议，再进入对应实现。

## Goal

- 建立后续拆分任务的轻量需求记录。
- 汇总第一轮迁移备注中值得关注的问题列表。
- 为每个问题记录初步处理建议、优先级和当前状态。
- 后续每次只选择一条进入实现，避免一次性大改。
- 每条拆分完成后补充处理结论和验证结果引用。

## Non-goal

- 不在本阶段修改产品代码。
- 不一次性完成全部拆分。
- 不引入兼容层、别名、新旧逻辑并存或默认值兜底。
- 不修改已执行的迁移文件。
- 不主动启动、停止或重启开发服务器。
- 不执行 `git add`、`git commit`、`git push` 等 Git 写操作。

## User scenarios

- 作为维护者，我希望逐条处理迁移后遗留的分层边界问题，避免一次性大范围重构引入风险。
- 作为维护者，我希望每条问题先有清楚的问题描述和处理建议，再决定是否进入实现。
- 作为维护者，我希望拆分记录能追踪每条问题的处理状态，便于暂停和恢复。

## Acceptance

- 文档位于 `docs/requirement/20260708-internal-layering-followup-split.md`。
- 文档列出后续值得关注的问题项。
- 每个问题项至少包含：关注文件、问题描述、处理建议、优先级、状态。
- 本阶段不修改产品代码。
- 后续进入实现时，每次只处理一条问题，完成后单独验证。

## Issue list

| 编号 | 关注文件 | 问题描述 | 处理建议 | 优先级 | 状态 |
|---|---|---|---|---|---|
| 1 | `internal/application/cd/route.go` | application 层直接包含文件系统写入/删除、YAML 生成、Traefik reload、Traefik HTTP API client、mkcert 命令调用等基础设施细节。 | 已完成大搬家：Traefik 文件发布、Traefik API client、Docker reload、mkcert 证书生成已搬到 `internal/infrastructure/traefik`，application 只保留权限、校验、状态流转和接口编排。 | 高 | 已实施，检查通过 |
| 2 | `internal/application/settings/service.go` | settings application service 直接读写 `.env` 文件，应用层与本地配置文件存储耦合。 | 已完成拆分：settings application 保留系统配置用例、key/value 转换、默认值与展示规则；`.env` 文件解析、写入、删除、路径解析移动到 `internal/infrastructure/config/envfile`，由 `EnvStore` 接口注入。 | 高 | 已实施，检查通过 |
| 3 | `internal/application/ci/runtime_variables.go` 与 `internal/workflow/activity/ci/runtime_variables.go` | runtime variable 逻辑跨触发与执行，迁移后 application/activity 两侧存在相近逻辑，可能出现重复和规则漂移。 | 已完成拆分：runtime variable、template variable 解析/清洗/内置变量/快照补全等纯规则集中到 `internal/common/civariable`；application 与 workflow activity 直接调用共享规则；删除 workflow activity wrapper-only 文件，不保留占位或兼容层。 | 高 | 已实施，检查通过 |
| 4 | `internal/application/ci/workspace.go` 与 `internal/workflow/activity/ci/workspace.go` | CI workspace 路径、artifact/log 路径和 Docker mount 计算跨 application/activity 使用，当前存在重复实现。 | 已完成拆分：共享 CI workspace 包位于 `internal/infrastructure/storage/local/ciworkspace`，集中承载路径规则、目录创建、physical root 解析缓存和 Docker mount 生成；application 与 workflow activity 直接依赖共享包，不保留 wrapper-only 兼容层。 | 中 | 已实施，检查通过 |
| 5 | `internal/application/cd/workspace.go` 与 `internal/workflow/activity/cd/workspace.go` | CD workspace 路径计算与物理数据根解析跨 application/activity 使用，当前存在重复实现，且依赖本地 physical data root 解析。 | 已完成拆分：共享 CD workspace 包位于 `internal/infrastructure/storage/local/cdworkspace`，集中承载 CD app/log 路径、physical root 解析缓存和 slash 格式 physical path 生成；application 与 workflow activity 直接依赖共享包，不保留 wrapper、别名、适配或兼容层。 | 中 | 已实施，检查通过 |
| 6 | `internal/repository/impl/sqlc/task/repository.go` | task repository 实现里包含跨层使用的 `Task` runtime model，导致 queue、worker、HTTP handler、application 接口依赖 sqlc repository impl。 | 已完成拆分：`Task` runtime model 迁到 `internal/queue/task`；sqlc task repository 只保留持久化和 SQLC row 转换；queue、worker、handler、application 调用点直接依赖 queue task model，不保留别名、wrapper 或适配层。 | 中 | 已实施，检查通过 |
| 7 | `internal/infrastructure/logger/logstore/logstore.go` | logstore 已放入 logger，但它实际承载 CI/CD execution log 的本地文件读写，不是应用 logger 初始化或 slog handler；当前包路径与 `logging` 并列容易误导职责边界，且 application/workflow 构造参数暴露 concrete infrastructure type。 | 已完成拆分：execution log 本地文件读写迁到 `internal/infrastructure/storage/local/executionlog`，旧 `logger/logstore` 删除；application/workflow 各自定义最小消费接口，bootstrap/server 注入 `executionlog.Store`；删除未使用的 `Exists`，不保留旧包、别名或适配层。 | 低 | 已实施，检查通过 |
| 8 | 根目录 `proto/` 与 `internal/gen/proto` | `internal/gen/proto` 是生成代码，根目录 `proto/` 是否作为 IDL 源目录保留尚未形成明确决策。 | 单独检查 proto 生成链路、Make/Task/Buf 配置；若根目录是源码约定，应保留并记录；不要强行迁入 internal。 | 低 | 待处理 |
| 9 | `internal/api/http/server.go` | HTTP server 是路由和依赖组装中心，当前可工作，但后续可能继续膨胀。 | 暂不优先拆；只有当新增路由继续增加 server 复杂度时，再按 route group 拆注册函数。 | 低 | 待处理 |
| 10 | `internal/bootstrap/app.go` 与 `internal/bootstrap/provider.go` | bootstrap 是生命周期和依赖组装中心，当前合理，但可能随功能增长变胖。 | 暂不优先拆；后续只在 provider 出现明显多职责或循环依赖时再拆。 | 低 | 待处理 |

## 问题 1 分析：CD route

### 参考架构依据

`D:\SourceCodes\mywork\k12-force\docs\analyze\arch.md` 中的分层倾向是：

- `application/` 负责 command/query/workflow orchestration，即业务用例编排。
- `infrastructure/storage/local` 负责本地文件存储。
- `infrastructure` 下承载外部系统实现，如 logger、storage、mq、database、mail、lock 等。
- `workflow/activity` 负责工作流 activity 执行能力。
- `api/http` 负责 HTTP handler、middleware、response，不承载外部系统实现。

据此，`internal/application/cd/route.go` 中的“权限校验、输入校验、Route 状态修改、调用 store 持久化”属于 application 可接受范围；但文件系统、外部命令和 Traefik API 调用属于 infrastructure/execution 细节。

### 当前读取结论

`internal/application/cd/route.go` 当前同时承担以下职责：

1. CD route 应用用例：
   - `ListRoutes`
   - `CreateRoute`
   - `UpdateRoute`
   - `DeleteRoute`
   - `EnableRoute`
   - `DisableRoute`
   - `SyncRoutes`
   - `UploadRouteCert`
   - `DisableRouteHTTPS`
   - `EnableRouteLetsEncrypt`
   - `EnableRouteMkcert`

2. 权限与业务校验：
   - `ensureProjectMembership`
   - `loadRouteForUser`
   - route name/domain/path/target URL 校验
   - Let's Encrypt 配置校验
   - enabled route 禁止删除等业务规则

3. Traefik 动态配置文件实现：
   - `syncRouteFiles`
   - `deployRouteFile`
   - `revokeRouteFiles`
   - `writeRouteCertFiles`
   - `revokeRouteCertFiles`
   - `routeConfigDir`
   - `routeCertDir`
   - `cleanConfigPath`
   - `routeTraefikConfig`

4. 外部进程执行：
   - `reloadTraefik` 使用 `docker ps` 和 `docker kill --signal=HUP`。
   - `generateMkcert` 使用 `mkcert -CAROOT` 与 `mkcert -cert-file ...`。

5. Traefik HTTP API client：
   - `fetchTraefikRouters`
   - `isTraefikConnectionError`
   - `TraefikRouterResp` / `TraefikRouteListResp` 当前作为 application 输出 DTO 使用。

### 问题判断

这个问题成立，且不是纯命名问题。

核心问题不是 `route.go` 太长，而是 application service 直接依赖了：

- `os.MkdirAll` / `os.WriteFile` / `os.Remove` / `os.ReadFile`
- `os/exec`
- `http.Client`
- `yaml.Marshal`
- Docker container reload 细节
- mkcert 命令细节
- Traefik 动态配置目录和证书目录解析

这会让 application 层同时知道业务状态流转和部署环境细节，后续测试、替换 Traefik、替换证书生成方式、迁移部署环境时都会被耦合。

### 处理建议

按用户反馈，本条不做“先小拆、留后续”的保守拆法；要在 issue 1 范围内把边界拆对，基础设施细节一次搬离 application，不留尾巴。行为仍必须保持不变，但实现边界要到位：

1. 在 `internal/application/cd` 中定义 application 依赖的接口，命名可偏业务语义：
   - `RouteConfigPublisher`
     - `Sync(ctx context.Context, route model.Route) error`
     - `Revoke(ctx context.Context, routeName string) error`
     - `WriteCertificate(ctx context.Context, routeName string, certPEM string, certKey string) error`
     - `RevokeCertificate(ctx context.Context, routeName string) error`
   - `RouteCertificateGenerator`
     - `Generate(ctx context.Context, domain string) (certPEM string, keyPEM string, err error)`
   - `TraefikRouterClient`
     - `ListRouters(ctx context.Context) ([]TraefikRouterResp, error)`

2. 将当前 `deployRouteFile` / `revokeRouteFiles` / `writeRouteCertFiles` / `revokeRouteCertFiles` / `reloadTraefik` / `routeTraefikConfig` 迁到 infrastructure 实现。
   - 建议目标：`internal/infrastructure/traefik`。
   - 理由：它不是普通 storage；它包含 Traefik 文件 provider 格式、Docker reload、Traefik 运行目录约定，是外部组件适配器。

3. 将 `fetchTraefikRouters` 迁到同一个 Traefik infrastructure client。
   - application 仍决定“用户必须属于项目才能看 Traefik routes”。
   - infrastructure 只负责调用 Traefik API 并返回中性结构。

4. 将 `generateMkcert` 拆到证书生成 adapter。
   - 可以先放在 `internal/infrastructure/traefik` 或 `internal/infrastructure/cert/mkcert`。
   - 若只服务 route HTTPS，第一步放在 `internal/infrastructure/traefik` 也可接受；若后续证书能力会复用，再独立 `cert/mkcert`。

5. `Service` 构造函数注入这些接口。
   - `New` 使用真实 infrastructure 实现。
   - 测试可以注入 fake，减少 HTTP route tests 对文件系统和外部命令细节的依赖。

### 不建议的做法

- 不建议把整个 `route.go` 移到 `workflow/activity/cd`：route CRUD 是用户请求驱动的应用用例，不是后台执行 activity。
- 不建议为了拆分新增兼容层或新旧逻辑并存。
- 不建议先抽一个巨大的 `RouteManager`，否则只是把大文件换个名字。
- 不建议在 HTTP handler 中直接处理 Traefik 或 mkcert 细节，HTTP 层应保持协议适配。

### 实现边界

只处理 issue 1，不顺手处理 settings、runtime variables、workspace 或 task repository；但在 issue 1 内部不做半截拆分，必须把 Traefik 文件发布、Traefik API client、Docker reload、mkcert 证书生成等基础设施细节完整搬离 application。

实现可接受的文件范围：

- `internal/application/cd/route.go`
- `internal/application/cd/service.go`
- 新增 `internal/infrastructure/traefik/*`
- 相关测试中必要的构造调整
- `internal/bootstrap/provider.go` 或 `internal/api/http/server.go` 中 CD service wiring，如构造函数需要注入 adapter

验收标准：

- [x] application route 用例不再直接 import `os`、`os/exec`、`net/http`、`gopkg.in/yaml.v3`。
- [x] application route 用例中不再直接执行文件写入/删除、Docker reload、mkcert、Traefik HTTP API。
- [x] Route CRUD、enable/disable、manual cert、Let's Encrypt、Traefik route list 既有测试继续通过。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

实施结果：

- 新增 `internal/infrastructure/traefik/route.go`：承载 Traefik dynamic route YAML 生成、route config/cert 文件写入删除、Windows Docker HUP reload、Traefik HTTP API router list client。
- 新增 `internal/infrastructure/traefik/mkcert.go`：承载 mkcert 命令探测、CA 检查和证书生成。
- `internal/application/cd/route.go` 保留 route 用例、权限校验、业务校验、状态流转和 store 调用，通过接口调用基础设施能力。
- `internal/application/cd/service.go` 定义 `RouteConfigPublisher`、`RouteCertificateGenerator`、`TraefikRouterClient` 接口，并在 `Service` 中显式依赖这些接口。
- `internal/api/http/server.go` 负责组装真实 Traefik infrastructure adapter。
- `internal/model/cd.go` 增加 `TraefikRouter` 中性模型，避免 application DTO 绑定 infrastructure HTTP 响应结构。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 2 分析：Settings .env 存储

### 参考架构依据

`D:\SourceCodes\mywork\k12-force\docs\analyze\arch.md` 中的分层倾向是：

- `application/` 负责 command/query/workflow orchestration，即业务用例编排。
- `infrastructure` 承载外部系统和本地存储实现。
- `api/http` 负责 HTTP handler、middleware、response，不承载存储实现。

据此，settings service 中“系统配置项定义、默认值、override 判断、显示值解析”属于 application 可接受范围；但 `.env` 文件路径解析、文件读取、目录创建、文件写入属于 infrastructure 存储细节。

### 当前读取结论

`internal/application/settings/service.go` 原本同时承担：

1. settings 应用用例：
   - `Config`
   - `Update`
   - `Reset`

2. 配置展示与转换规则：
   - `settingDefinitions`
   - `settingDescription`
   - `settingEnvKey`
   - `settingKeyFromEnv`
   - `settingValueString`
   - `parseSettingValue`

3. `.env` 本地文件实现：
   - `envPath`
   - `readEnvFile`
   - `writeEnvValues`
   - `deleteEnvValues`
   - `writeEnvFile`
   - `os.ReadFile` / `os.WriteFile` / `os.MkdirAll`

问题成立：settings application 的业务语义是“系统配置用例”，不是普通 `.env` 文件编辑器；因此 `.env` 文件读写应作为可替换存储能力注入。

### 实施结果

- `internal/application/settings/service.go` 新增 `EnvStore` 接口：
  - `Load(ctx context.Context) (map[string]string, error)`
  - `Set(ctx context.Context, values map[string]string) error`
  - `Delete(ctx context.Context, keys []string) error`
- `settings.Service` 通过 `EnvStore` 注入读取和更新 override 配置。
- `settings.Service` 不再直接 import `os`、`path/filepath`，也不再直接读写 `.env` 文件。
- 新增 `internal/infrastructure/config/envfile/store.go`，承载 `.env` 文件路径解析、读取、写入、删除和目录创建。
- `internal/api/http/server.go` 负责组装 `settingssvc.New(cfg, envfile.NewStore(cfg))`。
- `internal/api/http/settings_routes_test.go` 同步注入真实 envfile store，保持原有 HTTP 行为测试。
- 验证过程中发现 `internal/application/ci/snapshot.go` 和 `internal/application/ci/template.go` 存在重复导入 `internal/common/errors` 导致 typecheck 失败，已按格式化后实际包名清理重复 import；这不是 issue 2 的逻辑变更。

验收标准：

- [x] `internal/application/settings/service.go` 不再直接 import `os`、`path/filepath`。
- [x] `internal/application/settings/service.go` 不再直接执行 `.env` 文件读写、目录创建或路径解析。
- [x] `.env` 解析、写入、删除行为保持原规则：缺失文件返回空 map、忽略空行和注释、按 `=` 拆分、trim quote、写入时按 key 排序并追加尾随换行。
- [x] settings HTTP config get/update/reset 既有测试继续通过。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 3 分析：CI runtime variables 规则

### 参考架构依据

`internal/application/ci` 负责触发流水线、创建 snapshot、校验运行变量等应用用例；`internal/workflow/activity/ci` 负责执行流水线 activity。runtime variable 的内置变量、模板变量提取、变量声明清洗、snapshot 补全和值合并规则同时服务触发侧和执行侧，属于跨 application/activity 的纯规则，不应复制在两侧。

### 当前读取结论

原实现中相近规则分散在：

- `internal/application/ci/runtime_variables.go`：创建 pipeline run 时构建、校验并序列化 runtime variable snapshot。
- `internal/workflow/activity/ci/runtime_variables.go`：执行 pipeline run 时补全 declarations 并构建执行变量。
- `internal/application/ci/template.go` 与 `internal/application/ci/repository.go`：template/repository variable 的清洗、内置变量判断和 template stage 变量提取。
- `internal/workflow/activity/ci/runtime_helpers.go` 与 `internal/workflow/activity/ci/template_builtin.go`：workflow activity 侧的 wrapper 和重复 builtin specs。

问题成立：如果这些规则继续分散在 application 与 workflow activity，两侧可能在内置变量列表、空值判断、default/value 优先级、template_stage 变量处理等细节上漂移。

### 实施结果

- 新增 `internal/common/civariable/runtime.go`，集中承载 CI variable 纯规则：
  - snapshot declarations 补全；
  - runtime variables 构建；
  - runtime variables 校验与 snapshot 序列化；
  - repository/template builtin variables；
  - template variables 解析、清洗、提取和排序；
  - `HasRuntimeValue` 空值判断。
- `internal/application/ci/runtime_variables.go` 只保留 application 用例函数 `buildPipelineRunVariables`，直接调用 `civariable` 规则完成补全、构建、校验和序列化。
- `internal/application/ci/snapshot.go` 创建/读取 snapshot 时直接调用 `civariable.PipelineSnapshotStages`、`civariable.PipelineTemplateVariables`、`civariable.ResolveTemplateVariablesFromStageDefinitions` 和 `civariable.VariableDeclarationsFromMaps`。
- `internal/application/ci/template.go` 与 `internal/application/ci/repository.go` 删除重复的 template variable/builtin 规则实现，改为直接调用 `civariable`；保留 application 专属的 DTO 转换和模板 CRUD 编排。
- `internal/workflow/activity/ci/execution.go` 执行 pipeline run 时直接调用 `civariable.CompleteSnapshotVariableDeclarations`、`civariable.BuildRuntimeVariables`、`civariable.IsPipelineTemplateBuiltinVariable` 和 `civariable.HasRuntimeValue`。
- 删除 wrapper-only 文件：
  - `internal/workflow/activity/ci/runtime_variables.go`
  - `internal/workflow/activity/ci/runtime_helpers.go`
  - `internal/workflow/activity/ci/template_builtin.go`
- `internal/application/ci/template_test.go` 中 template variable 规则测试改为直接覆盖 `civariable.ResolveTemplateVariables`。

验收标准：

- [x] application 与 workflow activity 不再各自维护 runtime variable 规则副本。
- [x] workflow activity 侧不保留仅转调 `civariable` 的占位/兼容 wrapper 文件。
- [x] 内置变量列表、`runtime_datetime` 格式、default/value 优先级、template variable 提取和空值判断保持原规则。
- [x] repository/template/snapshot/run execution 调用点均直接依赖共享规则包。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 4 分析：CI workspace 规则

### 参考架构依据

`internal/application/ci` 负责 pipeline run 查询、stage log 读取等应用用例；`internal/workflow/activity/ci` 负责执行 pipeline run、创建运行目录、写入 stage log、挂载 workspace/artifacts 到 Docker。CI workspace 的逻辑路径规则同时服务两侧，不应在 application 与 workflow activity 中复制。

### 当前读取结论

`internal/application/ci/workspace.go` 与 `internal/workflow/activity/ci/workspace.go` 当前几乎完全一致，均包含：

- `CIWorkspace` 数据结构；
- `NewCIWorkspace` / `newCIWorkspaceWithResolver`；
- `WorkspacePath(projectCode)`；
- `ArtifactsPath(runId)`；
- `StageLogPath(runId, stageRunId)`；
- `CreateRunDirectories(projectCode, runId)`；
- `DockerStageMounts(ctx, projectCode, runId)`；
- `PhysicalDataRoot(ctx)`。

问题成立：workspace 逻辑路径、artifact/log 路径、physical root 解析缓存、Docker mount 生成规则如果继续双份维护，后续很容易出现 application 读 log 路径与 workflow activity 写 log 路径不一致，或 Docker artifact mount 与 artifact 保存路径不一致。

额外发现：`internal/application/ci/runner.go` 与 `internal/workflow/activity/ci/runner.go` 也重复定义 `VolumeMount` / `RunOptions` / `ContainerRunner` / `DockerRunner` 等执行器结构。Issue 4 不扩大为完整 runner 重构，但为了让 workspace 的 Docker mount 返回值能被两侧直接使用，需要把 `VolumeMount` 类型纳入共享 workspace 包或共享执行类型。

### 拆分策略

本 issue 只处理 CI workspace 相关重复，不顺手重构整个 runner：

1. 新增共享包 `internal/infrastructure/storage/local/ciworkspace`。
2. 将 `CIWorkspace` 改名为共享语义更清晰的 `Workspace`。
3. 将 `VolumeMount` 类型放入 `ciworkspace`，让 workspace 直接返回共享 mount 类型。
4. 共享包承载：
   - 逻辑路径规则：workspace、artifacts、stage log；
   - `CreateRunDirectories` 目录创建；
   - `PhysicalDataRoot` 缓存；
   - `DockerStageMounts` 生成 `/workspace`、`/artifacts` bind mount。
5. application 与 workflow activity 的 Service/Executor 直接依赖 `*ciworkspace.Workspace`。
6. 删除 `internal/application/ci/workspace.go` 与 `internal/workflow/activity/ci/workspace.go`，不保留仅转调共享包的 wrapper-only 文件。
7. application/workflow activity 的 runner 保留现有 DockerRunner 行为，仅将 `RunOptions.Volumes` 切换为 `[]ciworkspace.VolumeMount`；不在本 issue 中合并 DockerRunner。
8. 将现有 workflow activity workspace 测试迁移到 `internal/infrastructure/storage/local/ciworkspace`，保持现有断言。

### 实施结果

- 新增 `internal/infrastructure/storage/local/ciworkspace/workspace.go`：集中承载 CI workspace 逻辑路径、artifacts 路径、stage log 路径、运行目录创建、physical root 缓存和 Docker stage mounts。
- 新增 `internal/infrastructure/storage/local/ciworkspace/workspace_test.go`：迁移并保留原 workspace physical root 与 Docker mount 行为断言。
- 删除重复实现：
  - `internal/application/ci/workspace.go`
  - `internal/workflow/activity/ci/workspace.go`
- `internal/application/ci/repository.go` 中的 service workspace 改为 `*ciworkspace.Workspace`，构造时直接使用 `ciworkspace.New(dataRoot)`。
- `internal/workflow/activity/ci/service.go` 与 `internal/workflow/activity/ci/executor.go` 中的 workflow activity workspace 改为 `*ciworkspace.Workspace`。
- `internal/application/ci/runner.go` 与 `internal/workflow/activity/ci/runner.go` 保留各自 Docker runner 行为，只将 `RunOptions.Volumes` 切换为 `[]ciworkspace.VolumeMount`，避免在本 issue 中扩大为 runner 合并。
- `internal/workflow/activity/ci/workspace_test.go` 保留 Docker run args 相关测试，workspace 行为测试迁移到共享包。

### 验收标准

- [x] `internal/application/ci/workspace.go` 删除。
- [x] `internal/workflow/activity/ci/workspace.go` 删除。
- [x] 新共享 workspace 包承载路径规则、目录创建、physical root 缓存和 Docker mount 生成。
- [x] application 与 workflow activity 都直接调用共享 workspace 包，不保留 wrapper-only 兼容层。
- [x] workspace 行为保持不变：
  - logical path 仍为 `dataRoot/ci/<projectCode>/workspace`；
  - artifacts path 仍为 `dataRoot/ci/runs/<runId>/artifacts`；
  - stage log path 仍为 `dataRoot/ci/runs/<runId>/stages/<stageRunId>.log`；
  - Docker mounts 仍映射到 `/workspace` 和 `/artifacts`；
  - Docker host path 仍使用 `ResolvePhysicalDataRoot` 解析后的 physical root。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 5 分析：CD workspace 规则

### 参考架构依据

参考 `D:\SourceCodes\mywork\k12-force\docs\analyze\arch.md`：

- `application/` 负责 command/query/workflow orchestration，即应用用例编排。
- `workflow/activity` 负责工作流 activity 执行。
- `infrastructure/storage/local` 负责本地文件存储。
- `common/` 只放通用错误、常量、工具、校验等，不承载具体本地文件系统布局和 physical host path 解析。

CD workspace 规则依赖 `internal/infrastructure/storage/local.ResolvePhysicalDataRoot`，并描述 `dataRoot/cd/<appCode>` 下的本地文件布局，因此目标包应放在 `internal/infrastructure/storage/local/cdworkspace`，而不是 `common`。

### 当前读取结论

`internal/application/cd/workspace.go` 与 `internal/workflow/activity/cd/workspace.go` 当前实现完全重复，均包含：

- `Workspace` 数据结构；
- `NewWorkspace` / `newWorkspaceWithResolver`；
- `AppDir(appCode)`；
- `DeploymentLogPath(appCode, deploymentId)`；
- `PhysicalDataRoot(ctx)`；
- `PhysicalDir(ctx)`；
- `PhysicalAppDir(ctx, appCode)`。

application 侧使用这些规则删除应用目录、读取部署日志、执行 compose 状态/日志命令、渲染模板 physical path；workflow activity 侧使用同一规则写部署日志、执行 deploy/restart/stop 命令、渲染部署模板。如果继续双份维护，application 读日志路径和 activity 写日志路径、模板中的 physical path 规则都可能漂移。

### 拆分策略

本 issue 只处理 CD workspace 重复实现，不扩大为 command runner、`os.RemoveAll` 或部署执行编排拆分：

1. 新增共享包 `internal/infrastructure/storage/local/cdworkspace`。
2. 共享包承载 CD workspace 逻辑路径、部署日志路径、physical root 解析缓存、slash 格式 physical path 生成。
3. 暴露 `New(dataRoot)` 与 `NewWithResolver(dataRoot, resolver)`；测试使用 `NewWithResolver` 注入 resolver。
4. 保持现有方法语义：
   - `AppDir(appCode)`；
   - `DeploymentLogPath(appCode, deploymentId)`；
   - `PhysicalDataRoot(ctx)`；
   - `PhysicalDir(ctx)`；
   - `PhysicalAppDir(ctx, appCode)`。
5. `internal/application/cd/service.go` 与 `internal/workflow/activity/cd/service.go` 中的 workspace 字段直接改为 `*cdworkspace.Workspace`，构造时直接调用 `cdworkspace.New(cfg.DataRoot())`。
6. 测试中需要注入 resolver 的地方直接使用 `cdworkspace.NewWithResolver(...)`。
7. 删除 `internal/application/cd/workspace.go` 与 `internal/workflow/activity/cd/workspace.go`，不保留 wrapper-only 文件、类型别名、转发构造函数、适配层或新旧逻辑并存。
8. 将现有 workflow activity workspace 行为测试迁移到 `internal/infrastructure/storage/local/cdworkspace/workspace_test.go`，保持现有断言。

### 实施结果

- 新增 `internal/infrastructure/storage/local/cdworkspace/workspace.go`：集中承载 CD app 目录、deployment log 路径、physical root 缓存、slash 格式 physical dir/app dir 生成。
- 新增 `internal/infrastructure/storage/local/cdworkspace/workspace_test.go`：迁移原 workflow activity CD workspace 行为测试，保留 logical path、physical path、resolver cache 和错误传播断言。
- 删除重复实现：
  - `internal/application/cd/workspace.go`
  - `internal/workflow/activity/cd/workspace.go`
  - `internal/workflow/activity/cd/workspace_test.go`
- `internal/application/cd/service.go` 中的 workspace 改为 `*cdworkspace.Workspace`，构造时直接使用 `cdworkspace.New(cfg.DataRoot())`。
- `internal/workflow/activity/cd/service.go` 中的 workspace 改为 `*cdworkspace.Workspace`，构造时直接使用 `cdworkspace.New(cfg.DataRoot())`。
- `internal/application/cd/compose_test.go` 与 `internal/workflow/activity/cd/deployment_execution_test.go` 中需要注入 resolver 的测试直接使用 `cdworkspace.NewWithResolver(...)`。

### 验收标准

- [x] `internal/infrastructure/storage/local/cdworkspace` 新增并承载 CD workspace 规则。
- [x] `internal/application/cd/workspace.go` 删除。
- [x] `internal/workflow/activity/cd/workspace.go` 删除。
- [x] application 与 workflow activity 都直接依赖 `*cdworkspace.Workspace`。
- [x] 不保留 wrapper、别名、适配、兼容转发或新旧逻辑并存。
- [x] workspace 行为保持不变：
  - app dir 仍为 `dataRoot/cd/<appCode>`；
  - deployment log path 仍为 `dataRoot/cd/<appCode>/deployments/<deploymentId>.log`；
  - `PhysicalDir` 仍返回 slash 格式 physical root；
  - `PhysicalAppDir` 仍返回 slash 格式 `<physicalRoot>/cd/<appCode>`；
  - physical root resolver 仍只执行一次并缓存错误。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 6 分析：task runtime model 边界

### 参考架构依据

参考 `D:\SourceCodes\mywork\k12-force\docs\analyze\arch.md`：

- `repository/impl/sqlc` 是 repository 的 sqlc 持久化实现。
- `queue/` 承载消息定义、生产消费、dispatcher、worker、retry、codec 等队列运行时能力。
- `application/` 负责用例编排，不应暴露具体 repository implementation type。
- `api/http` 负责协议适配和响应转换，不应为了 DTO 转换依赖 sqlc impl。

因此 background task / queue item / worker dispatch payload 这种 runtime model 不应定义在 `internal/repository/impl/sqlc/task`；它属于 queue 边界。

### 当前读取结论

`internal/repository/impl/sqlc/task/repository.go` 当前定义的 `Task` 已被多层直接使用：

- `internal/queue/task/service.go` 的 service 返回 `*taskrepo.Task`。
- `internal/worker/worker.go` 的 `Handler`、`HandlerFunc`、`Repository` 使用 `taskrepo.Task`。
- `internal/worker/handler/ci/handler.go` 与 `internal/worker/handler/cd/handler.go` 的 `Handle` 参数使用 `taskrepo.Task`。
- `internal/api/http/handler/task/handler.go` 的 `taskResponse` 使用 `*taskrepo.Task`。
- `internal/application/ci/repository.go` 与 `internal/application/cd/service.go` 的 `TaskService` 接口返回 `*taskrepo.Task`。

这说明 `Task` 不是 sqlc impl 私有模型，而是跨 queue/worker/API/application 使用的 runtime model。继续放在 sqlc impl 下会让上层为了任务模型依赖具体数据库实现包。

### 拆分策略

本 issue 只处理 task runtime model 的边界，不扩大为 task API、worker 生命周期、retry 策略或任务类型常量重构：

1. 新增 `internal/queue/task/model.go`，定义 `Task` runtime model。
2. 删除 `internal/repository/impl/sqlc/task/repository.go` 中的本地 `Task` 定义。
3. `internal/repository/impl/sqlc/task` 直接返回 `*tasksvc.Task`，只保留 SQL/transaction/SQLC row 转换职责。
4. `internal/queue/task/service.go` 使用本包 `Task`，不再 import sqlc task repo 只是为了模型类型。
5. `internal/worker/worker.go`、`internal/worker/handler/ci/handler.go`、`internal/worker/handler/cd/handler.go` 直接使用 queue task model。
6. `internal/api/http/handler/task/handler.go` 的 response 转换直接接收 `*tasksvc.Task`。
7. `internal/application/ci/repository.go` 与 `internal/application/cd/service.go` 的 `TaskService` 接口返回 `*tasksvc.Task`。
8. 测试中构造 task 的地方直接改为 `tasksvc.Task{...}`。
9. 不保留 `type Task = ...`、wrapper-only 文件、适配层或新旧模型并存。

### 实施结果

- 新增 `internal/queue/task/model.go`，定义跨 queue/worker/API/application 使用的 `Task` runtime model。
- `internal/repository/impl/sqlc/task/repository.go` 删除本地 `Task` 定义，`ClaimNext`、`FindById` 与 `taskFromSQLC` 直接使用 `*tasksvc.Task` / `tasksvc.Task`。
- `internal/queue/task/service.go` 使用本包 `Task`，不再为了模型类型 import sqlc task repository。
- `internal/worker/worker.go`、`internal/worker/handler/ci/handler.go`、`internal/worker/handler/cd/handler.go` 直接使用 queue task model。
- `internal/api/http/handler/task/handler.go` 的 response 转换直接使用 `*tasksvc.Task`。
- `internal/application/ci/repository.go` 与 `internal/application/cd/service.go` 的 `TaskService` 接口返回 `*tasksvc.Task`。
- worker 相关测试构造 task 时直接使用 `tasksvc.Task{...}`。

### 验收标准

- [x] `internal/repository/impl/sqlc/task/repository.go` 不再定义 `Task`。
- [x] `Task` model 位于 `internal/queue/task/model.go`。
- [x] queue service、worker、worker handlers、HTTP task handler、application CI/CD 不再为了 `Task` 类型 import `internal/repository/impl/sqlc/task`。
- [x] sqlc task repository 直接返回 `*tasksvc.Task`，只保留持久化与 row conversion 职责。
- [x] 不保留类型别名、wrapper-only 文件、适配层或新旧模型并存。
- [x] 任务入队、claim、complete、fail、HTTP create/get task、CI/CD worker handler 行为不变。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 7 分析：logger/logstore 命名与边界

### 参考架构依据

参考 `D:\SourceCodes\mywork\best-practices\docs\guides\backend-service-architecture.md`：

- `infrastructure/logger` 用于日志实现，典型职责是日志初始化与适配。
- `infrastructure/storage` 用于本地/S3 对象存储等文件或对象存储实现。
- `application` 负责用例编排，可以记录用例级关键节点，但不应依赖具体基础设施实现类型；需要外部能力时应按“谁消费，谁定义”定义端口。
- `workflow`、`queue`、`scheduler` 是执行机制，不应沉淀业务核心；执行日志写入属于执行过程的技术支撑能力，应通过清晰端口使用。
- 日志相关建议区分：Handler 记录请求级信息，Application 记录用例级关键节点，Infrastructure 记录外部调用失败、耗时、状态码、重试次数。

据此，需要区分两类概念：

1. application/server runtime logger：当前 `internal/infrastructure/logger/logging` 初始化 `slog.Logger` 和 rolling file writer。
2. CI/CD execution log storage：当前 `internal/infrastructure/logger/logstore` 读写 pipeline stage / deployment 的业务执行日志文件。

二者虽然都叫“log”，但职责不同。后者更像本地文件存储 adapter，不是 logger 实现。

### 当前读取结论

当前 logger 相关包只有：

- `internal/infrastructure/logger/logging/logging.go`
  - `logging.New(cfg config.LoggingConfig) (*slog.Logger, func() error, error)`；
  - 校验 logging config；
  - 创建日志目录；
  - 使用 `lumberjack.Logger` 写应用运行日志；
  - 设置 `slog` 默认 logger。

- `internal/infrastructure/logger/logstore/logstore.go`
  - `LogStore.Writer(logPath string) (io.WriteCloser, error)`；
  - `LogStore.Read(logPath string, offset int) ([]byte, int, error)`；
  - `LogStore.Exists(logPath string) bool`。

`logstore.LogStore` 实际调用点：

- application 查询 execution log：
  - `internal/application/ci/pipeline_run.go` 使用 `Read` 读取 stage log；
  - `internal/application/cd/service.go` 使用 `Read` 读取 deployment log。
- workflow activity 写 execution log：
  - `internal/workflow/activity/ci/executor.go` 使用 `Writer` 写 stage log，失败时使用 `Read` 取最后日志片段；
  - `internal/workflow/activity/cd/deployment_execution.go` 使用 `Writer` 写 deploy/restart/stop log。
- 组装：
  - `internal/bootstrap/provider.go` 和 `internal/api/http/server.go` 构造 `logstore.LogStore{}` 并注入。

额外确认：`LogStore.Exists` 当前没有调用点。

### 问题判断

Issue 7 成立，但性质是“小边界/命名修正”，不是需要大迁移的核心架构问题。

问题不在 `LogStore` 的读写逻辑本身；当前 `Writer` 创建目录并 append 写文件、`Read` 按 offset 增量读取且缺失文件返回空内容，这些行为符合 execution log 存储需求。

真正的问题有两点：

1. 包路径误导：
   - `internal/infrastructure/logger/logging` 表达 runtime logger 初始化。
   - `internal/infrastructure/logger/logstore` 实际表达 execution log 文件存储。
   - 把 execution log storage 放在 logger 下，会让“应用运行日志”和“CI/CD 执行日志文件存储”两个概念混在一起。

2. concrete infrastructure type 穿透 application/workflow 构造签名：
   - `application/ci`、`application/cd`、`workflow/activity/ci`、`workflow/activity/cd` 的 Service 字段和构造函数直接使用 `logstore.LogStore`。
   - 按 best-practices 中“谁消费，谁定义接口”和 application 不依赖具体 infrastructure 实现的原则，调用侧应定义自己需要的最小端口：读取侧只需要 `Read`，写入侧需要 `Writer`，CI executor 需要 `Writer` + `Read`。

### 拆分策略

本 issue 只处理 execution log storage 的包边界与最小端口，不扩大为日志系统、CI/CD workspace 或 execution 编排重构：

1. 新增 `internal/infrastructure/storage/local/executionlog`。
2. 将 `LogStore` 改名为 `Store`，放入新包，表达 execution log 的本地文件存储实现。
3. 保持行为不变：
   - `Writer(logPath)` 创建父目录；
   - 写入仍为 append/create/write-only；
   - writer 仍用 mutex 串行化 `Write` 和 `Close`；
   - `Read(logPath, offset)` 仍从 offset 读取全部新增内容；
   - 文件不存在仍返回 `nil, offset, nil`。
4. 删除未使用的 `Exists` 方法，不迁移无调用点 API。
5. 在消费侧定义最小接口，而不是让 application/workflow 直接暴露 concrete store：
   - application CI/CD 查询侧定义 `LogReader`：`Read(logPath string, offset int) ([]byte, int, error)`；
   - workflow CD execution 定义 `LogWriter`：`Writer(logPath string) (io.WriteCloser, error)`；
   - workflow CI executor 需要写入并读取失败尾部日志，可定义 `LogStore` 或 `ExecutionLogStore` 包内接口，包含 `Writer` + `Read`。
6. `bootstrap/provider.go` 与 `api/http/server.go` 使用 `executionlog.Store{}` 作为 concrete implementation 注入。
7. 删除 `internal/infrastructure/logger/logstore/logstore.go`，不保留旧包、类型别名、转发构造或兼容 wrapper。
8. 不改 `internal/infrastructure/logger/logging`，因为它职责明确：初始化应用 runtime logger。

### 不建议的做法

- 不建议把 execution log storage 放进 `common`：它有具体文件存储和 CI/CD execution log 语义，不是跨项目通用工具。
- 不建议只把 `logstore` 改名后仍留在 `internal/infrastructure/logger`：这样仍混淆 runtime logger 与 execution log storage。
- 不建议为了测试方便保留 `type LogStore = executionlog.Store`：这违反当前“不做兼容或适配”的约束。
- 不建议在本 issue 合并 CI/CD workspace 或改 deployment/stage log 路径规则；路径规则已分别归属 `ciworkspace` 和 `cdworkspace`。
- 不建议重构 `logging.New` 或 slog/lumberjack 初始化；这不是 issue 7 的问题。

### 实现边界

只处理 issue 7，不顺手处理 issue 8 proto、issue 9 server 或 issue 10 bootstrap。

实现可接受的文件范围：

- 新增 `internal/infrastructure/storage/local/executionlog/store.go`。
- 删除 `internal/infrastructure/logger/logstore/logstore.go`。
- 更新以下调用点的 import、字段类型和构造参数：
  - `internal/application/ci/repository.go`
  - `internal/application/cd/service.go`
  - `internal/workflow/activity/ci/service.go`
  - `internal/workflow/activity/ci/executor.go`
  - `internal/workflow/activity/cd/service.go`
  - `internal/bootstrap/provider.go`
  - `internal/api/http/server.go`
  - 相关测试文件。
- 如需新增消费侧接口，应放在消费包内，不新增共享 wrapper 包。
- 更新本 requirement 文档的 issue 7 实施结果与验收状态。

### 实施结果

- 新增 `internal/infrastructure/storage/local/executionlog/store.go`：承载 execution log 本地文件读写实现，保留原 `Writer` / `Read` 行为。
- 删除 `internal/infrastructure/logger/logstore/logstore.go`：不保留旧包、类型别名、转发 wrapper 或兼容层。
- `internal/application/ci/repository.go` 与 `internal/application/cd/service.go` 新增消费侧 `LogReader` 接口，只暴露 application 查询 execution log 所需的 `Read` 能力。
- `internal/workflow/activity/ci/service.go` 新增消费侧 `ExecutionLogStore` 接口，覆盖 CI executor 写 stage log 与失败时读取日志尾部所需的 `Writer` + `Read` 能力。
- `internal/workflow/activity/cd/service.go` 新增消费侧 `LogWriter` 接口，只暴露 CD execution 写部署日志所需的 `Writer` 能力。
- `internal/bootstrap/provider.go` 与 `internal/api/http/server.go` 改为构造并注入 `executionlog.Store{}`。
- workflow activity 相关测试改为直接使用 `executionlog.Store{}`。
- 未修改 `internal/infrastructure/logger/logging`，保持 runtime logger 初始化职责不变。
- 未迁移原 `Exists` 方法，因为实现前确认无调用点。

验收标准：

- [x] `internal/infrastructure/logger/logstore/logstore.go` 删除。
- [x] `internal/infrastructure/storage/local/executionlog` 新增并承载 execution log 本地文件读写实现。
- [x] `internal/infrastructure/logger/logging` 保持只负责应用 runtime logger 初始化，不混入 execution log storage。
- [x] application CI/CD 不再直接依赖 concrete `executionlog.Store`；只依赖本包消费侧读取接口。
- [x] workflow activity CI/CD 不再直接依赖旧 `logger/logstore` 包；写入/读取 execution log 通过本包消费侧最小接口完成。
- [x] 未保留旧包、类型别名、转发 wrapper、兼容层或新旧实现并存。
- [x] execution log 行为保持不变：缺失文件读取返回空内容与原 offset；写入自动创建目录并 append；offset 读取返回新 offset。
- [x] 未修改 CI/CD workspace 路径规则、deployment/stage execution 编排、应用 runtime logger 配置或 API contract。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## Processing rules

- 每次只处理一条 issue。
- 进入实现前，先读取该 issue 相关文件和调用点。
- 如果读取后发现判断不成立，记录为“不处理/无需拆分”，不要为了拆分而拆分。
- 如果需要迁移代码，保持行为不变，但不要为了降低改动量而保留已识别的分层尾巴；issue 内部要拆到正确边界，再运行后端验证命令。
- 每条完成后更新本表状态，并在对应 verification 文档中记录验证结果。

## Decisions

- 后续拆分从高优先级开始，建议第一条处理 `internal/application/cd/route.go`。
- 对 application/activity 重复 helper 的处理原则是“先确认规则归属，再抽共享规则”，不先做机械去重。
- 对 infra 细节进入 application 的处理原则是“application 依赖接口，具体文件系统/外部命令实现放到 infrastructure 或 execution 层”。
- 用户纠正：issue 1 不采用“小拆分、后续再补”的方式；要在本 issue 范围内大搬家，拆对，不留基础设施尾巴。

## Open questions

- 第一条是否按建议从 `internal/application/cd/route.go` 开始；如果用户指定其他编号，则以用户指定为准。
- `proto/` 根目录是否要纳入本轮逐条拆分，取决于后续生成链路检查结果。

## Risk / Assumption

- 每条拆分都可能触及调用链和测试，需要单独验证，不能用总体验证代替单项验收。
- 迁移后当前已能通过检查；后续拆分应避免为了“架构纯净”引入不必要抽象。
- 当前任务处于活跃开发期，不做兼容层或新旧逻辑并存。

## User review notes

- 用户要求：light 模式。
- 用户要求：开始拆分任务，但不一次做完。
- 用户要求：以上的点过一条、拆一条。
- 用户要求：先记录问题和处理建议。
