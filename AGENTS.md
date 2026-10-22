# 开发约定

- 先读相关代码和测试，沿用项目职责边界；文档与代码冲突时以代码为准。
- 直接实现当前设计；不添加兼容层、别名、默认值兜底或新旧逻辑并存。
- 未经用户要求，不启动、停止或重启开发服务器，不修改已执行的迁移。
- API 路由使用单数形式，如 `/api/ci/repository`。
- 前端位于 `web/`，使用共享组件和现有样式；不散落底层 Reka primitives。时间展示使用 `web/src/utils/time.ts`。
- 后端遵循既有事务、仓储和日志边界。架构与测试依赖治理见 [后端技能](.agents/skills/backend-architecture-governance/SKILL.md)。
- 测试覆盖当前业务的主要流程和真实失败边界，不穷举理论情况，不保留已移除行为的断言。
- 前端只保留轻量单测，不做并发、性能、集成、端到端或整页串联测试；见 [前端测试技能](.agents/skills/frontend-unit-testing/SKILL.md)。后端持久化、事务及协议测试按实际边界分类。

## 验证

- 前端：在 `web/` 运行 `corepack yarn lint:fix`、`corepack yarn typecheck`，按变更补充单测。
- Go：运行 `task check` 和 `go test ./cmd/... ./internal/...`。
- Go 核心单测：`task test:go:unit -- -count=1`；完整测试：`task test:go:all -- -count=1`。
- 使用项目现有工具；计时使用 `-count=1`，不依据缓存结果判断性能。

## 文档

- 入口为 [README](README.md) 和 [业务模型与文档](docs/README.md)。文档只记录必要约定与操作，不复述代码和工作过程。
- 进行中的 SpecFlow 文档位于 `docs/intent|spec|plan|verification`；完成后归档到 `docs/archive/specflow/`。
- `docs/archive/` 仅供历史追溯，不作为当前实现或验收依据。
- 项目技能维护于 `.agents/skills/`。
