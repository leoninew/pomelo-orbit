# 本地 Docker MCP 部署控制与产品验证记录
最后修改时间: 2026-07-26 15:13:13

## Review status

Draft

## Verification scope

本次只验证 Git index 中的 8 个暂存 Go 文件。验证快照由 `git checkout-index` 导出，未包含工作区的未暂存 `.gitignore`、未跟踪 `mcp/` 和 SpecFlow 文档，因此所有命令结果均仅归因于暂存内容。

## Requirement alignment

- Version status 仅是标记：已移除 `published` 对更新和删除的门禁，仍保留运行时引用保护。
- 首次部署关联在命令阶段确定：命令先创建或更新 `deploying` Service，再创建引用同一 `service_id` 的 Deployment。
- worker 不创建或补写 Service 关联：缺少 `service_id` 的 Deployment 立即标记失败。

## Spec alignment

- 命令侧 `CommandStore` 取得 `UpsertService`，执行侧 `ExecutionStore` 删除 Service 查找与创建能力，接口边界阻止 worker 回退为创建或推断关联。
- Version 的集成测试覆盖已发布且未被引用的 Version 可更新/删除，以及仍会拒绝存在 Service 引用的删除。

## Plan alignment

计划中的 Go 行为变更均已在暂存范围内；`mcp/`、Python 测试、Docker 标记集成测试未暂存，不属于本次验证范围。

## Actual diff summary

暂存内容包括：

- 放宽 Version 的 `published` 状态门禁，并加入 SQLite 集成测试。
- 将首次部署 Service 准备移入 `DeployApplication` 命令路径，并通过测试验证 Service 在 Deployment 之前持久化。
- 执行端要求持久化的 `service_id`，删除根据 application/environment/instance 推断或创建 Service 的路径。

## Expected vs actual changed files

| 预期范围 | 实际暂存文件 | 结论 |
| --- | --- | --- |
| Version 状态语义和测试 | `internal/application/application/dto/application.go`、`internal/application/application/usecase/version.go`、`internal/application/application/usecase/version_integration_test.go` | 一致 |
| Deployment 命令与执行边界 | `internal/application/deployment/port/port.go`、`internal/application/deployment/usecase/command.go`、`internal/application/deployment/usecase/deployment_execution.go` | 一致 |
| Deployment 行为测试 | `internal/application/deployment/usecase/command_test.go`、`internal/application/deployment/usecase/deployment_execution_test.go` | 一致 |
| MCP 实现、锁文件、Python 测试 | 未暂存 | 本次不验证 |

## Acceptance checklist

- [x] 已发布且未被引用的 Version 可以更新和删除。
- [x] 被 Service 引用的已发布 Version 仍被拒绝删除。
- [x] 首次 deploy 的 Service 在 Deployment 创建前获得生成 ID、目标 Version 和 `deploying` 状态。
- [x] Deployment 持久化同一 Service ID。
- [x] worker 缺少 `service_id` 时标记 Deployment 为失败，而不创建或补写关联。
- [x] 暂存 diff 无空白错误。

## Test results

| 命令 | 结果 |
| --- | --- |
| `git diff --cached --check` | 通过 |
| `gofmt -l <8 个暂存 Go 文件>` | 通过，无输出 |
| `golangci-lint config verify` | 通过 |
| `golangci-lint run ./cmd/... ./internal/... ./sql` | 通过，0 issues |
| `go vet ./cmd/... ./internal/... ./sql` | 通过 |
| `go test -count=1 ./cmd/... ./internal/... ./sql` | 通过 |
| `sqlc generate` 后比较 `internal/gen/sqlc` 哈希 | 通过，无变更 |

## Naming review

- A1 [PASS] 新增领域标识符使用项目既有 `Id` 风格，例如 `versionId`、`deploymentId`。
- A2 [PASS] 暂存新增标识符没有项目内部缩写的全大写形式。
- B1 [PASS] `sql.DB` 使用标准名称 `SQL`。
- B2 [N/A] 暂存代码未引入 `Http`、`Json`、`Uuid` 等错误的协议名称。
- C1 [PASS] 新增标识符与项目既有命名一致。

## Risks

- 历史上缺失 `service_id` 的 Deployment 不会被回填；worker 将明确失败。这符合开发阶段不新增历史数据迁移和不以推断掩盖数据缺陷的决策。
- MCP 的初始实现已暂存；本轮 Makefile、质量工具、格式与类型收窄仍需随最终版本重新审阅和暂存。Docker 标记集成测试仍需显式配置隔离 workspace/project 后执行。

## Incomplete items

- Docker 标记集成测试未运行，因为当前未设置隔离的测试 workspace/project。
- 当前 `mcp/.env` 配置的 Orbit URL 无法连接；此前以临时可用的本机地址完成 nginx 真实只读运行时验证，但不修改用户配置。

## Conclusion

暂存的 Go 变更通过格式、静态分析、测试和 SQLC 幂等性检查，且与已接受的 Version 状态和 Deployment `service_id` 约束一致。MCP 未暂存是本次范围的明确缺口，不影响该 Go 暂存集的结论。

## MCP 追加验证

### 需求与规格对齐

- `POMELO_ORBIT_DATA_ROOT` 变为可选配置；未设置时以 `mcp/` 的父目录解析仓库相对 `data/`。
- `Settings.load` 在返回配置前执行 `mkdir(parents=True, exist_ok=True)`，因此首次启动可使用不存在的数据目录。
- 显式设置仍优先于 `.env`，且进程环境仍优先于 `.env`，未改变既有配置覆盖规则。

### 实际改动

| 文件 | 变更 |
| --- | --- |
| `mcp/src/pomelo_orbit_mcp/settings.py` | 将 data root 从必填项调整为默认仓库 `data/` 并创建目录。 |
| `mcp/tests/test_settings.py` | 覆盖未配置 data root 时的默认路径与目录自动创建。 |
| `mcp/.env.example`、`mcp/README.md` | 说明默认值、创建行为和仅在实际路径不一致时使用覆盖。 |

### 验收清单

- [x] 未设置 `POMELO_ORBIT_DATA_ROOT` 时解析为仓库 `data/`。
- [x] 默认目录不存在时由配置加载创建。
- [x] 显式配置保留进程环境优先于 `.env` 的覆盖规则。
- [x] 依赖锁文件仍可复现。

### 本轮命令结果

| 命令 | 结果 |
| --- | --- |
| `uv --directory mcp lock --check` | 通过 |
| `uv --directory mcp run pytest` | 15 passed，1 deselected（Docker 标记测试） |
| `uv --directory mcp run python -m compileall -q src` | 通过 |

### 风险与未完成项

- `mcp/` 仍未暂存；本轮完成测试后仍需要作为独立提交范围审阅。
- 默认目录以 MCP 包所在仓库为基准，而非进程当前工作目录，避免 MCP client 从任意目录启动时改变 Compose 工作空间定位。

## MCP 静态质量与命令入口

### 实际改动

- `mcp/pyproject.toml` 将 Ruff、mypy 与 PyYAML 类型桩加入开发依赖；Ruff 启用基础语法、导入排序、Python 3.11 升级和常见 bug 规则，mypy 严格检查 `src/`。
- `mcp/Makefile` 统一提供依赖同步、锁定检查、Ruff、mypy、pytest、显式 Docker 集成测试、汇总检查和 stdio 启动入口。
- 外部 Dynaconf、dotenv、Docker、Orbit HTTP 与 Compose/YAML JSON 边界已补充类型收窄，避免以宽泛忽略绕过 mypy。

### 命令结果

| 命令 | 结果 |
| --- | --- |
| `make -C mcp help` | 通过，列出全部目标 |
| `make -C mcp check` | 通过：锁定、Ruff 检查、Ruff 格式、mypy 和 pytest 均成功 |
| `mypy` | 12 个 MCP 源文件，0 issues |
| `pytest` | 15 passed，1 deselected（Docker 标记测试） |

### 仍未完成

- `make -C mcp test-docker` 需要调用方显式提供隔离的 Compose workspace/project；该约束防止测试在未知运行时目标上静默跳过或读取。
