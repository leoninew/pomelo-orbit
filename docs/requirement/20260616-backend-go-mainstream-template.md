# backend-go 主流模板架构改进需求

Review status: Accepted

## Background

`backend-go` 由现有 Python `backend` 迁移而来。当前 Go 后端实现更像是为了功能对等而手搓出来的一套结构：以 `chi + sqlx` 为基础，但大量逻辑集中在 `internal/httpserver`、`internal/orbit.Store`、`internal/orbit/model.go` 和 `internal/app/app.go` 中。

本次需求的核心不是补某个单独 API，也不是修改 Python 后端或模板来源，而是把 `backend-go` 从“功能对等移植版”调整为“符合主流 Go 后端模板/分层习惯的实现”。用户明确指出当前应先搞清楚问题和意图，避免直接展开成大而全的架构圣经。

本次不涉及 migration SQL 修改，也不涉及 `jingjia` 模板相关修改。后续分析和实现应聚焦 `backend-go` 自身的代码组织、职责边界和主流 Go 项目结构。

本次还需要引入 `https://github.com/air-verse/air`，用于开发阶段 hot reload，并且对 API 进程和 worker 进程都生效。

## Goal

- 梳理并改进 `backend-go` 当前手搓架构，使其符合主流 Go 后端项目结构。
- 保持现有业务能力和 API 行为基本不变，优先改变代码组织、职责边界和依赖方向。
- 将当前混杂在 `httpserver` 和 `orbit.Store` 中的职责拆分到更清晰的层次中，例如：
  - router / handler：HTTP 路由、请求解析、响应输出；
  - service / usecase：业务用例编排；
  - repository / dao：数据访问；
  - model / entity：持久化模型和业务实体；
  - config / bootstrap：配置加载和依赖组装；
  - middleware / response / error：HTTP 横切能力。
- 消除 `internal/orbit` 作为“全能 Store + DB model + 部分业务语义”的中心化设计。
- 拆分过大的 HTTP 文件和路由注册逻辑，特别是 CI/CD 相关的大文件和混合职责。
- 不围绕 Python 模板本身做修改；只把 Go 端落到更主流、可维护的结构上。
- 引入 `air-verse/air` hot reload 开发配置，覆盖 API 和 worker 两种开发进程。
- 实施阶段必须维护 checklist：列出全部改造项，做一项勾选一项，直到 checklist 全部完成。

## Non-goal

- 不重写 Python `backend`。
- 不修改、迁移或清理 `jingjia` 模板相关内容。
- 不修改 migration SQL，不调整 migration 文件组织，不处理 migration checksum 或数据库重建问题。
- 不改变前端功能或接口调用方式，除非后续发现 Go 后端 API 合约本身需要修正并获得确认。
- 不为了“架构升级”主动添加新业务功能。
- 不追求一次性引入复杂框架、ORM、代码生成、事件总线、CQRS 等额外能力。
- 不把 hot reload 做成生产运行机制；`air` 仅用于开发阶段。
- 不执行 git 写操作，例如 commit、push、merge、reset、stash 等。

## User scenarios

- 开发者阅读 `backend-go` 时，可以按主流 Go 后端分层快速定位：HTTP handler、业务 service、数据 repository、模型、配置和启动组装。
- 新增或修改一个 CI/CD/API 功能时，不需要在 `httpserver` 大文件和 `orbit.Store` 中交叉堆逻辑。
- 后续继续迁移或完善 Go 后端功能时，Go 端有稳定模板可循，不再继续扩大手搓结构。
- 测试和验证可以围绕分层后的 handler/service/repository 分别开展，而不是只能通过端到端 HTTP 路由覆盖所有逻辑。
- 开发者修改 Go 代码后，可以用 `air` 自动重编译并重启 API 进程。
- 开发者修改 Go 代码后，可以用 `air` 自动重编译并重启 worker 进程。
- 实施过程中，用户可以通过 checklist 看到哪些模块已改、哪些模块未改，避免“大重构做到一半不可见”。

## Acceptance

- 需求、规格、计划、实现和验证按 strict / 严格模式推进，不跨阶段直接实现。
- 规格阶段必须先明确“主流模板架构”的落地形态，避免抽象空谈或过度设计。
- 实现后，`backend-go` 至少应体现清晰的 handler/service/repository/bootstrap 分层，核心业务不再主要堆在 `internal/httpserver` 和 `internal/orbit.Store`。
- `internal/orbit` 的职责需要被拆解或替换；如果因阶段切分暂时保留，必须在 checklist 中体现退出路径，而不是长期兼容层。
- CI/CD、项目、认证/用户/角色、后台任务等现有能力在重构后保持 API 路径、请求响应结构和主要错误语义稳定。
- 提供 `air-verse/air` 开发配置，API 和 worker 均可通过 hot reload 启动。
- hot reload 配置不影响生产启动方式，不替代现有正常构建/运行命令。
- Plan / 计划阶段必须产出完整 checklist；Implementation / 实现阶段按 checklist 逐项推进，完成一项勾选一项，直至全部完成。
- 本次改造不包含 migration SQL 变更；如实现过程中发现必须改 schema，应停止并回到需求/规格阶段确认范围变更。
- 相关 Go 测试、lint 或项目约定检查应通过；如无法运行或失败，验证文档必须记录原因、命令和输出。
- 不修改无关模块，不做兼容层，不添加未明确需要的功能。

## Open questions

暂无当前必须阻塞后续规格阶段的问题。

## Decisions

- 使用 strict / 严格模式推进。
- 本需求以 `backend-go` 为目标，不扩大到 Python 后端、模板来源、migration 或前端。
- 不引入 go-zero、Kratos、GoFrame 等大框架。
- 保留现有轻量技术栈：`chi + sqlx + viper + slog`。
- 采用主流 Go 分层：handler / service / repository / model / bootstrap / middleware / response。
- 引入 `air-verse/air` 作为开发阶段 hot reload 工具，覆盖 API 和 worker。
- 不采用“只先做一个 project 纵向切片后暂停”的策略；计划阶段应列出完整 checklist，实施阶段做一项勾选一项，直到全部改完。
- 迁移过程中逐步拆除 `internal/orbit`，不把它作为长期兼容层保留。
- 本次不修改 migration，不处理 `jingjia` 模板。
- `go.mod` module path 暂不作为第一阶段内容；如后续计划认为需要，可作为独立 checklist 项处理。
- 用户已明确纠偏：当前首先要搞清楚任务含义，避免直接展开成大而全的实施长文。

## Risk

- 架构改造范围较大，若 checklist 粒度过粗，容易变成不可审查的大重构。
- 保留现有 `chi/sqlx` 时，需要确保“主流模板架构”不是只换目录名，而是真正改变职责边界。
- 大量文件移动和 import 更新可能造成 review 噪音，需要在计划阶段定义可审查的 checklist 粒度。
- `air` 会新增开发工具配置，需要避免生成的临时二进制、tmp 目录或日志污染仓库。
- API 和 worker 如果共用同一个 air 输出二进制，可能互相覆盖；计划阶段需要明确隔离配置。
- 当前工作区已有用户改动，实施阶段必须避免覆盖无关变更。

## User review notes

- 用户澄清：本次不涉及 migrations 和 `jingjia` 模板相关修改。
- 用户采纳建议：不引入大框架，保留当前轻量技术栈，采用主流 Go 分层，逐步拆除 `internal/orbit`，本次不处理 migration / jingjia，`go.mod` 暂不作为第一阶段重点。
- 用户调整建议第 4 项：不是只先做一个纵向切片，而是列出完整 checklist，做一项勾选一项，直到全部改完。
- 用户新增范围：引入 `https://github.com/air-verse/air` hot reload，开发阶段对 API 和 worker 都生效。
