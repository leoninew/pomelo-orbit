# internal 分层迁移记录
最后修改时间: 2026-07-08 14:04:39

Review status: Accepted

## Background

当前正在把现有 `internal/` 目录按 `D:\SourceCodes\mywork\k12-force\docs\analyze\arch.md` 的分层风格粗暴迁移到 `internal2/`。本轮只负责移动文件和记录路径，不修改 Go 源码内容，不更新 import/package 路径；后续任务再处理包路径与编译问题。

## Goal

- 建立本任务的迁移记录文档。
- 用三列表格记录每个被处理文件的：原路径、新路径、备注。
- 后续继续迁移时，每读取并判断一个文件，就在表格中补充或更新一行。
- 如果文件用途含混、分类不明或存在分类争议，在备注中记录。

## Non-goal

- 不在本阶段修复 import 路径。
- 不修改文件内容。
- 不运行格式化、测试或编译检查。
- 不做 git add/commit/push 等 Git 写操作。
- 不主动启动、停止或重启开发服务器。

## Acceptance

- 文档位于 `docs/requirement/20260708-internal-layering-record.md`。
- 文档包含三列表格：原路径、新路径、备注。
- 已处理过的迁移项记录在表格中。
- 后续遇到含混或分类不明的文件，必须写入备注。
- 迁移任务暂停或继续时，可通过本文档恢复上下文。

## Risk / Assumption

- 当前迁移 intentionally 会造成 Go 包路径/import 暂时不可编译，后续任务处理。
- 部分文件目前按内容粗分，后续更新包路径时可能需要微调目录归属。
- `internal2/` 下现有 package 名不会立即与目录名一致，本轮不处理。
- `internal/gen` 已按生成目录整体处理；根目录 `proto/` 是否整体移动尚未在本文档中执行记录，需后续明确是否纳入本轮 `internal` 分层任务。

## Decisions

- 日志相关实现放入 `internal2/infrastructure/logger/`，不是 `internal2/infrastructure/storage/`。
- sqlc 生成代码放入 `internal2/gen/sqlc/`。
- protobuf 生成代码目录 `internal/gen` 整体移动到 `internal2/gen`，不逐个读取。
- 业务模型从旧 `internal/repository/model` 迁到 `internal2/model`。

## Migration table

| 原路径 | 新路径 | 备注 |
|---|---|---|
| `internal/gen` | `internal2/gen` | 生成代码目录整体移动，未逐文件读取。 |
| `internal/apperror/error.go` | `internal2/common/errors/error.go` | 通用应用错误。 |
| `internal/db/errors.go` | `internal2/infrastructure/database/errors.go` | 数据库错误 helper。 |
| `internal/db/sqlc/application.sql.go` | `internal2/gen/sqlc/application.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/background_task.sql.go` | `internal2/gen/sqlc/background_task.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/config_file.sql.go` | `internal2/gen/sqlc/config_file.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/credential.sql.go` | `internal2/gen/sqlc/credential.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/db.go` | `internal2/gen/sqlc/db.go` | sqlc 生成代码。 |
| `internal/db/sqlc/models.go` | `internal2/gen/sqlc/models.go` | sqlc 生成代码。 |
| `internal/db/sqlc/pipeline_template.sql.go` | `internal2/gen/sqlc/pipeline_template.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/project.sql.go` | `internal2/gen/sqlc/project.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/querier.go` | `internal2/gen/sqlc/querier.go` | sqlc 生成代码。 |
| `internal/db/sqlc/repository.sql.go` | `internal2/gen/sqlc/repository.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/role.sql.go` | `internal2/gen/sqlc/role.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/route.sql.go` | `internal2/gen/sqlc/route.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/service_config.sql.go` | `internal2/gen/sqlc/service_config.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/user.sql.go` | `internal2/gen/sqlc/user.sql.go` | sqlc 生成代码。 |
| `internal/db/sqlc/webhook.sql.go` | `internal2/gen/sqlc/webhook.sql.go` | sqlc 生成代码。 |
| `internal/infrastructure/logstore/logstore.go` | `internal2/infrastructure/logger/logstore.go` | 日志读写实现；曾临时放到 storage，已按用户意见修正为 logger。 |
| `internal/repository/api_store.go` | `internal2/repository/option.go` | 分页/查询参数辅助。 |
| `internal/repository/model/cd.go` | `internal2/model/cd.go` | CD 持久化模型/实体。 |
| `internal/repository/model/ci.go` | `internal2/model/ci.go` | CI 持久化模型/实体。 |
| `internal/repository/model/project.go` | `internal2/model/project.go` | Project 模型。 |
| `internal/repository/model/user.go` | `internal2/model/user.go` | User/Role/Permission/Auth 相关模型。 |
| `internal/repository/status.go` | `internal2/common/util/id.go` | 内容实际是 `NewId()`，不是 status；归入通用工具。 |
| `internal/repository/store.go` | `internal2/repository/impl/sqlc/store.go` | sqlx Store，仓储实现基础。 |
| `internal/repository/store_test.go` | `internal2/common/util/id_test.go` | 测试的是 `NewId()`，跟随 `id.go`。 |
| `internal/runtimepath/physical_data_root.go` | `internal2/infrastructure/storage/local/physical_data_root.go` | 运行时数据目录/宿主机路径解析，归入本地存储基础设施。 |
| `internal/runtimepath/physical_data_root_test.go` | `internal2/infrastructure/storage/local/physical_data_root_test.go` | 测试跟随本地存储路径解析实现。 |
| `internal/security/fernet.go` | `internal2/common/crypto/fernet.go` | 通用 Fernet 加解密工具。 |
| `internal/security/fernet_test.go` | `internal2/common/crypto/fernet_test.go` | 测试跟随 Fernet 工具。 |
| `internal/service/auth/csrf.go` | `internal2/auth/csrf/csrf.go` | CSRF token 能力。 |
| `internal/service/auth/token.go` | `internal2/auth/jwt/token.go` | JWT 签发/校验。 |
| `internal/service/auth/token_test.go` | `internal2/auth/jwt/token_test.go` | 测试跟随 JWT token 实现。 |
| `internal/service/cd/compose_command.go` | `internal2/workflow/activity/cd/compose_command.go` | CD Docker Compose 执行命令构造。 |
| `internal/service/cd/runner.go` | `internal2/workflow/activity/cd/runner.go` | CD 命令执行器。 |
| `internal/service/cd/workspace_test.go` | `internal2/workflow/activity/cd/workspace_test.go` | 测试 CD workspace；实现文件尚待继续处理/确认。 |
| `internal/service/ci/runner.go` | `internal2/workflow/activity/ci/runner.go` | CI Docker Runner。 |
| `internal/service/ci/workspace_test.go` | `internal2/workflow/activity/ci/workspace_test.go` | 测试 CI workspace/Docker mount。 |
| `internal/service/ci/workspace.go` | `internal2/workflow/activity/ci/workspace.go` | CI workspace 路径、artifact/log 路径和 Docker mount 计算；用途跨 application/activity，暂归 activity。 |
| `internal/status/status.go` | `internal2/common/constant/status.go` | 通用状态/任务类型常量。 |
| `internal/templatex/templatex.go` | `internal2/common/template/template.go` | 通用 Liquid 模板渲染工具。 |
| `internal/templatex/templatex_test.go` | `internal2/common/template/template_test.go` | 测试跟随模板渲染工具。 |
| `internal/app/app.go` | `internal2/bootstrap/app.go` | 应用生命周期/运行编排。 |
| `internal/app/app_test.go` | `internal2/bootstrap/app_test.go` | 测试应用迁移生命周期。 |
| `internal/app/mysql_e2e_test.go` | `internal2/test/e2e/mysql_e2e_test.go` | MySQL 端到端测试。 |
| `internal/bootstrap/bootstrap.go` | `internal2/bootstrap/provider.go` | 依赖组装/provider。 |
| `internal/db/db.go` | `internal2/infrastructure/database/database.go` | 数据库连接初始化。 |
| `internal/db/db_test.go` | `internal2/infrastructure/database/database_test.go` | 数据库连接配置测试。 |
| `internal/db/dialect.go` | `internal2/infrastructure/database/dialect.go` | SQL 方言表达式 helper。 |
| `internal/db/migration.go` | `internal2/infrastructure/database/migration.go` | 数据库迁移 runner。 |
| `internal/db/migration_test.go` | `internal2/infrastructure/database/migration_test.go` | 数据库迁移集成测试，跟随 migration runner。 |
| `internal/logging/logging.go` | `internal2/infrastructure/logger/logging.go` | 日志初始化/滚动文件 logger。 |
| `internal/logging/logging_test.go` | `internal2/infrastructure/logger/logging_test.go` | 日志初始化测试。 |
| `internal/repository/cd/repository.go` | `internal2/repository/impl/sqlc/cd/repository.go` | CD 仓储 sqlx 实现。 |
| `internal/repository/ci/repository.go` | `internal2/repository/impl/sqlc/ci/repository.go` | CI 仓储 sqlx 实现。 |
| `internal/repository/ci/repository.go` | `internal2/repository/impl/sqlc/ci/repository.go` | CI 仓储 sqlx 实现。 |
| `internal/repository/ci/repository_test.go` | `internal2/repository/impl/sqlc/ci/repository_test.go` | CI 仓储实现测试。 |
| `internal/repository/dbmodel/convert.go` | `internal2/repository/impl/sqlc/dbmodel/convert.go` | sqlc row 与 model 转换 helper。 |
| `internal/repository/project/repository.go` | `internal2/repository/impl/sqlc/project/repository.go` | Project 仓储 sqlc/sqlx 实现。 |
| `internal/repository/role/repository.go` | `internal2/repository/impl/sqlc/role/repository.go` | Role/Permission 仓储 sqlc/sqlx 实现。 |
| `internal/repository/task/repository.go` | `internal2/repository/impl/sqlc/task/repository.go` | background task 队列仓储实现；文件内含 Task 持久化模型，后续可考虑拆到 model/queue。 |
| `internal/repository/task/repository_test.go` | `internal2/repository/impl/sqlc/task/repository_test.go` | background task 队列仓储测试。 |
| `internal/repository/user/repository.go` | `internal2/repository/impl/sqlc/user/repository.go` | User/Auth 仓储 sqlc/sqlx 实现。 |
| `internal/service/auth/service.go` | `internal2/application/auth/service.go` | 登录、改密、登录历史等认证应用用例服务。 |
| `internal/service/cd/application_extra.go` | `internal2/application/cd/application_extra.go` | CD application/config/route/import-export/stop-restart 等应用用例服务扩展。 |
| `internal/service/cd/compose_test.go` | `internal2/application/cd/compose_test.go` | CD 模板渲染、compose route label 注入、域名校验测试。 |
| `internal/service/cd/deployment_execution.go` | `internal2/workflow/activity/cd/deployment_execution.go` | CD 部署/重启/停止执行编排，属于 workflow activity。 |
| `internal/service/cd/deployment_execution_test.go` | `internal2/workflow/activity/cd/deployment_execution_test.go` | CD deployment execution activity 测试。 |
| `internal/service/cd/route.go` | `internal2/application/cd/route.go` | CD 路由/证书/Traefik 配置应用服务；含文件同步和外部命令等基础设施细节，后续可拆。 |
| `internal/service/cd/service.go` | `internal2/application/cd/service.go` | CD 应用层主服务与接口聚合；后续可拆接口定义。 |
| `internal/service/cd/workspace.go` | `internal2/workflow/activity/cd/workspace.go` | CD workspace 路径计算与物理数据根解析；用途跨 application/activity，暂归 activity。 |
| `internal/service/ci/artifact.go` | `internal2/application/ci/artifact.go` | CI artifact 查询应用用例。 |
| `internal/service/ci/build_stage.go` | `internal2/application/ci/build_stage.go` | CI build stage CRUD/复制/校验应用用例。 |
| `internal/service/ci/credential.go` | `internal2/application/ci/credential.go` | CI credential CRUD/import/export 与加解密应用用例。 |
| `internal/service/ci/execution.go` | `internal2/workflow/activity/ci/execution.go` | CI pipeline run 执行编排与 stage 变量渲染，属于 workflow activity。 |
| `internal/service/ci/execution_test.go` | `internal2/workflow/activity/ci/execution_test.go` | CI pipeline run execution activity 测试。 |
| `internal/service/ci/executor.go` | `internal2/workflow/activity/ci/executor.go` | CI stage DAG/layer 执行器、容器执行与 artifact 保存 activity。 |
| `internal/service/ci/pipeline_run.go` | `internal2/application/ci/pipeline_run.go` | CI pipeline run 触发、列表、详情、日志、取消、重试应用用例。 |
| `internal/service/ci/repository.go` | `internal2/application/ci/repository.go` | CI repository/webhook 主应用服务与接口聚合；也聚合 execution store 接口。 |
| `internal/service/ci/repository_test.go` | `internal2/application/ci/repository_test.go` | CI repository/webhook 应用服务兼容性测试。 |
| `internal/service/ci/runtime_variables.go` | `internal2/application/ci/runtime_variables.go` | CI pipeline runtime variables 构建/校验；跨触发与执行，暂归 application。 |
| `internal/service/ci/runtime_variables_test.go` | `internal2/application/ci/runtime_variables_test.go` | CI runtime variables 测试。 |
| `internal/service/ci/snapshot.go` | `internal2/application/ci/snapshot.go` | CI pipeline snapshot 创建/读取与模板 stage 快照构建。 |
| `internal/service/ci/template.go` | `internal2/application/ci/template.go` | CI pipeline template/build stage detail/orchestration/变量解析应用用例。 |
| `internal/service/ci/template_test.go` | `internal2/application/ci/template_test.go` | 测试跟随 CI pipeline template 应用用例。 |
| `internal/service/project/service.go` | `internal2/application/project/service.go` | Project CRUD、成员管理和业务校验应用用例服务。 |
| `internal/service/role/service.go` | `internal2/application/role/service.go` | Role/Permission 创建、更新、删除和校验应用用例服务。 |
| `internal/service/task/service.go` | `internal2/queue/task/service.go` | background task 入队、payload 校验和任务查询服务；队列语义更强，归入 queue。 |
| `internal/service/user/service.go` | `internal2/application/user/service.go` | User 创建、更新、状态变更、删除和密码哈希应用用例服务。 |
| `internal/transport/http/turnstile.go` | `internal2/api/http/turnstile.go` | HTTP 登录/注册链路使用的 Cloudflare Turnstile verifier。 |
| `internal/worker/handler/cd/handler.go` | `internal2/worker/handler/cd/handler.go` | CD background task payload 解析与 deploy/restart/stop worker handler。 |
| `internal/worker/handler/cd/handler_test.go` | `internal2/worker/handler/cd/handler_test.go` | 测试跟随 CD worker handler。 |
| `internal/worker/handler/ci/handler.go` | `internal2/worker/handler/ci/handler.go` | CI background task payload 解析与 pipeline run worker handler。 |
| `internal/worker/handler/ci/handler_test.go` | `internal2/worker/handler/ci/handler_test.go` | 测试跟随 CI worker handler。 |
| `internal/worker/worker.go` | `internal2/worker/worker.go` | background worker router、polling claim/complete/fail 执行循环。 |
| `internal/worker/worker_test.go` | `internal2/worker/worker_test.go` | 测试跟随 background worker 执行循环。 |
| `internal/transport/http/application_routes_test.go` | `internal2/api/http/application_routes_test.go` | CD application HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/artifact_routes_test.go` | `internal2/api/http/artifact_routes_test.go` | CI artifact HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/auth_routes_test.go` | `internal2/api/http/auth_routes_test.go` | Auth HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/build_stage_routes_test.go` | `internal2/api/http/build_stage_routes_test.go` | CI build-stage HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/credential_routes_test.go` | `internal2/api/http/credential_routes_test.go` | CI credential HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/deployment_routes_test.go` | `internal2/api/http/deployment_routes_test.go` | CD deployment HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/pipeline_run_routes_test.go` | `internal2/api/http/pipeline_run_routes_test.go` | CI pipeline run HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/project_routes_test.go` | `internal2/api/http/project_routes_test.go` | Project HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/repository_routes_test.go` | `internal2/api/http/repository_routes_test.go` | CI repository/webhook/trigger HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/route_routes_test.go` | `internal2/api/http/route_routes_test.go` | CD route/certificate HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/server_test.go` | `internal2/api/http/server_test.go` | HTTP server、健康检查、静态资源 fallback、auth/background task 等集成测试；跟随 HTTP API 层。 |
| `internal/transport/http/settings_routes_test.go` | `internal2/api/http/settings_routes_test.go` | Settings config HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/snapshot_routes_test.go` | `internal2/api/http/snapshot_routes_test.go` | CI pipeline snapshot HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/template_routes_test.go` | `internal2/api/http/template_routes_test.go` | CI pipeline template HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/user_role_routes_test.go` | `internal2/api/http/user_role_routes_test.go` | User/Role HTTP 路由集成测试，并包含测试 auth helper；跟随 HTTP API 层。 |
| `internal/config/config.go` | `internal2/config/config.go` | 配置结构、加载、环境变量绑定和校验；保留独立 config 层。 |
| `internal/config/config_test.go` | `internal2/config/config_test.go` | 测试跟随配置加载与校验。 |
| `internal/service/settings/service.go` | `internal2/application/settings/service.go` | 系统配置读取、更新、重置应用服务；含 `.env` 文件读写基础设施细节，后续可拆。 |
| `internal/transport/http/middleware/cors.go` | `internal2/api/http/middleware/cors.go` | Gin CORS middleware，归入 HTTP API middleware。 |
| `internal/transport/http/auth_support.go` | `internal2/api/http/auth_support.go` | HTTP Server currentUser/jwtSecret 支撑函数。 |
| `internal/transport/http/handler/authz/authz_test.go` | `internal2/api/http/handler/authz/authz_test.go` | HTTP authz/current user 测试，跟随 authz handler。 |
| `internal/transport/http/middleware/cors_test.go` | `internal2/api/http/middleware/cors_test.go` | 测试跟随 Gin CORS middleware。 |
| `internal/transport/http/traefik_route_test.go` | `internal2/api/http/traefik_route_test.go` | Traefik route/config HTTP 路由集成测试，跟随 HTTP API 层。 |
| `internal/transport/http/turnstile_test.go` | `internal2/api/http/turnstile_test.go` | 测试跟随 Cloudflare Turnstile verifier。 |
| `internal/transport/http/middleware/logging_test.go` | `internal2/api/http/middleware/logging_test.go` | 测试跟随 Gin HTTP logging middleware。 |
| `internal/transport/http/middleware/logging.go` | `internal2/api/http/middleware/logging.go` | Gin request id、real ip、request/response logging、recovery middleware。 |
| `internal/transport/http/ci_api_compatibility_test.go` | `internal2/api/http/ci_api_compatibility_test.go` | CI HTTP API JSON 兼容性/响应字段契约测试，跟随 HTTP API 层。 |
| `internal/transport/http/response/response_test.go` | `internal2/api/http/response/response_test.go` | 空测试占位文件，跟随 HTTP response 层。 |
| `internal/transport/http/permission.go` | `internal2/api/http/permission.go` | HTTP permission guard 与 current user permission 响应结构。 |
| `internal/transport/http/handler/authz/authz.go` | `internal2/api/http/handler/authz/authz.go` | HTTP authz authenticator、current user 与 permission 校验 handler 支撑。 |
| `internal/transport/http/codec/protojson_test.go` | `internal2/api/http/codec/protojson_test.go` | 测试跟随 HTTP protojson codec。 |
| `internal/transport/http/response/response.go` | `internal2/api/http/response/response.go` | HTTP JSON/proto decode、分页、时间/值转换响应 helper。 |
| `internal/transport/http/codec/protojson.go` | `internal2/api/http/codec/protojson.go` | Gin/protobuf HTTP JSON codec 与 protojson marshal/unmarshal 配置。 |
| `internal/transport/http/handler/auth/handler.go` | `internal2/api/http/handler/auth/handler.go` | Auth HTTP handler：csrf、login、me、password、login history、OAuth 占位。 |
| `internal/transport/http/handler/project/handler.go` | `internal2/api/http/handler/project/handler.go` | Project HTTP handler：项目 CRUD、废弃、成员管理响应转换。 |
| `internal/transport/http/handler/role/handler.go` | `internal2/api/http/handler/role/handler.go` | Role/Permission HTTP handler：列表、详情、创建、更新、删除和响应转换。 |
| `internal/transport/http/handler/settings/handler.go` | `internal2/api/http/handler/settings/handler.go` | Settings HTTP handler：系统配置读取、更新、重置与响应转换。 |
| `internal/transport/http/handler/user/handler.go` | `internal2/api/http/handler/user/handler.go` | User HTTP handler：用户 CRUD、启停、角色绑定、权限保护和响应转换。 |
| `internal/transport/http/handler/task/handler.go` | `internal2/api/http/handler/task/handler.go` | Background task HTTP handler：任务创建、查询和 CI/CD task enqueue 入口。 |
| `internal/transport/http/handler/cd/application_extra.go` | `internal2/api/http/handler/cd/application_extra.go` | CD application extra HTTP routes：import/export/files/stop/restart/status/logs/route/service-config。 |
| `internal/transport/http/handler/cd/handler.go` | `internal2/api/http/handler/cd/handler.go` | CD application/deployment HTTP handler：CRUD、deploy、deployment list/detail/log/cancel。 |
| `internal/transport/http/handler/cd/route.go` | `internal2/api/http/handler/cd/route.go` | CD route/Traefik route HTTP handler：route CRUD、证书、HTTPS、同步、Traefik 列表。 |
| `internal/transport/http/handler/ci/artifact.go` | `internal2/api/http/handler/ci/artifact.go` | CI artifact HTTP handler：artifact 列表查询与分页响应。 |
| `internal/transport/http/handler/ci/build_stage.go` | `internal2/api/http/handler/ci/build_stage.go` | CI build-stage HTTP handler：stage CRUD、复制、artifact 配置转换。 |
| `internal/transport/http/handler/ci/credential.go` | `internal2/api/http/handler/ci/credential.go` | CI credential HTTP handler：凭据列表、导入导出、CRUD 和响应脱敏转换。 |
| `internal/transport/http/handler/ci/dto.go` | `internal2/api/http/handler/ci/dto.go` | CI repository webhook update DTO/JSON 兼容辅助，区分 `branch_filter` 是否显式传入。 |
| `internal/transport/http/handler/ci/pipeline_run.go` | `internal2/api/http/handler/ci/pipeline_run.go` | CI pipeline run HTTP handler：触发、列表、详情、artifact、stage log、取消、重试和响应转换。 |
| `internal/transport/http/handler/ci/repository.go` | `internal2/api/http/handler/ci/repository.go` | CI repository/webhook HTTP handler：repository CRUD、webhook CRUD/receive、错误响应和分页响应转换。 |
| `internal/transport/http/handler/ci/snapshot.go` | `internal2/api/http/handler/ci/snapshot.go` | CI pipeline snapshot HTTP handler：snapshot 详情路由和 stage/artifact/variable 响应转换。 |
| `internal/transport/http/handler/ci/template.go` | `internal2/api/http/handler/ci/template.go` | CI pipeline template HTTP handler：template CRUD、复制、变量解析、编排和变量响应/请求转换。 |
| `internal/transport/http/server.go` | `internal2/api/http/server.go` | Gin HTTP server 组装：middleware、健康检查、auth/user/role/settings/project/CI/CD/task 路由注册、静态资源 fallback 与 runtime config 注入。 |

## Open questions

- 根目录 `proto/` 是否也要在本轮整体迁入某个 `internal2`/`api`/`gen` 结构？当前尚未执行。
- `internal/service/*` 中哪些文件应进入 `application/workflow`，哪些保留为 domain service，需要继续逐文件读取判断。
- `internal/transport/http/*` 大量路由测试和 server 代码应按 `internal2/api/http` 细分到 router/handler/middleware/response/codec，需继续逐文件判断。

## User review notes

- 用户要求：一次处理一个文件，处理一个打印一条信息。
- 用户要求：读取源码、基于 `arch.md` 判断移动到 `internal2` 对应位置，没有目录可以新建。
- 用户要求：用途含混或分类不明时，写进备注。
- 用户纠正：日志应该在 `infrastructure/logger/`，不是 `infrastructure/storage/`。
