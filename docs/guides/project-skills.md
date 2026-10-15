# 项目内 Agent Skills

工程实践 skill 统一维护在仓库的 `.agents/skills/`，随源码版本管理，只供当前项目使用。

| Skill | 使用场景 |
| --- | --- |
| [backend-architecture-governance](../../.agents/skills/backend-architecture-governance/SKILL.md) | Go 后端架构复查、测试依赖治理与跨层编排 |
| [frontend-unit-testing](../../.agents/skills/frontend-unit-testing/SKILL.md) | 前端轻量单元测试的编写、审查、警告排查与漂移清理 |

## 使用

Codex 从仓库的 `.agents/skills/` 自动发现项目 skill，路径约定见 [OpenAI 官方文档](https://developers.openai.com/codex/skills/)。[AGENTS.md](../../AGENTS.md) 也直接引用这两个 skill；执行对应任务时读取相关 `SKILL.md` 及按需引用的材料。

## 维护

- 每个 skill 只保留一份源码，目录名与 frontmatter 的 `name` 一致。
- 项目规则以 `AGENTS.md` 和活文档为依据；skill 聚焦对应实践，通过相对链接引用项目代码与文档。
- 仅调整 skill 或文档时，核验 frontmatter、相对链接和变更内容；涉及实现时执行项目要求的检查。
