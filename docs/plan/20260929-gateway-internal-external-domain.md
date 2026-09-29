# Gateway 内外网域名与自定义路由域名辅助实施计划
最后修改时间: 2026-09-29 10:01:19

Review status: Accepted

Flow mode: standard

## Intent basis

依据 [意图文档](../intent/20260929-gateway-internal-external-domain.md)：将 `base_domain` 直接更名为 `internal_domain`，新增可选 `external_domain`；保留内网域名运行时行为；Route 创建和编辑时以合法编码联动现有域名控件，手填域名不覆盖；存量非法 Route 编码由外部离线 SQL 处理。

## Implementation steps

1. 新增 Gateway schema migration，将 `base_domain` 重命名为 `internal_domain` 并增加 `external_domain`；更新 sqlc 查询、仓储与生成代码，不修改已执行迁移。
2. 统一 GatewayConfig、创建/更新/定义导入导出、项目初始化、配置加载、HTTP Proto、MCP 与 Route 配置读取中的新字段，并重新生成 Proto DTO。
3. 保持 Docker label 与 Gateway 派生地址从 `internal_domain` 生成；更新 Gateway 初始化/编辑界面和文案，提供可选外网域名，两个域名字段各占一行；核对 Web、HTTP API、MCP 的 Gateway 创建和更新入口。
4. 统一 Route 编码的前后端 DNS 单标签校验（字母开头，最多 32 字符），按主机名规则校验外网域名后缀；在创建页、列表编辑和详情编辑中实现“编码 -> 域名控件”联动，识别并保留手填域名。
5. 更新活文档和针对性测试，执行仓库规定的前后端检查。

## Files to change

- `sql/migration/{sqlite,mysql,postgres}/000049_*.sql`、`sql/query/gateway/gateway.sql`、`internal/repository/impl/sqlc/gateway/` 及生成的 sqlc 文件。
- `internal/model/`、`internal/application/{gateway,route,project_initialization,settings,deployment}/`、`internal/config/`、`internal/api/{http,mcp}/`、`proto/orbit/v1/{gateway,route,project_initialization}/` 及生成的 Proto DTO。
- `web/src/views/{gateway,project,route}/`、`web/src/i18n/locales/`、`configs/config.yaml` 和相关测试。
- 受本次字段更名影响的活产品、架构、指南和决策文档。

## Verification plan

- 验证新增 migration 保留内网域名值并建立空外网域名，Gateway 配置读写与项目初始化贯通。
- 验证 gateway mode Docker label 保持原域名结果，Route 表单自动生成与手填保护在创建和两处编辑入口一致。
- 验证 Route 编码 32 字符 DNS 单标签规则在前后端一致；无外网域名时创建表单域名留空，有外网域名时生成的完整域名实际写入 Route。
- 运行 `yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...` 和必要的定向测试。

## Blockers and assumptions

- 无阻塞。`external_domain` 为一个可选 Gateway 后缀，未配置时 Route 创建表单的域名控件留空。
- 存量不合规 Route 编码不在本任务中迁移、兼容或测试；由上线前的离线 SQL 处理。

## Risks

- `base_domain` 直接更名涉及部署 YAML 与 API 契约；部署配置需与新版本同步更新。
- 跨数据库 migration 与生成代码需一致；任何遗留旧字段引用都可能导致启动、编译或前端类型检查失败。

## Rollback

如需回退版本，先用新增迁移对应的 down migration 恢复 Gateway 字段结构，并将部署 YAML 改回旧键；Route 域名已保存为完整值，无须因本功能回写。

## User review notes

- 用户明确要求开始实现，视为接受 Intent 并授权进入 Implementation；Plan 作为标准模式的实施依据在进入实现前补齐。
- 用户确认存量不合规编码由离线 `UPDATE` 处理，业务逻辑不做兼容，本次不考虑非法存量数据。
