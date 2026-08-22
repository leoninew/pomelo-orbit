# 显式 Compose 挂载源路径验证
最后修改时间: 2026-08-21 07:43:33

Review status: Accepted
Flow mode: light

## Requirement alignment

对照 [需求文档](../requirement/20260820-explicit-relative-mount-source.md) 验证：非 `named_volume` 的 `directory`、`file`、`controlled_file` source 统一要求绝对路径或显式 `./` 相对路径；裸 source 不再进入 Compose，避免被解释为卷挂载。`named_volume` 继续使用裸卷名。

## Actual diff summary

- 模型、Application 校验和 Deployment 校验统一路径规则，并拒绝 drive-relative 的 `C:foo`。
- Compose 解析保留 source 原值；相对 source 才解析到 Service/宿主机目录，绝对 source 直接物化和渲染，不拼接 Service 目录。
- Gateway 初始化和 SQLite/MySQL Gateway 种子的 `traefik.yml`、`acme.json` 均使用 `./` 前缀；没有运行时兼容或自动迁移逻辑。
- MCP tool 描述、输入映射测试、前端 Version/Service 表单和活文档同步更新。
- 新增 `./`、绝对路径、裸路径和绝对受控文件的回归覆盖。
- 当前 SQLite 的 `version_component_mount` 中 15 条裸 source 已通过一次性线下更新改为 `./<原值>`；`service_component_mount` 没有不合规行。

## Expected vs actual changed files

预期范围为挂载模型/校验、部署渲染、初始化数据、MCP、前端、测试和活文档；实际变更均落在这些边界内。SQLite/MySQL 的 Gateway 种子更新为正确初始化值，未新增数据迁移或业务兼容层。

## Acceptance checklist

- [x] 非 `named_volume` source 接受 `/etc/app/app.env`、`C:/etc/app/app.env` 和 `./config/app.env`。
- [x] 非 `named_volume` 的裸 `data`、`config/app.env` 被校验拒绝。
- [x] `controlled_file` 保持文件挂载语义，不接受 `source_is_host_path`，不被当作命名卷。
- [x] 原生 Compose 保留 source 原值；绝对受控文件不拼接 logical Service 目录。
- [x] DooD 只转换相对 source，绝对 source 保持原值。
- [x] MCP 契约和前端校验覆盖相对、绝对及裸路径边界。
- [x] Gateway 初始化和 SQLite/MySQL 种子已更正；当前数据库的 15 条 Version 挂载已线下更新，未增加运行时迁移。

## Test results

- `task --output=interleaved check`：通过，包含 Go format/lint 和前端 typecheck/lint/format。
- `go test ./cmd/... ./internal/...`：通过。
- `yarn --cwd web test`：通过，13 个测试文件、71 个测试用例。
- `git diff HEAD --check`：通过。

## Scope and risks

- 需求在实现过程中根据用户最终澄清收敛为“所有非卷挂载统一规则”，没有保留仅针对 `controlled_file` 的过窄适配。
- 只接受 `/...`、`X:/...`、`./...` 三种路径表示，反斜杠及 UNC 形式会被拒绝；调用方需要改用 Compose 语法。
- 当前数据库和目录盘点已完成；后续环境中的既有裸 source 仍需在部署前离线更新。
- `data/deployment` 中的 `alist-default`、`cli-proxy-api-default`、`easytier-relay-default`、`minio-default` 没有对应的 `service` 记录；未自动删除或移动，需由运维确认归属后另行处理。
- 绝对 `controlled_file` 会按显式 source 直接物化，调用方需确保该路径具备写入权限且符合部署环境约束。

## Conclusion

实现与需求对齐，验证通过，无待完成项。
