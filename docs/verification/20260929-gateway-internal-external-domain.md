# Gateway 内外网域名与自定义路由域名辅助验证
最后修改时间: 2026-09-29 10:23:16

Review status: Draft

Flow mode: standard

## Intent alignment

- GatewayConfig 使用 `internal_domain` 和可选 `external_domain`；组件派生 Host、Docker labels 与 Gateway 自身地址仍取内网域名。外网域名仅供自定义 Route 表单拼接，未增加 DNS 管理或 Gateway 对外路由。
- Route 创建、列表编辑和详情编辑在编码变化时使用 `编码.external_domain` 填写现有域名控件；未配置外网域名时创建控件留空。已保存或手填且不符合当前生成值的域名不会被覆盖，Route 仍保存控件中的完整域名。
- Route 编码前后端均限制为小写字母开头的 DNS 单标签，最长 32 字符；外网域名允许多级标签、末级非纯数字，最长 220 字符。
- 没有为旧配置键或存量非法 Route 编码加入运行时兼容。部署配置与存量数据更新按已接受意图由上线流程处理。

## Plan alignment

| 计划项 | 核对结果 |
| --- | --- |
| 三种数据库迁移、Gateway 仓储与 sqlc | 已实现；SQLite 保值实测通过，PostgreSQL 实库已处于干净的版本 49 且可接收 SQLite 导出，MySQL 脚本静态核对；PostgreSQL 的 48→49 升级过程未单独见证 |
| Gateway 创建、更新、初始化、导入导出及 HTTP/Proto/MCP 契约 | 已贯通新字段；MCP 更新可设置两个域名并清空外网域名 |
| 内网域名运行时行为与 Gateway 表单布局 | 派生地址和 label 使用 `internal_domain`；初始化与编辑表单的两个域名各占整行 |
| Route 三处表单联动与编码校验 | 创建页、列表编辑和详情编辑均调用同一联动函数；服务端 HTTP/TCP Route 校验使用同一编码约束 |
| 活文档、生成代码及检查 | 已更新；规定的检查和项目完整测试入口通过 |

## Actual diff summary

- 新增三种数据库的 `000049_gateway_domains` up/down 迁移；Gateway schema、查询、仓储、模型与生成的 SQLC 代码统一为两个域名字段。
- Gateway 初始化默认值、创建/更新、定义导入导出、HTTP Proto、MCP、配置文件及前端表单均传递新字段；Gateway 展示和双域名单行布局已更新。
- Route 配置响应暴露外网域名；三处表单依编码变化生成或保留完整域名。Route 编码校验统一为 DNS 单标签规则，更新中英文提示。
- Docker label 与 Gateway 派生地址改读 `internal_domain`；针对存量迁移、Gateway/MCP 传参、域名规则和联动补充了测试。相关活文档已同步。

## Expected vs actual changed files

| Plan 预计范围 | 实际文件与差异 |
| --- | --- |
| `sql/migration/{sqlite,mysql,postgres}`、Gateway 查询与仓储 | 三方言 `000049`、`sql/query/gateway/gateway.sql`、schema、仓储与生成 SQLC 已变更；共享 `gateway_config` 模型使多个 sqlc 包的生成模型同步变化 |
| Gateway/Route/Initialization/Deployment/Config、HTTP/MCP/Proto | 对应 `internal/model`、`internal/application`、`internal/api`、`internal/config`、`proto` 及生成 DTO 已变更 |
| Gateway、项目初始化和 Route 前端表单 | `web/src/views/gateway`、`project`、`route`、i18n 与生成 DTO 已变更；新增共用 `routeDomain.ts` 及测试 |
| 配置、活文档和相关测试 | `configs/config.yaml`、产品/架构/指南/决策文档及测试已变更；新增 Intent、Plan 与本 Verification 文档 |

实际改动与计划范围一致。完整测试首次运行暴露 `routeSync.test.ts` 缺少新增 Gateway 配置请求的 mock，已补齐并重新通过；该修正属于本功能影响的既有测试。

## Acceptance checklist

- [x] Gateway 初始化、编辑、HTTP/Proto、MCP、配置与持久化统一使用 `internal_domain`、`external_domain`，内网域名仍必填；MCP 可清空外网域名。
- [x] Gateway 初始化和编辑表单中两个域名字段各占一行。
- [x] SQLite 迁移保留旧内网值，并为已有 Gateway 建立空外网域名；三方言迁移均未修改旧迁移文件。
- [x] 业务代码和当前配置不再读取或写入 `base_domain`；旧字段仅用于迁移脚本、过程文档中的历史说明，以及刻意构造旧 schema 的迁移测试。
- [x] `gateway` mode 派生 Host 和 Docker labels 仍使用原值对应的 `internal_domain`；无外网域名时 Route 创建域名留空。
- [x] Route 创建、列表编辑、详情编辑均在编码变化时生成完整域名；编辑打开时保留已保存值，手填其他域名后不覆盖。
- [x] Route 编码在前后端限制为小写字母开头、最多 32 字符的 DNS 单标签；外网后缀允许多级子域名，末级非纯数字，最多 220 字符。
- [x] 修改外网域名不会批量改写已有 Route，也不会直接触发 Traefik 发布；Route 同步流程未改。
- [ ] MySQL 的 `000049` 迁移及 PostgreSQL 的 48→49 升级过程尚未实库见证；上线前需在目标方言验证。

## Test results

| 命令或核对 | 结果 |
| --- | --- |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 通过；修正测试 mock 后复查通过 |
| `task check` | 通过；TypeScript、ESLint、Prettier、Go format 与 golangci-lint 均通过，0 issues |
| `task test` | 通过；Vitest 34 文件、147 用例；Go `cmd`、`internal`、`sql` 包通过 |
| `go test -count=1 ./internal/infrastructure/database -run '^TestMigrateGatewayDomainsSQLite$'` | 通过；从旧 schema 升级后内网域名保值且外网域名为空 |
| PostgreSQL 实库只读核对及 `copy-from-sqlite --no-dry-run` | PostgreSQL 与 SQLite 均为干净的版本 49；47 表、2894 行导入完成。最终行数校验失败：`mcp_access_token` 在源库为 1 行、目标库为 2 行；旧 token 已从 SQLite 删除，但 `upsert` 不删除目标独有行。此差异与 Gateway 域名迁移无关 |
| `git diff --cached --check` | 通过；暂存改动无空白错误 |
| MySQL/PostgreSQL 迁移 | up/down 脚本与 schema 静态核对；已连接 PostgreSQL 确认当前版本 49，但未单独见证从版本 48 升级 |

未启动开发服务器，也未进行浏览器端人工交互验证。三处表单的调用路径已静态核对，联动函数由 Vitest 覆盖。

## Missed or expanded scope

没有发现超出已接受意图的产品行为。生成 SQLC 模型涉及多个包，是共享 Gateway 表结构的生成结果；`routeSync.test.ts` 的 mock 修正是新增配置读取的必要测试调整。

## Risks and incomplete items

- 部署配置必须与新版本同步改用 `project_initialization.gateway.internal_domain` 和 `external_domain`；业务代码没有旧键兼容。
- 存量不符合新规则的 Route 编码须由部署方上线前离线 `UPDATE`，本次未提供运行时修复或处理非法存量数据。
- MySQL 的 `000049` 迁移及 PostgreSQL 的 48→49 升级过程尚未实库验证；生产升级前应在对应数据库执行迁移演练。
- 当前 PostgreSQL 目标库多保留一条源库已删除的 MCP token。完整行数校验须在清理这条目标独有数据或重建目标库后重新运行；清理会使旧 token 失效。
- 外网域名只用于地址填写辅助；实际 DNS、网关入口与证书由部署方维护。

## Conclusion

已接受的 Intent 与 Plan 对应功能均已实现，自动化检查和 SQLite 迁移保值验证通过。MySQL/PostgreSQL 实库迁移、部署配置更新及存量非法编码离线更新属于上线前事项；本验证文档待用户审阅。
