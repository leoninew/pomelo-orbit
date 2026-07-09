# internal 分层后续逐条拆分
最后修改时间: 2026-07-09 13:16:15

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
| 8 | 根目录 `proto/` 与 `internal/gen/proto` | `internal/gen/proto` 是生成代码，根目录 `proto/` 是否作为 IDL 源目录保留尚未形成明确决策。 | 已完成分析：根目录 `proto/` 是 Buf module 指定的 IDL 源目录，同时生成 Go `internal/gen/proto` 与前端 `web/src/gen/proto`；该布局符合架构指南中“契约源文件在根目录 proto、生成代码在 gen 隔离层”的约束，不应强行迁入 `internal`，也不需要兼容层或适配层。 | 低 | 已分析，建议不迁移 |
| 9 | `internal/api/http/server.go` | HTTP server 当前不只是 Gin 入站适配器，还承担 repository impl、application service、infra adapter 的构造与部分重复 service 组装，边界超过 `api/http` 职责。 | 已完成实施：repository/service/infrastructure adapter 组装已移到 `bootstrap/provider.go`；`api/http.Server` 改为接收已装配依赖并只注册 Gin middleware/routes、health、static fallback；Cloudflare Turnstile concrete verifier 已迁到 `internal/infrastructure/turnstile`；孤儿 root HTTP route 集成测试已清理，仅保留与 `api/http` 源码职责对应的 server/static、codec、middleware、authz 测试。 | 低 | 已实施，检查通过 |
| 10 | `internal/bootstrap/app.go` 与 `internal/bootstrap/provider.go` | bootstrap 同时承载进程生命周期编排、数据库迁移入口、HTTP server/worker 启动协调、repository/service/infra provider 组装；组合根职责本身合理，但当前 `app.go` 的生命周期控制与 `provider.go` 的对象装配边界还可以更清晰，避免后续继续膨胀。 | 已完成实施：保留 bootstrap 作为唯一组合根；database/migration、repository store、HTTP server factory、worker factory、runtime lifecycle 已在 bootstrap 内部分文件承载；删除 provider 大杂烩文件，不保留空占位、兼容构造或新旧装配路径；bootstrap 新增 runtime lifecycle 测试，测试仍跟随源码职责。 | 低 | 已实施，检查通过 |

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

## 问题 8 分析：proto IDL 源目录与生成代码边界

### 参考架构依据

参考 `D:\SourceCodes\mywork\best-practices\docs\guides\backend-service-architecture.md`：

- 根目录 `proto/` 是推荐的接口契约源文件（IDL）目录，按产品或服务命名空间与版本组织，例如 `proto/<product>/v1/<domain>.proto`。
- `internal/gen/proto` 是 Protobuf 生成代码目录，属于 `gen` 隔离层；生成代码只作为边界类型或底层辅助，不写手工业务逻辑。
- Proto / OpenAPI Schema 是外部契约，DTO 是应用层输入输出，Model 是领域概念，三者不要混用。
- Proto 生成类型主要出现在 `api`、契约转换层或少量 application DTO 边界；不应贯穿到 repository、model 或基础设施内部业务逻辑。
- 生成目录可以整体重建，不能成为手写逻辑依赖点；但 IDL 源文件是契约资产，不应因为后端 Go 代码位于 `internal` 就强行迁入 `internal`。

据此，判断 issue 8 的关键不是“根目录下出现 `proto/` 是否违反 internal 分层”，而是确认：

1. `proto/` 是否确实是 IDL 源目录；
2. `internal/gen/proto` 是否只是生成产物；
3. 生成产物是否只在允许的边界使用；
4. 是否存在需要拆分或迁移的真实边界问题。

### 当前读取结论

当前 proto 生成链路清晰：

- `buf.yaml` 明确声明 Buf module path 为根目录 `proto`：
  - `version: v2`；
  - `modules: [{ path: proto }]`。
- 根目录 `proto/orbit/v1/*.proto` 是 IDL 源文件，当前包含 `application.proto`、`repository.proto`、`pipeline_run.proto`、`common.proto` 等业务契约文件。
- `.proto` 文件使用 `package orbit.v1`，文件内部 import 形态为 `import "orbit/v1/<name>.proto"`，与 `buf.yaml` 的 module root `proto` 匹配。
- `buf.gen.yaml` 配置 Go 生成：
  - `managed.enabled: true`；
  - `go_package_prefix` 为 `gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto`；
  - `out: internal/gen/proto`；
  - `paths=source_relative`。
- `web/buf.gen.yaml` 配置 TypeScript 生成：
  - remote plugin `buf.build/community/stephenh-ts-proto:v2.11.8`；
  - `out: src/gen/proto`；
  - 由 `task proto` 执行时通过 `--output web` 落到 `web/src/gen/proto`。
- `Taskfile.yml` 的 `proto` task 是当前显式生成入口：
  - `./bin/buf dep update`；
  - `./bin/buf generate`；
  - `./bin/buf generate . --template web/buf.gen.yaml --output web`。
- `tools.go` 固定了 Go 侧 protoc-gen-go 工具依赖。
- `internal/gen/proto/orbit/v1/*.pb.go` 文件头为 `Code generated by protoc-gen-go. DO NOT EDIT.`，且 `source: orbit/v1/<name>.proto`，确认它们是从根目录 `proto` module 生成的产物。
- `web/src/gen/proto/orbit/v1/*.ts` 是前端生成 DTO，前端 API 与页面以 `@/gen/proto/orbit/v1/...` 引用。

当前使用边界也基本符合架构约束：

- Go 生成 proto 类型主要出现在 `internal/api/http` handler、codec 和 HTTP route tests 中，用于 HTTP JSON 契约 decode/encode 与测试断言。
- 未发现 `application`、`model`、`repository/impl/sqlc`、`workflow/activity` 等核心业务/持久化/执行层直接依赖 `internal/gen/proto/orbit/v1`。
- 前端生成类型位于 `web/src/gen/proto`，作为 web API client 与页面展示类型使用，属于前端契约边界。

### 问题判断

Issue 8 的初始疑问经检查后结论是：根目录 `proto/` 应保留，不需要迁移，也不应强行迁入 `internal`。

理由：

1. `proto/` 是契约源目录，不是后端运行时代码目录。
   - 它服务 Go HTTP 适配层和前端 TypeScript 两端生成。
   - 把它迁入 `internal` 会把跨端契约伪装成 Go 后端内部实现，反而弱化“外部契约源文件”的边界表达。

2. `internal/gen/proto` 是生成代码隔离层，当前位置符合架构指南。
   - 生成 Go package 位于 `internal/gen/proto`，避免被模块外部直接 import。
   - 文件头和 Buf 配置都表明该目录可重建，不承载手写业务逻辑。

3. 当前生成类型的使用范围基本正确。
   - Go 侧集中在 `api/http` 协议适配边界。
   - 没有发现 proto 类型穿透到 `model`、repository impl、workflow activity 或 application 用例编排。

4. 强行迁移会制造不必要的大范围改动。
   - 需要同时调整 `buf.yaml`、`buf.gen.yaml`、proto import path、生成输出、前端生成命令和大量导入。
   - 该改动不解决真实分层问题，风险高于收益。

### 处理建议

本 issue 建议记录为“无需代码迁移 / 保留现有布局”，只补充边界决策，不做兼容或适配层：

1. 保留根目录 `proto/` 作为唯一 IDL 源目录。
2. 保留 `internal/gen/proto` 作为 Go Protobuf 生成代码目录。
3. 保留 `web/src/gen/proto` 作为前端 TypeScript proto DTO 生成目录。
4. 后续新增契约文件继续放在 `proto/orbit/v1` 或新版本目录，例如 `proto/orbit/v2`。
5. 后续修改 proto 后使用现有显式入口 `task proto` 重新生成 Go 与前端代码。
6. 确保 proto 源文件和生成文件在指定目录集中管理：
   - IDL 源文件集中在 `proto/orbit/v1` 或后续版本目录；
   - Go generated files 集中在 `internal/gen/proto`；
   - 前端 TypeScript generated files 集中在 `web/src/gen/proto`；
   - 不在业务目录散落 `.proto` 源文件或手写 generated 文件。
7. 继续约束生成类型使用范围：
   - Go proto generated types 只应出现在 `internal/api/http`、codec、契约 mapper 或协议测试；
   - 不要把 generated proto 类型传入 `model`、repository impl、workflow activity 或核心 application 用例；
   - 如 application 需要稳定输入输出，使用 application DTO 或 model，而不是直接复用 proto message。
8. 不新增 wrapper、别名、适配层，也不保留新旧路径并存，因为本 issue 不需要路径迁移。

### 不建议的做法

- 不建议把 `proto/` 移到 `internal/proto`：IDL 是跨端契约源，不是 Go internal runtime 包。
- 不建议把 Go generated code 移出 `internal/gen/proto`：当前后端生成代码应由本模块内部 HTTP 边界消费，不需要对外暴露 Go import path。
- 不建议新增 `internal/api/proto`、`internal/common/proto` 或 wrapper package：这只会增加 indirection，并不能改善边界。
- 不建议为了“目录都在 internal 下”改 Buf module root；这会破坏当前 `orbit/v1/...` import 约定并引入前后端同步风险。
- 不建议在本 issue 中重写 HTTP handler 的 proto DTO mapper；当前问题只检查 proto 源/生成目录边界，不扩大为 API contract 重构。

### 实现边界

本 issue 暂无产品代码实现项。若后续进入 Verification，只需验证：

- `buf.yaml` 仍以 `proto` 为 module root；
- `Taskfile.yml` 的 `proto` task 仍同时生成 Go 与 TypeScript DTO；
- `internal/gen/proto` 与 `web/src/gen/proto` 均为生成目录；
- `internal/gen/proto` import 使用范围没有穿透到核心业务层；
- requirement 文档记录本决策。

### 验收标准

- [x] 已确认根目录 `proto/` 是 Buf module 的 IDL 源目录。
- [x] 已确认 `internal/gen/proto` 是 Go Protobuf generated code 隔离目录。
- [x] 已确认 `web/src/gen/proto` 是前端 TypeScript generated DTO 目录。
- [x] 已确认现有生成入口为 `task proto` / `Taskfile.yml` 中的 `proto` task。
- [x] 已确认当前 Go proto generated types 主要停留在 HTTP 协议适配边界。
- [x] 已明确不迁移 `proto/`，不新增兼容层、别名、wrapper 或适配层。
- [x] 已明确 proto 源文件与生成文件必须在指定目录集中管理，不在业务目录散落。

## 问题 9 分析：HTTP server 路由与依赖组装边界

### 参考架构依据

参考 `D:\SourceCodes\mywork\best-practices\docs\guides\backend-service-architecture.md`：

- `api/http` 是 Gin 入站适配器，职责是创建 Gin engine、注册 middleware/routes、绑定协议输入、映射协议错误、调用 application service。
- Gin 类型只能停留在 HTTP 入站适配层；Handler 不直接依赖数据库、Redis、消息队列、第三方 SDK 或 repository 实现。
- `bootstrap` 是组合根，允许看见所有层，负责选择 repository implementation、infrastructure adapter，并把它们注入 application 或 adapter。
- `bootstrap` 不承载业务规则，但可以负责“把对象接起来”。
- `api/http` 可以依赖 application 接口、协议契约、common；不应依赖具体数据库、repository impl 或外部 SDK client。
- 外部 API client 属于 infrastructure；例如 Cloudflare Turnstile verification 这种第三方 HTTP 调用不应长期沉淀在 `api/http` 包内。

据此，判断 issue 9 的关键不是 `server.go` 是否“长”，而是它是否把组合根职责、基础设施实现选择、Gin 路由适配职责混在一起。

### 当前读取结论

`internal/api/http/server.go` 当前承担了几类职责：

1. HTTP/Gin 入站适配职责，属于 `api/http` 合理范围：
   - `gin.SetMode(gin.ReleaseMode)`；
   - `gin.New()`；
   - 注册 RequestID、RealIP、LogRequest、Recovery、CORS middleware；
   - 注册 `/api/health`；
   - 调用各 domain handler 的 `Register...Routes`；
   - `NoRoute` 中区分 API 404 与前端 SPA fallback；
   - `serveStatic` / `serveIndexHTML` / `injectRuntimeConfig` 为 HTTP 静态资源适配与 runtime config 注入。

2. 组合根 / 依赖选择职责，当前放在 `api/http` 中，边界不理想：
   - `New` 接收 sqlc store 与 task repository，然后在 `api/http` 内部 new 各 domain repository：
     - `userrepo.NewRepository`；
     - `rolerepo.NewRepository`；
     - `projectrepo.NewRepository`；
     - `cirepo.NewRepository`；
     - `cdrepo.NewRepository`。
   - `New` 在 `api/http` 内部 new application services：
     - auth / role / user / project / settings / CI / CD service；
     - settings service 还直接组装 `envfile.NewStore(cfg)`；
     - CI/CD service 组装 `executionlog.Store{}`；
     - CD service 通过 `newCDService` 组装 `traefik.NewRouteManager` 与 `traefik.MkcertGenerator{}`。
   - `Server` struct 存放 repository impl、application service、token service、task service、infra logStore 等混合依赖。

3. 重复或漂移风险：
   - `New` 已经构造了 `ciService` / `cdService` 并放入 `Server` 字段；但 `Handler()` 又重新构造局部 `ciService` / `cdService`，实际注册路由使用的是局部 service。
   - 这不是行为 bug，但说明 server 同时承担“保存依赖”和“重新组装依赖”的职责，后续新增 domain 时容易出现字段未用、重复组装、测试替换困难等问题。

4. 第三方 verification adapter 边界：
   - `internal/api/http/turnstile.go` 当前实现了 Cloudflare Turnstile HTTP client；auth handler 依赖 `TurnstileVerifier` 接口是合理的，但 concrete verifier 的 HTTP 调用实现长期放在 `api/http` 下不符合 infrastructure 外部 client 边界。
   - 本 issue 可以把 concrete Turnstile verifier 一并作为 server dependency 由 bootstrap 注入；若进入实现，建议将 concrete 实现迁到 `internal/infrastructure/turnstile` 或同类明确 infrastructure 包。

### 问题判断

Issue 9 成立，而且不只是“后续可能膨胀”的文件长度问题。

当前 `server.go` 的 Gin route/middleware/static 适配职责本身合理；真正需要拆的是：

- repository implementation 选择不应在 `api/http`；
- application service 构造不应在 `api/http`；
- infrastructure adapter 构造不应在 `api/http`；
- handler 注册可继续留在 `api/http`，但应拆成清晰的私有注册函数，避免 `Handler()` 继续膨胀。

这类拆分应把职责移到正确边界，而不是新增“兼容构造函数”或保留新旧组装路径并存。

### 拆分策略

建议后续实施时一次性把 issue 9 的边界拆清楚，但不扩大为 issue 10 的 bootstrap 生命周期重构：

1. 调整 `transporthttp.Server` 的构造方式。
   - `api/http` 不再接收 sqlc store / task repository 并自行选择 repository implementation。
   - `Server` 接收已装配好的 HTTP handler 所需依赖，例如 application services、authenticator 所需 store/token service、turnstile verifier、config、logger。
   - 可以定义 `ServerDependencies` 或同等结构承载依赖，但不保留旧 `New(cfg, logger, store, taskRepo, maxAttempts)` 兼容签名。

2. 将 repository/service/infra adapter 组装移到 `bootstrap/provider.go`。
   - `bootstrap` 构造 user/role/project/ci/cd repositories。
   - `bootstrap` 构造 auth/role/user/project/settings/ci/cd/task application services。
   - `bootstrap` 构造 envfile、executionlog、traefik route manager、mkcert generator 等 infrastructure adapter。
   - `bootstrap` 将这些已装配依赖传给 `transporthttp.New`。

3. 消除 `Server` 中未使用或重复组装的 service 字段。
   - `Handler()` 不再重新 new CI/CD service。
   - `Server` 只保存路由注册实际需要的依赖。
   - 如果字段只用于构造某个 handler，就由对应私有 register 函数消费，避免重复构造。

4. 将 `Handler()` 拆成小的私有方法，保持 Gin adapter 内聚但降低膨胀风险。
   - `newRouter` / `registerMiddleware`；
   - `registerHealthRoutes`；
   - `registerAuthRoutes`；
   - `registerUserRoleRoutes` 或分开 user/role；
   - `registerSettingsRoutes`；
   - `registerProjectRoutes`；
   - `registerCIRoutes`；
   - `registerCDRoutes`；
   - `registerTaskRoutes`；
   - `registerFallbackRoutes`。

5. 处理 Turnstile concrete verifier 边界。
   - auth handler 已依赖 `TurnstileVerifier` 接口；保持消费侧接口。
   - concrete Cloudflare Turnstile HTTP client 不继续放在 `api/http`。
   - 建议迁到 `internal/infrastructure/turnstile`，由 bootstrap 构造后注入 HTTP server/auth handler。
   - 不保留 `api/http` 下的转发 wrapper 或旧 constructor。

6. 保持静态资源与 runtime config 注入在 HTTP adapter 边界。
   - `serveStatic`、`serveIndexHTML`、`injectRuntimeConfig` 属于 HTTP/static web adapter，不是 application 业务逻辑。
   - 可按文件拆到 `static.go` 降低 `server.go` 体积，但不需要移出 `api/http`。

### 不建议的做法

- 不建议只按行数把 `server.go` 机械拆成多个文件，而保留 repository/service/infra adapter 构造在 `api/http`。
- 不建议新增 `NewWithDependencies` 同时保留旧 `New`：这会形成新旧组装逻辑并存，违反“不做兼容或适配层”。
- 不建议把 route registration 下沉到 `bootstrap`：Gin route/middleware 注册仍属于 HTTP adapter。
- 不建议让 HTTP handler 直接依赖 sqlc repository 或 infrastructure concrete type；handler 应依赖 application service 与少量消费侧端口。
- 不建议把 static web serving 移到 application；这是协议/静态资源适配，不是用例编排。
- 不建议在本 issue 同时重构 `bootstrap.App` 的 serve/worker 生命周期；那属于 issue 10 范围。

### 实现边界

如果用户确认进入实现，issue 9 可接受改动范围：

- `internal/api/http/server.go`
- 可能新增 `internal/api/http/routes.go`、`internal/api/http/static.go` 或类似私有拆分文件
- `internal/api/http/turnstile.go` 删除或迁出
- 新增 `internal/infrastructure/turnstile/*`（如迁移 concrete Turnstile verifier）
- `internal/bootstrap/provider.go` 中 HTTP server 依赖组装
- `internal/api/http/server_test.go` 及受构造签名影响的 HTTP tests

不在 issue 9 中处理：

- 不改 API route path 或 request/response contract。
- 不改 handler 内部业务映射逻辑。
- 不改 application service 行为。
- 不改 repository SQL/mapper。
- 不改 worker lifecycle / bootstrap app lifecycle。
- 不改前端。

### 验收标准

- [x] `internal/api/http/server.go` 不再 import sqlc repository implementation packages。
- [x] `internal/api/http/server.go` 不再构造 application services、repository implementation 或 infrastructure adapters。
- [x] HTTP server 只负责 Gin engine、middleware、route registration、static fallback 和 HTTP adapter 依赖持有。
- [x] `bootstrap/provider.go` 负责选择并装配 HTTP server 所需 repository/service/infrastructure dependencies。
- [x] `Handler()` 不再重复构造 CI/CD services。
- [x] route registration 被拆为清晰私有函数或文件，新增 route group 时不继续扩张单个 `Handler()` 大函数。
- [x] Cloudflare Turnstile concrete HTTP client 不再作为 concrete implementation 留在 `api/http`；auth handler 仍通过接口消费。
- [x] 不保留旧 `transporthttp.New` 签名、兼容 wrapper、别名或新旧构造路径并存。
- [x] API 路由、middleware 顺序、static SPA fallback、runtime config 注入行为保持不变。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

### 实施结果

- `internal/api/http/server.go` 删除 repository impl、envfile、executionlog、traefik、jwt 等组合根 import；新增 `ServerDependencies`，`New` 只接收已装配依赖。
- `Handler()` 不再 new CI/CD services，改为调用 `registerMiddleware`、`registerHealthRoutes`、`registerAuthRoutes`、`registerUserRoutes`、`registerRoleRoutes`、`registerSettingsRoutes`、`registerProjectRoutes`、`registerCIRoutes`、`registerCDRoutes`、`registerTaskRoutes`、`registerFallbackRoutes`。
- `internal/bootstrap/provider.go` 负责构造 token service、sqlc repositories、application services、envfile store、executionlog store、Traefik route manager、mkcert generator、Turnstile verifier，并注入 `transporthttp.New`。
- Cloudflare Turnstile concrete HTTP verifier 迁到 `internal/infrastructure/turnstile`，原 `internal/api/http/turnstile.go` 与测试删除，不保留 wrapper 或旧构造路径。
- `internal/test/e2e/mysql_e2e_test.go` 改用 `bootstrap.NewHTTPServer`，避免测试继续调用旧 `transporthttp.New` 签名。
- 根据用户反馈清理 root `internal/api/http/*_routes_test.go` 孤儿测试；`api/http` 仅保留与当前源码职责直接对应的 `server_test.go`、`codec`、`middleware`、`handler/authz` 测试。被删除的 route 集成测试主要覆盖 application service / repository / filesystem 业务组合，不再放在 HTTP adapter 根包中伪装成 HTTP 层测试；其中需要保留的业务覆盖已按源码归属重建到 `internal/application/{auth,cd,ci,project,role,settings,user}` 的 service integration tests。

验证结果：

```text
go fmt ./cmd/... ./internal/...
./bin/golangci-lint fmt ./cmd/... ./internal/...
./bin/golangci-lint run ./cmd/... ./internal/...
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
```

结果：通过，`golangci-lint run` 输出 `0 issues.`。

## 问题 10 分析：bootstrap 生命周期与 provider 边界

### 参考架构依据

参考 `D:\SourceCodes\mywork\best-practices\docs\guides\backend-service-architecture.md`：

- `bootstrap` 是组合根，允许看见所有层，职责是初始化数据库、HTTP server、queue/worker、workflow 等运行时组件，选择 repository implementation 和 infrastructure adapter，并把实现注入 application 或 adapter。
- `bootstrap` 不承载业务规则，只负责“把对象接起来”和管理资源关闭顺序。
- `cmd/*` 只负责进程级参数、配置/日志初始化、根 context 与信号处理，然后调用 `bootstrap`；不应直接 new handler、repository 或第三方 client。
- `api/http` 是 Gin 入站适配器，不应承担 repository/service/infrastructure 选择。
- `queue`、`workflow`、`scheduler` 是执行机制，不是业务核心；它们的 runtime 装配可以由 bootstrap 完成。
- 测试替身和测试归属应按源码职责放置：某个包的生命周期/装配行为由该包测试，application 业务覆盖放 application，HTTP 协议适配放 api/http，基础设施 client 行为放 infrastructure。

据此，Issue 10 的判断重点不是“bootstrap 依赖很多包是否违规”。作为组合根，bootstrap 看见 repository impl、infrastructure、api/http、worker、workflow activity 是合理的；需要判断的是当前 bootstrap 内部是否已经把不同组合根子职责混在同一个文件/函数中，导致后续继续增长时难以维护或测试。

### 当前读取结论

当前 `internal/bootstrap` 只有三个源码文件：

- `app.go`
  - `App` 保存 `config.Config` 和 `*slog.Logger`。
  - `Migrate()` 打开数据库、执行迁移并关闭连接。
  - `MigrationVersion()` 打开数据库、读取迁移版本并关闭连接。
  - `RunWorker(ctx)` 打开数据库、执行迁移、构造 task repository/store/router/worker 并运行 worker。
  - `Serve(ctx)` 打开数据库、执行迁移、构造 HTTP server、构造 background worker，同时协调 HTTP server 与 background worker 的生命周期、错误传播和 shutdown。

- `provider.go`
  - `OpenDatabase` / `RunMigrations` / `MigrationVersion` 包装 database infrastructure。
  - `NewRepositoryStore` / `NewTaskRepository` 构造 sqlc store 和 task repository。
  - `NewHTTPServer` 构造 HTTP server 所需依赖：token service、user/role/project/ci/cd repositories、application services、envfile store、executionlog store、Traefik route manager、mkcert generator、Turnstile verifier。
  - `NewTaskRouter` 构造 CI/CD workflow activity execution services 与 worker handlers。
  - `NewWorker` 构造 worker runtime。

- `app_test.go`
  - 目前只覆盖 `App.Migrate()` 与 `App.MigrationVersion()`。

调用点当前也比较集中：

- `cmd/server/main.go` 只调用 `bootstrap.New(cfg, logger)`，再按命令调用 `Serve(ctx)` 或 `RunWorker(ctx)`。
- `cmd/migrate/main.go` 只调用 `bootstrap.New(cfg, logger)`，再调用 `MigrationVersion()` 或 `Migrate()`。
- `internal/test/e2e/mysql_e2e_test.go` 调用 `bootstrap.NewHTTPServer(...)` 做 e2e HTTP server 装配。

### 问题判断

Issue 10 部分成立，但不是“跨层依赖错误”。

合理部分：

- `bootstrap/provider.go` import repository impl、infrastructure adapter、application services、worker handlers、workflow activity 是组合根职责，符合架构指南。
- `App.Serve()` 同时启动 HTTP server 与 background worker，是当前 server 进程模式下的生命周期编排，不属于 application 业务逻辑。
- `cmd/server` 和 `cmd/migrate` 目前保持薄入口，没有直接装配 repository、handler 或 infrastructure client。

需要改善的部分：

1. `provider.go` 已经变成“所有 provider 放一起”的大杂烩。
   - database/migration provider、repository provider、HTTP server dependency provider、task router provider、worker provider 全在一个文件。
   - Issue 9 后 HTTP server 组装集中到 bootstrap 是正确方向，但继续全塞进 `provider.go` 会让组合根内部也失去结构。

2. `app.go` 的 `Serve()` 同时承载多件生命周期细节。
   - 打开数据库、执行迁移、构造 HTTP server、构造 worker、创建 `http.Server`、启动两个 goroutine、协调 cancel/shutdown/error channel。
   - 这些仍属于 bootstrap，但可拆为 bootstrap 内部 runtime/lifecycle helper，避免 `App` 方法继续膨胀。

3. database/migration helper 命名与归属可以更明确。
   - `OpenDatabase`、`RunMigrations`、`MigrationVersion` 目前在 `provider.go`，但它们更像 bootstrap database/migration resource helpers。
   - 可以移到 `database.go` / `migration.go`，仍保留在 bootstrap 包内。

4. 测试覆盖跟源码职责还不完整。
   - 目前 bootstrap test 只覆盖 migration happy path。
   - 如果拆分 lifecycle/provider，应把测试放在 `internal/bootstrap`，只覆盖装配和生命周期协调，不把 application 业务测试塞回 bootstrap。
   - 用户已明确“测试用例跟着源码走”：application service 测业务，api/http 测 HTTP adapter，infrastructure 测 external/local adapter，bootstrap 测 composition/lifecycle。

因此 Issue 10 建议处理的是 bootstrap 内部职责切分和测试归属，不是把 bootstrap 职责移出 bootstrap，也不是抽一个额外 DI/compat layer。

### 拆分策略

本 issue 可以实施，但要控制边界：只在 `internal/bootstrap` 内按职责重排组合根代码，不改变外部行为，也不把业务逻辑搬进 bootstrap。

建议拆分方向：

1. 保留 `bootstrap.App` 作为对 `cmd` 暴露的进程应用入口。
   - `cmd/server` 和 `cmd/migrate` 继续只面对 `bootstrap.New(cfg, logger)` 与 `App` 方法。
   - 不新增 `NewAppV2`、`NewWithProviders`、旧构造兼容层或并存路径。

2. 将 database / migration helper 从 generic `provider.go` 中拆出。
   - 目标文件建议：`internal/bootstrap/database.go`。
   - 放置：`OpenDatabase`、`RunMigrations`、`MigrationVersion`。
   - 仍只是 bootstrap 对 database infrastructure 的薄封装，不新增接口适配层。

3. 将 HTTP server 装配拆到专门文件。
   - 目标文件建议：`internal/bootstrap/http.go`。
   - 放置：`NewHTTPServer` 及其私有 helper。
   - 可以把 HTTP dependencies 组装拆成私有 `newHTTPServerDependencies(...)`，但不要导出额外兼容入口。
   - 保持 `bootstrap.NewHTTPServer(...)` 作为当前 e2e 装配入口是否保留需谨慎判断：它是当前源码真实入口，不是兼容旧签名；若保留，语义应是“bootstrap HTTP server factory”。如果实施时发现只有 e2e 使用且可改为更高层 App 测试，也可以收敛为私有函数，但不要留下新旧两套。

4. 将 worker / task router 装配拆到专门文件。
   - 目标文件建议：`internal/bootstrap/worker.go`。
   - 放置：`NewTaskRouter`、`NewWorker` 及 workflow activity execution service 装配。
   - 如果 `RunWorker` 和 `Serve` 都需要同样 worker runtime，保持单一路径，避免 server 进程和 worker 命令使用不同装配逻辑。

5. 将 `Serve()` 中 HTTP + worker 并发生命周期协调抽成 bootstrap 内部 runtime helper。
   - 目标文件建议：`internal/bootstrap/runtime.go` 或保留在 `app.go` 的私有函数。
   - 可抽出私有函数，例如 `runHTTPAndWorker(ctx, logger, httpServer, worker)`。
   - 该 helper 只处理 context、shutdown、error propagation，不知道业务 service。
   - 不启动/停止开发服务器；这里只是产品代码生命周期逻辑，不在分析阶段执行。

6. 保持 `provider.go` 聚焦“跨组件 provider”或删除。
   - 如果上述拆分后 `provider.go` 为空，应删除，不保留空占位。
   - 如果仍保留，应只放真正跨多个 runtime 共用的 provider；不要保留 wrapper-only 文件。

7. 测试跟随源码职责。
   - `internal/bootstrap/app_test.go` 可继续覆盖 `Migrate` / `MigrationVersion`。
   - 新增或调整 bootstrap tests 时，只测：migration helper、HTTP server factory 装配可生成 handler、worker router 注册/worker config、runtime helper 在 HTTP/worker 失败时能 cancel/shutdown。
   - 不把已迁到 application 的业务覆盖搬回 bootstrap。
   - 不在 `api/http` root 恢复 route 集成测试来覆盖 service/repository 业务。
   - Turnstile verifier 继续由 `internal/infrastructure/turnstile` 测；HTTP static/fallback 继续由 `internal/api/http` 测；application service 行为继续由 `internal/application/*` 测。

### 不建议的做法

- 不建议把 bootstrap provider 下沉回 `api/http`，这会回退 Issue 9。
- 不建议把 repository/service/infrastructure 组装移到 `cmd/server` 或 `cmd/migrate`，cmd 应保持薄入口。
- 不建议引入 Wire/Dig/Fx 等 DI 框架；当前问题是文件和职责组织，不是缺少 DI 容器。
- 不建议新增 provider interface、factory interface、`NewWith...` 等测试专用抽象；这会形成不必要适配层。
- 不建议把 `App.Serve()` 拆成两个对外入口后让调用方自己协调 HTTP 和 worker；生命周期协调属于 bootstrap。
- 不建议在本 issue 改 worker 执行业务流、workflow activity 业务规则、task retry/lease/concurrency 语义。
- 不建议为了“测试方便”恢复 root HTTP route integration tests；测试应跟随源码职责。

### 实现边界

如果用户确认进入实现，Issue 10 可接受改动范围：

- `internal/bootstrap/app.go`
- `internal/bootstrap/provider.go`
- 可新增：
  - `internal/bootstrap/database.go`
  - `internal/bootstrap/http.go`
  - `internal/bootstrap/worker.go`
  - `internal/bootstrap/runtime.go`
- `internal/bootstrap/app_test.go` 或新增 bootstrap test 文件。
- 如 HTTP server factory 可见性调整影响 e2e，可调整 `internal/test/e2e/mysql_e2e_test.go`，但不把 e2e 改成覆盖业务细节。
- 更新本 requirement 文档的 Issue 10 状态、实施结果和验收标准。

不在 Issue 10 中处理：

- 不改 API route path 或 request/response contract。
- 不改 application service 行为。
- 不改 repository SQL/mapper。
- 不改 worker handler / workflow activity 执行业务逻辑。
- 不改 task retry、lease、concurrency 语义。
- 不改 config schema。
- 不改前端。
- 不改数据库迁移。

### 验收标准

- [x] `bootstrap` 仍是唯一组合根；repository/service/infrastructure 装配不回流到 `api/http`、`application` 或 `cmd`。
- [x] `cmd/server` 与 `cmd/migrate` 保持薄入口，不直接 new repository、handler、worker handler 或 infrastructure client。
- [x] database/migration helper 与 repository store factory、HTTP server factory、worker factory、runtime lifecycle 职责在 bootstrap 内部分文件或私有函数中清晰分离。
- [x] 不保留空 provider、wrapper-only 文件、兼容构造、别名或新旧装配路径并存。
- [x] `App.Serve()` 行为保持不变：启动 HTTP server 与 background worker；任一失败时取消另一侧；context 取消时 shutdown HTTP；正常退出返回原有语义。
- [x] `App.RunWorker()` 行为保持不变：打开数据库、执行迁移、构造 task router/worker 并运行。
- [x] `App.Migrate()` / `App.MigrationVersion()` 行为保持不变。
- [x] 测试用例跟随源码职责：bootstrap 只测试 composition/lifecycle，application/api/infrastructure 继续测试各自业务或 adapter；不恢复 root HTTP orphan route tests。
- [x] 后端命令继续通过：`go fmt ./cmd/... ./internal/...`、`./bin/golangci-lint fmt ./cmd/... ./internal/...`、`./bin/golangci-lint run ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...`。

### 实施结果

- `internal/bootstrap/database.go` 新增 database/migration helper：`OpenDatabase`、`RunMigrations`、`MigrationVersion`。
- `internal/bootstrap/repository.go` 新增 repository store factory：`NewRepositoryStore`、`NewTaskRepository`。
- `internal/bootstrap/http.go` 新增 HTTP server factory：`NewHTTPServer` 与私有 `newHTTPServerDependencies`，继续由 bootstrap 选择 repository/service/infrastructure implementation 并注入 `api/http`。
- `internal/bootstrap/worker.go` 新增 worker factory：`NewTaskRouter`、`NewWorker`，集中装配 worker router、CI/CD workflow activity execution service 与 worker handlers。
- `internal/bootstrap/runtime.go` 新增 runtime lifecycle helper：`runHTTPServerAndWorker`，集中处理 HTTP server 与 background worker 并发运行、context cancel、HTTP shutdown 和 error propagation。
- `internal/bootstrap/app.go` 保留对 `cmd` 暴露的 `App` 生命周期入口，移除 HTTP + worker 并发协调细节，继续调用单一路径的 bootstrap factories。
- `internal/bootstrap/provider.go` 删除；拆分后没有保留空占位、wrapper-only 文件或兼容路径。
- `internal/bootstrap/runtime_test.go` 新增 lifecycle 测试，覆盖 HTTP serve error 时取消 worker、context cancel 时 shutdown 并返回 context error；测试仍留在 bootstrap 包内。

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
- `proto/` 根目录已在 issue 8 中确认保留为 IDL 源目录；proto 源文件与生成文件按指定目录集中管理，不做迁移。

## Risk / Assumption

- 每条拆分都可能触及调用链和测试，需要单独验证，不能用总体验证代替单项验收。
- 迁移后当前已能通过检查；后续拆分应避免为了“架构纯净”引入不必要抽象。
- 当前任务处于活跃开发期，不做兼容层或新旧逻辑并存。

## User review notes

- 用户要求：light 模式。
- 用户要求：开始拆分任务，但不一次做完。
- 用户要求：以上的点过一条、拆一条。
- 用户要求：先记录问题和处理建议。
