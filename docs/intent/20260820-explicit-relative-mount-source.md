# 显式 Compose 挂载源路径
最后修改时间: 2026-08-21 07:38:10

Review status: Accepted
Flow mode: light

## Background

Compose 的短语法会把没有路径标记的 `config/app.env`、`data` 等 source 解释为卷名。当前挂载模型同时支持目录、普通文件、受控文件和命名卷，source 必须明确表达它是路径还是卷；渲染层也不能通过隐式补 `./` 改写业务值。

## Goal

统一非 `named_volume` 挂载的 source 契约：source 必须是绝对路径（例如 `/etc/app/app.env` 或 `C:/etc/app/app.env`），或显式以 `./` 开头的相对路径（例如 `./config/app.env`）。裸的 `config/app.env`、`data` 等形式禁止用于 `directory`、`file` 和 `controlled_file`。业务代码直接使用保存的 source，不提供旧数据兼容或默认补全；已有业务数据由线下更新。

## Non-goal

- 不在运行时兼容或自动改写裸相对 source。
- 不新增数据库迁移来自动改写已有业务数据。
- 不改变 `named_volume` 的裸卷名语义。
- 不改变受控文件的内容、mode、物化或 `ignore_if_exists` 能力。

## User scenarios

1. 用户创建或编辑 `directory`、`file`、`controlled_file` 挂载时，使用 `./config/app.env` 或绝对路径；该值被原样保存并用于 Compose。
2. 输入 `config/app.env` 或 `data` 时，HTTP、部署校验、MCP 和前端均拒绝该值，避免它被 Compose 当作卷挂载。
3. MCP `orbit_update_version_component_mounts` 的描述明确区分路径和卷：非 `named_volume` 使用绝对路径或 `./...`，`named_volume` 使用裸卷名，`controlled_file` 不被当作卷。
4. 原生 Compose 渲染保留 source 原值；DooD 仅把相对 source 解析到宿主机工作区，绝对 source 不拼接 Service 目录。
5. Gateway 初始化及其 SQLite/MySQL 种子创建的受控 `traefik.yml`、`acme.json` 使用 `./` 前缀；现有数据库中的业务数据由运维线下更新。

## Acceptance

- 应用层、部署层和模型校验对所有非 `named_volume` source 只接受绝对路径或以 `./` 开头的相对路径；裸相对 source 被拒绝。
- `named_volume` 继续只接受裸卷名，`controlled_file` 只能作为文件挂载，不得使用 `source_is_host_path` 或变成命名卷。
- 原生 Compose 直接使用保存的 source，不存在自动补 `./` 的适配函数。
- 相对逻辑目录和受控文件在 Service 目录物化；绝对受控文件直接在其绝对 source 物化，并由 Compose 使用该绝对 source。
- DooD 只将相对 source 解析到 Compose host workspace；绝对 source 原样保留。
- Gateway 初始化数据、前端校验、MCP tool description、MCP 输入映射测试和活文档采用同一 source 契约。
- 不新增迁移或运行时数据改写；Gateway 种子作为新库初始化数据保持正确，既有数据库通过一次性线下操作处理。
- `task check`、`go test ./cmd/... ./internal/...`、前端 typecheck/lint/test 通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 非 `named_volume` 的挂载统一使用“绝对路径或 `./` 相对路径”规则；不为 `controlled_file` 另造特殊路径格式。
- 业务代码不提供旧格式适配；旧数据由线下 SQL 或运维操作处理。
- source 是什么就渲染成什么；只在 DooD 的相对路径执行环境转换，绝对路径不拼接平台目录。
- MCP 是对外契约的一部分，必须与 HTTP、前端和部署层同步修改。

## Risk

- 未线下更新的旧 Version 或 Service Component mount source 会在新的写入校验或部署校验处失败；上线前需要完成数据盘点和更新。
- 已执行数据库不会因 Gateway 种子内容改变而自动修正；仍需线下校正其已有挂载值。
