# 移除环境领域验证

最后修改时间: 2026-07-26 16:52:55

Review status: Draft

流程模式: light

## 需求对齐

- 环境表、环境权限和默认环境种子已从 SQLite/MySQL 的既有建表、种子脚本移除；服务唯一键改为应用和实例键。
- 后端模型、仓储、用例、HTTP 路由、Proto 及 SQLC/Proto 生成代码不再声明环境领域或字段。
- 部署工作目录、Compose 项目名、运行状态、日志和预览均以应用与实例键定位。
- 前端环境页面、路由、导航、API、部署选择器、服务筛选和展示字段已删除；导航分组改为“运行”。
- MCP 不再查询环境 API、接受环境 ID 或将环境编码写入工作目录。

## 实际 Diff

- 数据库、SQLC 查询和生成模型：删除环境资源，直接修改既有迁移/种子脚本，不新增增量迁移。
- Go/Proto：删除环境消息与依赖注入，收敛服务和部署接口到应用加实例键。
- 前端：删除环境资源页面及生成类型，更新应用、版本、网关、服务和导航视图。
- MCP：删除环境工具和客户端请求，运行目录调整为 `deployment/<应用>/<实例>`。

实际改动与 Requirement 一致。MCP 调整是为消除其对已删除 API 的直接调用，已补充进 Requirement。

## 验收清单

- [x] 既有 SQLite/MySQL 脚本不再创建或引用 `environment`、`environment_id`。
- [x] 服务唯一维度为 `application_id + instance_key`。
- [x] 后端、Proto、SQLC 和前端生成代码不再保留环境领域类型或参数。
- [x] 部署路径与 Compose 项目名不再包含环境层级。
- [x] 前端无环境页面、路由、导航、API、筛选或部署字段。
- [x] MCP 不再调用环境 API 或接受环境 ID。
- [x] 开发数据库按重建后的既有脚本初始化；不提供旧数据迁移或兼容。

## 命令结果

- 通过：`bin\\sqlc.exe generate`
- 通过：`bin\\buf.exe generate`
- 通过：`go test ./...`
- 通过：`npm run typecheck`
- 通过：`npm run lint`
- 通过：`npm run test`，7 个文件、39 项测试。
- 通过：`uv run ruff check src tests`、`uv run ruff format --check src tests`、`uv run mypy`、`uv run pytest`，15 项通过、1 项 Docker 集成测试按标记跳过。
- 通过：`git diff --cached --check`
- 未通过：`npm run build`。`reka-ui@2.9.7` 的 `dist/index.js` 引用了缺失的 `./date/index.js`；按 `yarn.lock` 执行 `yarn --cwd web install --frozen-lockfile` 后仍可复现。开发服务器使用 `node_modules/.vite/deps/reka-ui.js` 的预构建缓存，未直接解析该入口，因此可正常运行。
- 未通过：`bin\\buf.exe lint`。全量既有领域 Proto 的包版本后缀规则均不符合当前 Buf 默认规则，包含未修改文件；`Taskfile.yml` 未将该命令列为检查入口。

## 范围偏差

无功能范围偏差。为使客户端契约一致，新增了 MCP 环境 API/参数的移除，已在 Requirement 中记录。

## 风险与未完成项

- 开发数据库必须重建；旧数据库、旧前端和外部 API 调用无法继续使用。
- 前端生产构建在当前锁定依赖下不可用；开发服务器的 Vite 预构建缓存掩盖了该问题，清空缓存或在新环境安装后也应重新确认。需单独修复或更新 `reka-ui` 的发布包/锁定版本后重跑构建。
- Buf lint 的全仓包命名规则问题不由本次变更引入，需单独调整 Buf 配置或 Proto 包名。
- 未连接实际 MySQL 实例验证重建后的脚本；Go 集成测试覆盖 SQLite 迁移路径。

## 结论

环境领域移除的代码契约、生成链路、Go 测试、前端类型/lint/单测及 MCP 检查均通过。验证状态保持 Draft，等待前端生产构建依赖问题与既有 Buf lint 基线问题的处置决定。
