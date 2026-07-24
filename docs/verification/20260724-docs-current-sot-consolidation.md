# 文档整理：归档历史并沉淀当前 SoT — 验证记录
最后修改时间: 2026-07-24 11:17:30

Flow mode: light

## Requirement alignment / 需求对齐

| 验收项 | 结论 |
|--------|------|
| 活文档结构 product / architecture / decisions | **符合** — 已建且含 CD 模型、后端架构、运行时、决策账本 |
| README + **独立** INDEX | **符合** |
| archive README 声明无须采信 | **符合** — `docs/archive/README.md` + `archive/specflow/README.md` |
| HEAD 已提交过程库物理归档 | **符合** — requirement 62 / spec 14 / plan 19 / verification 50 → `archive/specflow/` |
| 已提交 analyze 消化后归档 | **符合** — usability 要点进 product；原文 archive/analyze |
| 文码冲突以代码为准 | **符合** — 活文档声明并以 `model/cd.go` 等为锚点 |
| 未提交文档不处理 | **符合** — 并行 20260724 过程/analyze 仍 untracked，未进本提交暂存 |
| 文档-only | **符合** — 暂存无 `sql/` / `internal/` 产品代码 |

## Spec alignment / 规格对齐

不适用（light，无独立 spec）。

## Plan alignment / 计划对齐

不适用（light，无独立 plan）。按 requirement 待处理事项 A–E 已关闭。

## Actual diff summary / 实际差异摘要

1. 新增活 SoT：`docs/product/*`、`docs/architecture/*`、`docs/decisions/ledger.md`。  
2. 新增/重写入口：`docs/INDEX.md`、`docs/README.md`、`docs/archive/README.md`。  
3. 过程库与过时 designs/guides/analyze **rename** 入 `docs/archive/...`。  
4. guides/frontend 校准（deployment 重写、living banner、`web/` 路径）。  
5. `CLAUDE.md` 文档阅读顺序。  
6. 本任务 requirement：`docs/requirement/20260724-docs-current-sot-consolidation.md`。

## Expected vs actual changed files / 预期与实际

| 预期 | 实际 |
|------|------|
| 活 SoT + INDEX + archive 声明 | 有 |
| HEAD 过程四件套归档 | 有（R 检测为 rename） |
| designs / 状态机 / usability 退场 | 有 |
| 不含未提交并行任务文 | 有（仍 ??） |
| 不含 migration 代码 | 有（未暂存） |

## Acceptance criteria checklist / 验收清单

- [x] A1–A5 结构与入口  
- [x] B1–B6 活 SoT 与校准  
- [x] C1–C4 搬移与对账  
- [x] D1 commit 边界（与代码拆分暂存）  
- [x] E1 CLAUDE.md  

## Test / command results / 测试 / 命令结果

| 检查 | 结果 |
|------|------|
| 活 SoT 路径存在 | 8/8 OK |
| archive/specflow 计数 | 62+14+19+50 |
| `git diff --cached` 范围 | 仅 `docs/**` + `CLAUDE.md`（约 179 路径） |
| 暂存中无 sql/internal/并行任务 | 无匹配 |
| go/yarn | **未跑**（文档-only，无产品代码变更） |

## Missed or expanded scope / 范围偏差

- guides 中 CI/权限等仅加 living 声明，未做与代码逐段深校准（requirement 已允许运维笔记薄校准）。  
- archive 内历史互链未全量修复（按需求允许）。  

## Risks / 风险

- 读者若仍书签旧过程路径，需改走 INDEX / archive。  
- 并行未提交任务文档仍在工作区，提交本 diff 时勿 `git add docs/` 整树。  

## Incomplete items / 未完成事项

无阻塞项。可选后续：CI/Auth 产品活文档骨架、guides 深校准。

## Conclusion / 结论

**通过。** 满足 light 需求验收；可单独提交文档整理变更。
