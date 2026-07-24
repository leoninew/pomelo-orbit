# backend-go Liquid 模板实现迁移需求
最后修改时间: 2026-06-18 13:29:02

Review status: Accepted

## Background

早前 Python `backend` 的模板实现基于 Jinja 语法与运行时。当前项目正在推进 `backend-go` 迁移，Go 端已有模板渲染能力，但仍保留 Jinja 兼容语义与 `.jinja` 文件命名，例如 `internal/templatex`、CI stage script/artifact 渲染、CD application config/render preview/deploy 路径，以及 migration 初始化数据中的 `.jinja` 模板文件名。

本次需求目标是把 Go 后端相关模板实现替换为 Go 生态的 Liquid 实现。用户最初提到 Shopify Liquid，随后明确纠正 Go 端选型为 `https://github.com/osteele/liquid`。后续规格和实现必须以 `github.com/osteele/liquid` 为准，不使用 Shopify 仓库作为 Go 依赖。

## Goal

- 将 `backend-go` 中模板渲染实现从当前 Jinja 兼容实现迁移为基于 `github.com/osteele/liquid` 的 Liquid 实现。
- 覆盖 Go 后端实际使用模板渲染的路径，包括但不限于：
  - CI pipeline stage script 渲染；
  - CI artifact path/name 渲染；
  - DB migration 初始化数据渲染；
  - CD application config file 渲染；
  - CD compose preview 与实际部署渲染。
- 统一模板语义，避免继续维护 Jinja 兼容转换层或 Jinja 风格 filter 适配。
- 在需求、规格、计划、实现、验证阶段按 strict / 严格模式推进，不跨阶段直接改产品代码。
- 保持现有业务能力和任务执行流程不变，只替换模板语言与 Go 端实现方式。

## Non-goal

- 不修改 Python `backend` 的模板实现。
- 不在本阶段直接实现代码变更；当前只产出 Requirement / 需求草稿。
- 不引入除 `github.com/osteele/liquid` 之外的模板引擎。
- 不继续扩展 Jinja 兼容层、别名层或双模板兼容分支。
- 不主动修改前端功能或前端 API 调用。
- 不主动修改已执行的 migration SQL；如后续确认必须调整初始化模板数据或文件名，需要在 Spec / Plan 阶段明确风险和处理方式。
- 不改变 CI/CD task 类型、payload key、状态流转或 worker 分层结构。
- 不执行任何 git 写操作，例如 commit、push、merge、reset、stash 等。

## User scenarios

- 开发者在 Go 后端新增 CI/CD 模板字段时，只需要理解 Liquid 语法，不需要再理解 Go 端 Jinja 兼容转换规则。
- CI pipeline 执行时，stage script、artifact path 和 artifact name 使用 Liquid 渲染变量。
- CD application preview 与实际部署使用同一套 Liquid 渲染语义，避免预览和部署结果不一致。
- 初始化数据中的模板内容与 Go 运行时模板引擎保持一致，避免“存量数据写 Jinja、运行时跑 Liquid”的语义错位。
- 后续清理 Go 后端模板能力时，可以以 `internal/templatex` 或其替代实现为单一入口定位问题。

## Acceptance

- strict / 严格模式的 Requirement、Spec、Plan、Implementation、Verification 分阶段推进。
- Go 模板依赖选型明确为 `github.com/osteele/liquid`。
- `backend-go` 生产代码不再依赖当前 Jinja 兼容渲染实现或 Jinja 语法转换逻辑。
- CI stage script、CI artifact path/name、CD config render、CD compose preview、CD deploy render、migration 初始化渲染的 Liquid 语义在规格阶段被逐一确认。
- `.jinja` 相关内容采用就地修改，不做历史迁移或兼容分支；具体代码改动在 Spec / Plan 阶段明确。
- 不保留 Jinja 与 Liquid 双轨兼容逻辑，除非用户在后续阶段明确改变范围。
- 相关 Go 测试覆盖 Liquid 变量渲染、缺失变量行为、filter/默认值行为、CD preview 与实际部署一致性。
- 相关 Go 检查通过；至少包含 `go fmt ./...`、`go test ./...`、`go vet ./...`，如命令无法运行或失败，Verification 必须记录原因和输出。
- 不修改无关模块，不修改前端，不修改 Python 后端。

## Open questions

暂无当前必须阻塞后续规格阶段的问题。

## Decisions

- 使用 strict / 严格模式推进。
- Go 端 Liquid 实现选型为 `https://github.com/osteele/liquid`。
- 当前阶段只创建 Requirement / 需求草稿，不进入 Spec / Plan / Implementation。
- 本需求聚焦 `backend-go`，不扩大到 Python backend 或 frontend。
- 不做 Jinja/Liquid 双轨兼容；后续规格应定义直接迁移后的目标状态。
- `.jinja` 相关内容采用就地修改，不做额外迁移流程，不做历史兼容。
- 不迁移历史数据，不为旧模板语法保留兼容逻辑。
- 用户判断当前应该没有缺失变量场景；后续实现仍应避免静默容错，若发现缺失变量应按错误处理。
- 现有 Jinja 风格模板、filter、默认值写法和示例都需要改为 Liquid 方式重新组织。

## Risk

- 模板语言替换可能影响 CI/CD 用户已有模板内容，尤其是 filter、默认值、缺失变量和条件语法。
- `.jinja` 文件名与 Liquid 实现不一致会造成长期认知债；但直接改名可能涉及 migration 初始化数据和已有配置迁移风险。
- 如果 Liquid 缺失变量默认行为较宽松，可能与项目“不要容错隐藏错误”的原则冲突，需要在 Spec 阶段设计严格校验或断言策略。
- CD preview 和实际部署若没有统一渲染入口，迁移后仍可能出现预览与部署不一致。
- `github.com/osteele/liquid` 的 API、严格模式能力、filter 行为需要查证；不能凭经验猜测。
- 当前工作区已有其他 backend-go 架构迁移改动，后续实现必须避免混入无关改动。

## User review notes

- 用户要求使用 strict / 严格模式。
- 用户说明：早前 backend 实现模板基于 Jinja，现在迁移到 Go 实现，需要将相关实现替换为 Go 实现。
- 用户纠正 Go 端选型：不是 Shopify 仓库，Go 实现为 `https://github.com/osteele/liquid`。
- 用户确认：`.jinja` 相关内容就地修改，不做历史迁移和历史兼容。
- 用户确认：当前应该没有缺失变量场景；若后续发现缺失变量，不应静默兼容。
- 用户确认：Jinja 风格写法和示例都需要改为 Liquid 方式重新组织。
