# MCP 项目与 RAGFlow 部署方式重组验证记录
最后修改时间: 2026-07-31 16:32:13

Review status: Draft

## 需求对齐

- Delivery MCP 已与 Pomelo Orbit 控制面产品名区分：分发和命令使用 `pomelo-delivery-mcp`，Codex Server 使用 `pomelo_delivery`，工具仍保留 `orbit_*`。
- `mcp/src/` 已按职责形成两个独立项目：已实现的 `delivery-mcp` 与预留的 `pipeline-mcp`。
- RAGFlow 部署已按集成式和拆分式路径分为两个 skill；二者初始化同一份完整库存，但只部署各自的 Application 和 Service。
- 本记录只覆盖 MCP 与 RAGFlow 部署方式重组，不包含其他工作区变更。

## 实际改动

| 预期区域 | 实际结果 |
| --- | --- |
| `mcp/src/delivery-mcp/` | 承载 pyproject、lock、源码、测试、Makefile、README、`.env.example` 和本地配置忽略规则；Python 包为 `pomelo_delivery_mcp`。 |
| `mcp/src/pipeline-mcp/` | 承载独立 `pomelo-pipeline-mcp` 空包与 lock；没有 scripts、MCP Server 或 Codex 注册。 |
| `.codex/config.toml` | 以 `uv --directory mcp/src/delivery-mcp run pomelo-delivery-mcp` 注册 `pomelo_delivery`。 |
| AGENTS、指南与技能 | 明确新注册名、命令与 `mcp__pomelo_delivery__orbit_*` 工具前缀；保留 `orbit_*` 的控制面适配语义。 |
| 本地交付配置 | `.env` 与 JWT 缓存移动到 delivery 项目目录，未读取或输出内容；默认数据根仍解析为仓库 `data/`。 |
| RAGFlow 部署方式 | 新增集成式和拆分式 skill 及共享库存：集成式拥有一个六组件 Application，拆分式拥有一个两组件 RAGFlow Application 与四个独立 backing Applications。 |

## 验收清单

- [x] `mcp/src/` 只有 `delivery-mcp` 和 `pipeline-mcp` 两个 MCP 项目目录。
- [x] 交付命令、Codex 注册、Python 包和源码目录均使用 delivery 命名。
- [x] 旧根目录命令、旧注册名和 `pomelo_orbit_mcp` 导入未出现在活动 MCP 实现中。
- [x] Pipeline 空包未注册、无执行入口，并可构建。
- [x] 活动 MCP 文档与 RAGFlow skill 反映新命令和工具前缀。
- [x] 两个 RAGFlow skill 共同初始化 CPU/GPU Versions 和四个 backing Applications，但分别部署集成式或拆分式运行路径。
- [x] RAGFlow Application、模型缓存、运行时变量及 local `9380` 切换规则具有明确隔离约束。

## 命令结果

- `make -C mcp/src/delivery-mcp check`：通过，Ruff、格式检查、mypy 通过；pytest 为 68 passed、1 deselected。
- `uv lock --check`（`mcp/src/pipeline-mcp`）：通过。
- `uv build --wheel`（`mcp/src/pipeline-mcp`）：通过。
- 仅加载 Delivery Settings 的配置位置断言：通过，确认默认数据根为仓库 `data/`；未启动 Server 或发起控制面调用。
- 两个 RAGFlow skill 的 `quick_validate.py`：通过。
- `git diff --check` 与 `git diff --cached --check`：通过。

## 范围偏差

无。此前的根目录 pipeline 骨架已随最终结构移动到 `mcp/src/pipeline-mcp/`；RAGFlow 由单一不清晰入口重组为集成式和拆分式两个明确入口，均符合本需求边界。

## 风险与未完成项

- Pipeline MCP 仍是刻意空置的工程，后续实现前不具备可调用能力。
- 需要在新的 Codex 会话中重新发现 `pomelo_delivery`；已打开的 stdio MCP session 不会热加载本次注册变更。
- GPU 路径只在具备 NVIDIA 条件且完成预检的 Host 上部署；无 GPU 或预检失败时按技能规则选择或阻断。
- 本次记录和技能重组不替代目标 Host 的实际 RAGFlow 健康、模型推理和数据完整性验收。

## 结论

本次交付满足已接受的 MCP 命名、目录边界与 RAGFlow 部署方式重组需求。Delivery MCP 在新项目目录通过完整检查，Pipeline MCP 保持可构建但未注册的预留状态；两个 RAGFlow skill 已形成可区分、可初始化且数据隔离的部署约束。
