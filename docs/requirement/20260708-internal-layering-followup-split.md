# internal 分层后续逐条拆分
最后修改时间: 2026-07-08 15:37:50

Review status: Draft

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
| 2 | `internal/application/settings/service.go` | settings application service 直接读写 `.env` 文件，应用层与本地配置文件存储耦合。 | 先确认 settings 的业务语义是“系统配置用例”还是“.env 文件编辑器”；若仍保留应用用例，应将文件读写抽为配置存储接口，具体 `.env` 实现放到 infrastructure/config 或 infrastructure/storage/local。 | 高 | 待处理 |
| 3 | `internal/application/ci/runtime_variables.go` 与 `internal/workflow/activity/ci/runtime_variables.go` | runtime variable 逻辑跨触发与执行，迁移后 application/activity 两侧存在相近逻辑，可能出现重复和规则漂移。 | 对比两份实现的输入输出与调用点；将纯规则计算沉到 shared domain/common 位置，application 与 activity 只调用同一规则，避免复制。 | 高 | 待处理 |
| 4 | `internal/application/ci/workspace.go` 与 `internal/workflow/activity/ci/workspace.go` | CI workspace 路径、artifact/log 路径和 Docker mount 计算跨 application/activity 使用，当前可能有重复实现。 | 先梳理 application 与 activity 各自需要的 workspace 能力；如果是纯路径规则，抽到公共 workspace/path 组件；如果涉及执行环境，保留在 workflow activity。 | 中 | 待处理 |
| 5 | `internal/application/cd/workspace.go` 与 `internal/workflow/activity/cd/workspace.go` | CD workspace 路径计算与物理数据根解析跨 application/activity 使用，当前边界需要确认。 | 与 CI workspace 同步审视；把稳定路径规则与执行时副作用分离，避免 application 依赖 activity 私有实现。 | 中 | 待处理 |
| 6 | `internal/repository/impl/sqlc/task/repository.go` | task repository 实现里包含 Task 持久化模型，repository impl 与模型定义边界不清。 | 读取文件确认模型是否只服务 sqlc impl；若为跨层任务模型，应迁到 `internal/model` 或 `internal/queue/task`；sqlc impl 只保留转换和持久化。 | 中 | 待处理 |
| 7 | `internal/infrastructure/logger/logstore/logstore.go` | logstore 已放入 logger，但需要确认 logging 与 logstore 的 package/API 命名是否表达清晰。 | 检查 logger/logging/logstore 三者调用关系；若只是命名问题，优先小范围重命名或补清晰接口，不做大迁移。 | 低 | 待处理 |
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
