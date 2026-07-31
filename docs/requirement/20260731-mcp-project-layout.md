# MCP 项目与 RAGFlow 部署方式重组
最后修改时间: 2026-07-31 16:32:13

Review status: Accepted

## Background

原有本地 stdio MCP 以 `pomelo-orbit-mcp` 和 `pomelo_orbit` 暴露，但职责是 Application、Version、Service、Gateway 的交付生命周期及运行时验收，不是持续集成。未来还需要独立的持续集成/流水线 MCP，不能继续让两个职责共享一个含义不清的项目根目录。

RAGFlow 同时需要集成式和拆分式部署：集成式将 RAGFlow、TEI 与 MySQL、Valkey、MinIO、Elasticsearch 置于一个 Application；拆分式让 RAGFlow+TEI 使用独立 Application，并搭配四个独立 backing Application。两种方式不能共享部署目录或运行时密钥边界，但需要在任一技能被调用时初始化同一份完整可选库存。

## Goal

1. 将当前交付控制面命名为 `pomelo-delivery-mcp`，Codex 注册名为 `pomelo_delivery`。
2. 将 MCP 工程统一组织为 `mcp/src/delivery-mcp/` 与 `mcp/src/pipeline-mcp/` 两个并列项目。
3. 交付 MCP 保留 Pomelo Orbit 控制面适配语义和 `orbit_*` 工具名，但 Python 包名、命令、配置和文档使用 delivery 命名。
4. 预留独立的 `pomelo-pipeline-mcp` 空骨架，供后续实现持续集成/流水线能力。
5. 将 RAGFlow 部署技能分为集成式和拆分式两个明确入口；两者均初始化完整库存，但只部署自己拥有的运行拓扑。
6. 对集成式与拆分式 RAGFlow 保持独立 Application、服务、模型缓存和运行时变量边界，并根据 NVIDIA 探测和预检选择 CPU 或 GPU Version。

## Non-goal

1. 不实现 pipeline MCP Server、工具、依赖或 Codex 注册。
2. 不改变 Orbit HTTP API、`orbit_*` 工具契约、RAGFlow 部署拓扑或运行时生命周期行为。
3. 不保留旧 `pomelo-orbit-mcp` 命令、`pomelo_orbit` 注册名、`pomelo_orbit_mcp` Python 包或旧目录入口的兼容层。
4. 不启动开发服务器或执行部署生命周期写操作。
5. 不将集成式 RAGFlow 数据迁移、挂载或复用到拆分式拓扑，反之亦然。

## User scenarios

1. Codex 在新会话通过 `pomelo_delivery` 发现交付控制工具，并从 `mcp/src/delivery-mcp` 启动本地 stdio Server。
2. 维护者在 `mcp/src/` 中能直接区分已实现的交付 MCP 与尚未实现的 pipeline MCP。
3. RAGFlow 运维技能和操作文档引用新的 MCP 注册名、启动命令和工具前缀，不会误用旧名称。
4. 用户请求默认 RAGFlow 部署时使用集成式 skill；明确请求拆分部署时使用拆分式 skill，二者先初始化共同库存再只部署所选拓扑。

## Acceptance criteria

1. `mcp/src/` 仅包含并列的 `delivery-mcp` 与 `pipeline-mcp` MCP 项目目录。
2. Delivery 项目提供 `pomelo-delivery-mcp` 命令，并以 `pomelo_delivery_mcp` 包实现；`.codex/config.toml` 注册 `pomelo_delivery`。
3. Delivery 项目的 `.env`、JWT 缓存、数据根解析、Makefile、锁文件、测试和 README 均在新项目目录下有效；默认数据根仍为仓库 `data/`。
4. Pipeline 项目可作为独立 Python 包构建，但没有 MCP 可执行入口、实现或 Codex 注册。
5. AGENTS、MCP 指南、RAGFlow 运维手册和技能元数据明确新的命令与注册名；工具仍以 `mcp__pomelo_delivery__orbit_*` 出现。
6. Delivery 静态检查、类型检查和默认测试通过；Pipeline 锁文件与 wheel 构建通过。
7. `deploy-ragflow-integrated-orbit` 维护 `ragflow` 的 CPU/GPU 六组件 Versions；`deploy-ragflow-split-orbit` 维护 `ragflow-split` 的 CPU/GPU 两组件 Versions 及四个独立 backing Applications。
8. 两个 RAGFlow skill 都要求初始化上述完整库存、在每次 lifecycle write 前读取 `pomelo_delivery` schema，并只启动自己拥有的 RAGFlow 服务。
9. 两个 RAGFlow Services 均使用 local `9380`，切换时通过 Orbit 停止另一服务且不删除 volumes；模型缓存和运行时变量不跨 Application 复用。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. `pomelo_delivery_mcp` 是交付 MCP 的 Python 包名；`orbit_*` 保留为外部控制面适配工具名。
2. `mcp/src/delivery-mcp` 与 `mcp/src/pipeline-mcp` 是各自独立的 Python/uv 项目根，不在顶层 `mcp/` 提供代理入口。
3. 将已有本地 MCP `.env` 与 JWT 缓存直接移动到 delivery 项目目录，不读取、不输出或转换其中值。
4. 集成式和拆分式 RAGFlow 使用不同 Application 与 managed deployment directory；两个 skill 共享初始化库存的定义，但不共享部署责任。
5. NVIDIA 可用且 GPU 预检通过时才选择 GPU Version；检测到 GPU 但预检失败时阻断部署，不静默回退 CPU。

## Risks and assumptions

1. 已打开的 Codex stdio MCP session 不会热加载注册和源码路径，必须重新建立会话后才能使用新工具前缀。
2. Pipeline 目录是有意不可运行的预留工程；在实现实际流水线能力前不能将其注册为 MCP。
3. RAGFlow 的模型缓存和既有非空 backing store 需要保留授权的匹配运行时值；技能不得猜测、导出、旋转或覆盖这些值。
4. GPU Version 的实际推理验收仍依赖目标 Host 的 NVIDIA driver、Container Toolkit 与模型/镜像预检条件。
