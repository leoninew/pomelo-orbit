# Component Policy Governance
最后修改时间: 2026-08-31 12:13:55

Review status: Accepted

## Background

Version Component 的 `pull_policy` 和 `restart_policy` 当前存在空值或有限枚举，前端、MCP 与运行时行为不一致，导致新建数据和历史 seed 的策略表达不统一。

## Goal

- `pull_policy` 必须有值，且只能是 `missing`、`always`、`never`。
- `restart_policy` 必须有值，且只能是 `no`、`on-failure`、`always`、`unless-stopped`。
- 前端创建 Component 时，默认分别使用 `missing` 和 `unless-stopped`；MCP 创建必须显式提交策略。
- Service Component overlay 继续保留 `nil` 继承 Version 的语义，并支持上述重启策略集合。
- 既有 seed 直接更新为有效值；不新增迁移，不新增数据库约束。

## Non-goal

- 不修改已执行迁移或为历史数据新增迁移修复。
- 不改变 `pull_policy` 的 Compose 拉取语义。
- 不改变 Service overlay 的继承、重置和显式 `no` 语义。

## User scenarios

1. 用户在前端创建或编辑 Version Component，策略字段有默认值并只能选择有效枚举。
2. MCP 调用创建 Version 或 Component 时必须提交策略，应用层校验非空和枚举。
3. MCP 更新 Component basic 信息时必须提交两个策略字段。
4. Service overlay 使用有效重启策略时，部署 Compose 输出对应 `restart`；`no` 不输出该字段。
5. Gateway seed 中的既有 Component 使用 `unless-stopped`，不再写入空值。

## Acceptance

- [ ] 应用层拒绝空值和非法 `pull_policy`。
- [ ] 应用层拒绝空值和非法 `restart_policy`。
- [ ] 前端创建默认值为 `missing` / `unless-stopped`，并提供字段级校验。
- [ ] MCP 创建和更新工具要求显式提交策略，由应用层执行常规校验。
- [ ] Service overlay 支持四种重启策略并保留 `nil` 继承。
- [ ] Compose 对 `on-failure`、`always`、`unless-stopped` 输出 `restart`，对 `no` 省略。
- [ ] 三种数据库的既有 Gateway seed 原位改为 `unless-stopped`。

## Open questions

暂无。

## Decisions

- 策略校验集中在应用层，不增加数据库约束。
- MCP 继续使用已有的 `applicationv1.VersionComponentReq`，mapper 直接传递策略字段，不增加自定义请求类型、默认值或 `UnmarshalJSON` 兼容层。
- 已执行迁移保持不变，seed 文件直接修正。

## Risk

- 既有运行数据库的历史 Service overlay 约束不在本次迁移范围内；本次验证覆盖应用层和 Compose 行为，未执行生产数据库变更。
