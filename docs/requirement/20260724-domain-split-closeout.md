# 领域拆分收尾
最后修改时间: 2026-07-24 17:39:00

Review status: Accepted

## Background

领域拆分已将 HTTP、Proto、sqlc 和部分 application 代码迁入资源领域目录，但仍需完成业务术语、application family 接线和历史 `ci` / `cd` 双桶的收口。本需求定义收尾后的唯一生产边界；此前按任务拆分的实施文档由同名 Plan 统一替代。

领域边界、物理组织和依赖方向以 [领域拆分共识](../analyze/20260724-domain-split-consensus-共识.md) 为准。

## Goal

1. 以 `pipeline`、`pipeline_run`、`application`、`service`、`deployment`、`gateway` 等资源领域取代 `ci` / `cd` 流程桶，形成唯一生产实现路径。
2. 让 application、service、deployment 和 gateway 各自拥有明确的 use case、DTO、port 与入站调用面；组合和注入只留在 bootstrap。
3. 将历史名称、HTTP / SPA 路径、任务类型、本地数据目录和协议字段一次性切换到领域契约，不保留运行时兼容层。
4. 保持分层依赖：adapter -> application -> model / ports；Proto 和 sqlc 细节不得渗入 application 或 model。

## Scope

### Pipeline vocabulary

- 可复用流水线阶段的业务主名称为 `PipelineStage`；单次运行中的阶段实例为 `PipelineStageRun`。
- model、DTO、Repository port、use case、SQL query、sqlc 生成物、HTTP mapper / handler、worker 和测试都使用该术语。
- pipeline run 的对外字段为 `pipeline_stage_runs`。旧 `BuildStage`、`StageRun` 主类型和旧字段不作为兼容入口保留。

### Application family ownership

| 领域 | 拥有职责 | 不拥有职责 |
| --- | --- | --- |
| application | application 与 version 规格、校验、导入导出 | service 运行时查询、deployment command / execution、gateway 配置与编译 |
| service | application x environment x version 的运行时绑定读取、目标解析和 view | deployment 历史与操作、version 规格 |
| deployment | deploy / restart / stop / cancel / log、异步 command、执行状态流转与 worker 协调 | application version 生命周期、Gateway 规则实现 |
| gateway | gateway 配置、暴露投影、Traefik Compose 编译、公开端口冲突校验 | application、service、deployment 的资源生命周期 |

- 跨领域读取或执行能力通过所属领域声明的窄 application port 提供；不得注入另一个领域的具体 use case、HTTP、Proto、sqlc implementation、runner 或 infrastructure。
- deployment 必须通过 gateway port 使用 Gateway 协调能力，不复制 Gateway 编译、暴露冲突或受管监听规则。
- HTTP handler、worker handler 与 queue dispatcher 是 adapter；routes 只注册路径；bootstrap 是唯一组合根。

### Naming and transport cutover

- 生产内部业务 package、DTO、model、port、Repository、基础设施目录、HTTP / SPA 路径、任务值和本地数据目录不得继续以 `ci` / `cd` 作为聚合边界。
- 不注册旧 API URL、SPA route、redirect、alias、回退解析、双写或旧 JSON 字段兼容。旧路径仅可在负向测试中出现。
- 任务类型使用 `pipeline_run.execute`、`deployment.deploy`、`deployment.restart`、`deployment.stop`。
- 基础设施按实际技术职责命名，例如 pipeline / deployment runner、workspace 及 HTTP security；不得引入 `common.go` 或新的无边界 `common` 桶。
- 物理模型文件按领域拆分，禁止继续使用 `model/ci.go`、`model/cd.go` 作为跨领域容器。

### Layering and generated contracts

- Proto 生成类型只由 HTTP、gRPC、worker task 等 transport adapter 使用；application、model、Repository 和 infrastructure 不得导入 `internal/gen/proto`。
- sqlc Row、生成 query 参数和数据库方言细节只留在 `repository/impl/sqlc/<domain>`；adapter 通过 mapper 将协议类型转换为 application command、query 和 view。
- SQL 和 migration 的名称、生成配置与领域地图保持一致；需要更名时使用保留数据的迁移策略，不能以删除重建表替代。

## Non-goals

- 不为旧调用方、旧书签、旧任务 payload 或历史字段提供兼容路径。
- 不借本需求重组 API 单复数、引入 API 版本前缀或改变无关 DTO 语义。
- 不将跨领域能力重新收纳到 application、`ci`、`cd` 或 `common` 桶。

## Acceptance criteria

1. application 仅保留 application / version 规格；service、deployment、gateway 的 HTTP、worker 和 dispatch 调用面均指向各自领域服务，且没有重复生产写入路径。
2. `PipelineStage`、`PipelineStageRun`、`pipeline_stage_runs` 和新的任务类型在生产契约中一致；旧主标识符、路径、任务值、目录和字段没有正向兼容面。
3. application、model、Repository、infrastructure 不导入 `internal/gen/proto`；use case 不依赖 HTTP、sqlc implementation 或具体外部 runner。
4. 重新生成 Proto 与 sqlc 后，生成物、领域地图和实际目录一致；不保留粗粒度 `ci` / `cd` 模型文件或无职责的 `common.go`。
5. 定向领域测试、`go vet ./...`、`go test ./...`、前端 typecheck / test、静态旧路径与依赖扫描均通过。

## Evidence

- [统一实施计划](../plan/20260724-domain-split-closeout.md)
- [统一验收记录](../verification/20260724-domain-split-closeout.md)
