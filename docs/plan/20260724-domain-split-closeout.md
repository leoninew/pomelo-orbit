# Plan：领域拆分收尾
最后修改时间: 2026-07-24 17:39:00

Review status: Accepted

## Basis

| 文档 | 用途 |
| --- | --- |
| [requirement](../requirement/20260724-domain-split-closeout.md) | 最终业务边界、兼容策略与验收要求 |
| [domain consensus](../analyze/20260724-domain-split-consensus-共识.md) | 领域地图、物理组织和依赖方向 |
| [route cutover workboard](20260724-domain-route-cutover-00-workboard.md) | 后续 API / SPA 路径硬切换的固定路径地图 |

本 Plan 合并并替代此前 `20260724-domain-split-closeout-00` 至 `06` 的分片任务文档。阶段 1--4 可并行实现，阶段 5 是共享装配点的唯一切换，阶段 6 执行全局审计和硬切换；所有阶段已按此顺序完成。

## Delivery order

| 阶段 | 工作项 | 可并行 | 前置 | 共享高冲突范围 | 状态 |
| --- | --- | --- | --- | --- | --- |
| 1 | Pipeline 术语收尾 | 是 | 无 | pipeline / pipeline_run 模型、SQL 和调用面 | 完成 |
| 2 | Gateway use case 提取 | 是 | 无 | 新建 `application/gateway/` | 完成 |
| 3 | Deployment command 与 worker use case 提取 | 是 | 无 | 新建 `application/deployment/` | 完成 |
| 4 | Service runtime binding use case 补全 | 是 | 无 | 新建 `application/service/` | 完成 |
| 5 | Application family 总切换 | 否 | 1、2、3、4 | bootstrap、routes、handler、dispatch、worker | 完成 |
| 6 | 历史命名与边界审计 | 否 | 5 | 领域边界、数据契约与全局静态检查 | 完成 |

阶段 1--4 不修改 `internal/bootstrap/http.go`、`internal/bootstrap/worker.go`、`internal/api/http/routes/routes.go` 或 `internal/queue/dispatch/dispatch.go`，并且不删除仍由旧 application service 使用的代码。阶段 5 统一接线和删除旧实现；阶段 6 才完成无兼容的全局清理。

## Implementation phases

### Phase 1: Pipeline vocabulary

1. 将 `BuildStage` 重命名为 `PipelineStage`，将 `StageRun` 重命名为 `PipelineStageRun`；同步 model、DTO、Repository port、use case、mapper、handler、worker、测试、SQL query 与 sqlc 生成物。
2. 将 SQL query 和生成物收敛为 `pipeline_stage`、`pipeline_stage_run`，并将对外 pipeline run 字段收敛为 `pipeline_stage_runs`。
3. 不用 type alias 掩盖旧主名称；协议路径或持久化对象如有迁移需要，只在 adapter / migration 中处理，不能让旧名回流 application 或 model。

完成记录：生产 Go 代码已不含 `model.BuildStage`、`model.StageRun`、`BuildStageCreateInput` 或旧主 DTO / use case 标识符；Pipeline / PipelineRun 的定向测试、生成检查与全量 Go 测试已通过。

### Phase 2: Gateway use case

1. 在 `internal/application/gateway/{dto,port,usecase}` 建立 gateway 配置 CRUD、暴露投影、Traefik Compose 编译和公开端口冲突校验。
2. 将 `GatewayCreateInput`、`GatewayUpdateInput`、`GatewayView` 与暴露投影归入 gateway DTO；通过窄 port 读取 project、application / version、运行 service、gateway 配置和 workspace。
3. 保持 `kind=gateway` 的 application 是部署载体，但 gateway 不是 application 子聚合，也不拥有其他领域的生命周期。

完成记录：Gateway 用例不依赖 HTTP、Proto、sqlc implementation 或 infrastructure；配置、编译、投影和端口冲突均有直接测试。

### Phase 3: Deployment command and worker use case

1. 在 `internal/application/deployment/{dto,port,usecase}` 集中 deploy / restart / stop / cancel / log、命令创建、异步 dispatch 输入、执行状态流转、Compose 渲染和 worker 协调。
2. 以 deployment port 表达 application、version、environment、service、workspace、runner、execution log、dispatcher 和 Repository 能力；application 不再作为执行总线。
3. 通过 `gateway/port.DeploymentCoordinator` 使用 Gateway 的配置准备、暴露冲突、受管监听和滚动决策，避免在 deployment 复制 Gateway 业务规则。

完成记录：命令创建、dispatch、worker 执行、状态流转、Gateway 协作及故障分支均有测试；Deployment 用例不依赖 application 具体实现、HTTP、Proto、sqlc 或 runner 实现。

### Phase 4: Service runtime binding use case

1. 在 `internal/application/service/{dto,usecase}` 建立 application x environment x version 的运行时绑定读取、application-scoped 列表、primary service、target resolver 与对外 view。
2. application 和 environment 仅以读取 port 提供校验和展示信息；停止、重启和部署 command 始终归 deployment。
3. 对外结果以 application view 表达应用、环境和版本标签，不泄露 SQL join Row 给 adapter。

完成记录：service use case 已覆盖 project list、单项读取、application-scoped 查询、primary service 与 target resolver，且无 Proto、HTTP、sqlc implementation 或 application use case 依赖。

### Phase 5: Application family cutover

1. 在 `bootstrap/http.go`、`bootstrap/worker.go` 和 `bootstrap/stores.go` 构造领域 service 和 port，实现 HTTP、worker 与 queue dispatch 的唯一注入路径。
2. 将 gateway handler 指向 gateway service，deployment handler 与 worker 指向 deployment service，service handler 和 application-scoped service 查询指向 service service。
3. 将 dispatcher 输入和 task message 映射迁到 deployment DTO / port；task 表只保存通用队列信封。
4. 删除 application 中已迁出的 gateway、deployment、worker、Compose runtime 和 service 查询实现与旧 DTO，确保 application 只保留 application / version 规格。
5. 不注册旧 URL、alias、redirect 或旧 task value / JSON payload 回退；不因 URL 含 `application` 而继续委派 application service。

完成记录：bootstrap 是唯一组合根；handler、worker 和 dispatcher 不再导入 application 的旧 use case / DTO；application 不再依赖 environment、service、deployment、gateway Repository、workspace、dispatcher、runner 或 query runner。

### Phase 6: Legacy cleanup and boundary audit

1. 按实际技术职责将基础设施归位为 pipeline / deployment runner、workspace 和 HTTP security；拆分 `model/ci.go`、`model/cd.go` 等粗粒度模型文件。
2. 将 HTTP / SPA 路径、任务类型、持久化 task payload、本地数据目录、Traefik 证书目录和协议字段一次性切换到领域名称；不保留 alias、fallback parser、redirect 或双写。
3. 审计依赖方向：adapter -> application -> model / ports；Proto 仅在 transport，sqlc Row 仅在实现层；不创建 `common.go` 或无边界 `common` 桶。

完成记录：任务类型为 `pipeline_run.execute`、`deployment.deploy`、`deployment.restart`、`deployment.stop`；旧 `ci` / `cd` HTTP、SPA、任务值和数据目录未注册；`authz` 已归入 HTTP security，模型与基础设施目录均按领域或技术职责命名。

## Validation

### Required commands

```text
./bin/sqlc.exe generate
./bin/buf.exe generate
./bin/buf.exe generate . --template web/buf.gen.yaml --output web
go vet ./...
go test ./...
yarn --cwd web typecheck
yarn --cwd web test
./bin/golangci-lint.exe run ./cmd/... ./internal/... ./sql
git diff --check
```

### Static gates

```text
rg -n 'model\\.(BuildStage|StageRun)|BuildStageCreateInput|StageRun' internal --glob '*.go'
rg -n 'internal/gen/proto' internal/application internal/model internal/repository internal/infrastructure --glob '*.go'
rg -n '/api/(ci|cd)(/|$)|/ci/|/cd/' internal web/src --glob '!**/*_test.go'
rg -n 'pipeline\\.execute|deployment\\.(deploy|restart|stop)|ciworkspace|cdworkspace|runner/(ci|cd)' internal web sql --glob '!**/*_test.go'
```

除有明确用途的负向测试或历史文档外，上述扫描不得有生产代码匹配。生成、定向领域测试、全量 Go 测试、前端 typecheck / test、lint 和 diff 检查均已通过；具体命令结果见[统一验收记录](../verification/20260724-domain-split-closeout.md)。

## Closeout

- [x] Pipeline vocabulary
- [x] Gateway use case
- [x] Deployment command and worker use case
- [x] Service runtime binding use case
- [x] Application family cutover
- [x] Legacy naming and boundary audit
- [x] Code generation, static gates and full validation
