# 领域拆分收尾验证
最后修改时间: 2026-07-24 17:39:00

Verification status: Passed

## Scope

本记录合并此前按阶段维护的 Pipeline 术语、Gateway use case、Deployment use case、Application family 总切换和历史命名 / 边界审计验证。验收对象是 [领域拆分收尾需求](../requirement/20260724-domain-split-closeout.md) 与 [统一实施计划](../plan/20260724-domain-split-closeout.md) 中定义的最终生产边界。

## Phase results

| 阶段 | 验收结论 | 证据 |
| --- | --- | --- |
| 1. Pipeline 术语 | 通过 | 生产 Go 代码不再使用 `model.BuildStage`、`model.StageRun`、`BuildStageCreateInput` 或旧主 DTO / use case 标识符；Pipeline / PipelineRun 的 SQL query、sqlc 生成物、HTTP、worker 和定向测试已同步为 `PipelineStage`、`PipelineStageRun` 与 `pipeline_stage_runs`。 |
| 2. Gateway use case | 通过 | `application/gateway` 已包含 DTO、port、use case 和直接测试；配置 CRUD、暴露投影、Traefik Compose 编译、公开端口冲突均通过窄 port 实现，不依赖 HTTP、Proto、sqlc implementation 或 infrastructure。 |
| 3. Deployment use case | 通过 | deployment command、dispatch、worker 执行、状态流转和故障分支有测试；Gateway 的配置准备、冲突校验和滚动决策经 `gateway/port.DeploymentCoordinator` 协作，Deployment 未复制 Gateway 规则。 |
| 4. Service runtime binding | 通过 | service 覆盖 project list、单项读取、application-scoped 查询、primary service 与 target resolver；生产 use case 不依赖 Proto、HTTP、sqlc implementation 或 application use case。 |
| 5. Application family 切换 | 通过 | HTTP、worker 和 dispatcher 已注入 gateway、deployment、service 各自领域服务；application 收缩为 application / version 规格，旧执行、Gateway 与 service runtime 实现已移除。 |
| 6. 历史命名与边界审计 | 通过 | 旧 `ci` / `cd` HTTP、SPA、任务值、数据目录和协议兼容面均未保留；基础设施归位到 pipeline / deployment runner、workspace 与 HTTP security，跨层 Proto import 扫描无生产残留。 |

## Contract and architecture checks

- 任务类型为 `pipeline_run.execute`、`deployment.deploy`、`deployment.restart`、`deployment.stop`；旧任务值和 payload 不提供解析或回退。
- HTTP 和 SPA 只使用领域路径；旧 `ci` / `cd` 地址仅允许作为 404 等负向测试用例。
- application、model、Repository、infrastructure 的生产 Go 文件不导入 `internal/gen/proto`；Proto 仅由 transport adapter 使用，sqlc Row 仅在 `impl/sqlc` 实现层使用。
- bootstrap 是唯一组合根；routes 只注册，handler / worker / dispatcher 仅适配领域用例。
- 不存在 `runner/ci`、`runner/cd`、`ciworkspace`、`cdworkspace`、`dockerci`、粗粒度 `model/ci.go`、`model/cd.go` 或无职责的 `common.go`。

## Command results

| 命令或检查 | 结果 |
| --- | --- |
| `./bin/sqlc.exe generate` | 通过 |
| `./bin/buf.exe generate` | 通过 |
| `./bin/buf.exe generate . --template web/buf.gen.yaml --output web` | 通过 |
| `go vet ./...` | 通过 |
| `go test ./...` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `yarn --cwd web test` | 通过，4 个测试文件、17 个测试 |
| `./bin/golangci-lint.exe run ./cmd/... ./internal/... ./sql` | 通过，0 issues |
| `git diff --check` | 通过 |
| 旧术语、旧目录、旧路径、旧任务值与跨层 Proto import 扫描 | 未发现生产代码残留 |

## Residual risk

本次为硬切换，不支持旧 API、SPA 地址、任务值、目录或协议字段。升级与回退必须以完整版本回退处理，不能在运行期混用两套领域契约。

## Conclusion

需求规定的领域归属、依赖方向、契约切换和验证门禁均已满足；领域拆分收尾完成。
