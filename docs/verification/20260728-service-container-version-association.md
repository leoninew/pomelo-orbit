# 服务容器版本关联验证记录
最后修改时间: 2026-07-28 16:10:16

Review status: Draft

## Verification scope

- 流程模式：标准模式 / `standard`
- 依据：`docs/requirement/20260728-service-container-version-association.md` 与 `docs/plan/20260728-service-container-version-association.md`，均为 `Accepted`。
- 验证对象：服务的“最近成功版本”移除，以及容器实际运行版本的 Compose label、状态 API 与详情页展示。

## Requirement alignment

1. `service` 的 SQLite/MySQL 建表脚本、查询、模型、仓储、DTO、Proto、MCP 投影与 i18n 均不再包含 `last_successful_version_id` 或相应展示字段。
2. Compose 渲染为每个组件写入 `pomelo.orbit.version-id` 与 `pomelo.orbit.version-label`；渲染测试覆盖两个 label。
3. 容器状态查询先取得 Compose 容器 ID，再以受限 `docker inspect --format` 读取 ID 和 labels；未读取环境变量。解析结果经 API 返回 `version_id` 与 `version_label`。
4. 服务详情容器表展示可跳转的版本；无标签的历史容器显示 `—`，不以 Service 当前 `version_id` 推测。
5. 本地 `orbit` SQLite 已就地核对：迁移版本为 `30`、`dirty=0`、目标列数为 `0`、外键违例数为 `0`。

## Plan alignment

- 迁移、SQL/SQLC、服务和部署 usecase、Proto 生成输入、MCP 投影与服务详情页均已覆盖，符合计划范围。
- 验证中发现两份仍在进行中的 MCP 过程文档描述了已删除的接口字段，已按当前代码回写。
- 暂存区还包含服务运行时配置表格的既有交互细化；它不改变本需求的容器版本语义，已由同一组 Go/前端检查共同覆盖。

## Actual diff summary

- 暂存区包含 62 个文件，757 行新增、593 行删除；其中 SQLC/Proto 生成文件随字段和 API 契约变化更新。
- 服务持久化与接口删除“最近成功版本”字段；版本引用计数只保留 Service 当前版本与 Deployment 版本。
- Compose 生成与运行状态读取建立了容器标签这一唯一的运行时版本关联，前端通过返回的实际标签展示版本链接。

## Expected vs actual changed files

| 类别 | 预期 | 实际 |
| --- | --- | --- |
| 迁移与数据模型 | `sql/migration/`、`sql/query/`、SQLC、Service 模型/仓储 | 已修改 |
| 部署运行态 | Compose renderer、Docker command、status query、DTO/HTTP mapper | 已修改，并有解析与 mapper 单测 |
| API 与消费方 | Proto、Web 生成类型、MCP Service 投影、服务详情 | 已修改 |
| 过程文档 | Requirement、Plan、Verification | 已创建/更新；同时修正两份存活 MCP 过程文档 |

## Acceptance checklist

- [x] `service` 表及其代码路径不再含 `last_successful_version_id`。
- [x] 服务接口和详情页不再暴露或展示“最近成功版本”。
- [x] Compose 服务包含版本 ID 与标签 labels。
- [x] 状态接口从容器标签取得版本 ID/标签并返回前端。
- [x] 容器表显示可访问的版本；无标签时显示空值占位。
- [x] 当前 SQLite 的结构、迁移状态与外键一致性已核验。

## Command results

| 命令或检查 | 结果 |
| --- | --- |
| `task sqlc` | 通过 |
| `task proto` | 通过 |
| `go fmt ./cmd/... ./internal/...` | 通过 |
| `go vet ./cmd/... ./internal/...` | 通过 |
| `go test ./cmd/... ./internal/...` | 通过 |
| `yarn --cwd web lint:fix` | 通过 |
| `yarn --cwd web typecheck` | 通过 |
| `make -C mcp check` | 通过：35 passed，1 deselected |
| `pomelo-db` SQLite 核验 | `version=30`、`dirty=0`、目标列数 0、外键违例 0 |

## Risks and incomplete items

1. 旧容器没有新 labels，需在下一次部署后才显示版本；这是需求中明确接受的限制。
2. 本轮未对真实 Docker 容器执行部署/inspect 集成回归，已由 Compose 渲染、label 解析与 API mapper 单测覆盖主要逻辑。
3. 用户明确要求就地改写已执行迁移，并已同步当前 SQLite；这与仓库默认的迁移约束不同，后续其他环境需按同一策略核验结构后再使用。

## Conclusion

自动化检查、生成结果和本地 SQLite 一致性均通过，未发现本需求的功能性问题。本记录保持 `Draft`，等待人工复核实际 Docker 部署后的容器版本显示。
