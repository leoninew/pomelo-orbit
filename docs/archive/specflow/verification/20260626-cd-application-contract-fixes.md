# CD 应用管理契约修复验证
最后修改时间: 2026-06-26 22:17:11

Review status: Accepted

## Requirement alignment / 需求对齐

依据 `docs/requirement/20260626-cd-application-contract-fixes.md` 核对，本次实现覆盖需求中限定的 4 个契约问题：

1. 新建应用时 `route_managed` 正确进入 backend-go 请求模型，并由 service 持久化到 `application.route_managed`。
2. 删除应用时前端改为通过 query 参数传递 `remove_dir`，与 backend-go `DELETE /api/cd/application/{app_id}` 当前契约一致。
3. 删除应用工作目录文案从旧路径 `data/apps/{code}` 修正为 backend-go 实际语义 `data/cd/{code}`。
4. 服务镜像覆盖接口允许 `image: null` 或空字符串表达移除覆盖，并返回基础 compose image 状态。

需求中的 Non-goal 也保持一致：未实现创建向导、部署预检、配置文件模型重构；未处理 Python backend；未修改迁移文件；未加入兼容层或额外别名。

## Spec alignment / 规格对齐

不适用。当前流程为轻量模式 / light，本次没有单独创建 spec 文档，按 requirement / 需求核对。

## Plan alignment / 计划对齐

不适用。当前流程为轻量模式 / light，本次没有单独创建 plan 文档，按 requirement / 需求直接实现并验证。

## Actual diff summary / 实际差异摘要

跟踪文件 diff 统计：

```text
7 files changed, 49 insertions(+), 11 deletions(-)
```

实际修改包括：

- `backend-go/internal/transport/http/handler/cd/handler.go`
  - `ApplicationCreateReq` 增加 `RouteManaged bool`。
  - `createApplication` 将 `req.RouteManaged` 传入 `cdsvc.ApplicationCreateInput`。
- `backend-go/internal/service/cd/service.go`
  - `ApplicationCreateInput` 增加 `RouteManaged`。
  - `CreateApplication` 创建 `model.Application` 时写入 `RouteManaged: input.RouteManaged`。
- `backend-go/internal/service/cd/application_extra.go`
  - 移除 `image == nil` 直接报错逻辑。
  - 当没有已有 service config 且 image 为空时，直接返回基于 compose 原始 image 的视图。
  - 当已有 service config 时，允许将 `config.Image` 更新为 `nil`，表达移除覆盖。
- `backend-go/internal/transport/http/application_routes_test.go`
  - 增加 `route_managed` 创建断言。
  - 增加 `remove_dir=true` 删除实际工作目录断言。
  - 增加 `image:null` 和 `image:""` 重置镜像覆盖断言。
- `frontend/src/api/cd/application.ts`
  - 删除应用请求从 DELETE body 改为 query params：`params: { remove_dir: removeDir }`。
- `frontend/src/i18n/locales/zh-CN.ts`
  - 删除工作目录文案改为 `data/cd/{code}`。
- `frontend/src/i18n/locales/en-US.ts`
  - 英文删除工作目录文案改为 `data/cd/{code}`。

另外，当前工作区还包含此前按用户要求新增的分析文档目录：

- `docs/analyze/application-management-usability.md`

以及本次 SpecFlow requirement 文档：

- `docs/requirement/20260626-cd-application-contract-fixes.md`

## Expected vs actual changed files / 预期与实际改动对比

| 预期范围 | 实际文件 | 结论 |
| --- | --- | --- |
| backend-go 创建应用契约 | `backend-go/internal/transport/http/handler/cd/handler.go`, `backend-go/internal/service/cd/service.go` | 符合 |
| backend-go service image reset 行为 | `backend-go/internal/service/cd/application_extra.go` | 符合 |
| backend-go 相关接口测试 | `backend-go/internal/transport/http/application_routes_test.go` | 符合 |
| 前端删除应用请求契约 | `frontend/src/api/cd/application.ts` | 符合 |
| 前端工作目录文案 | `frontend/src/i18n/locales/zh-CN.ts`, `frontend/src/i18n/locales/en-US.ts` | 符合 |
| 旧 Python backend | 无修改 | 符合 Non-goal |
| 迁移文件 | 无修改 | 符合 Non-goal |

未发现超出本次需求边界的产品功能扩展。

## Acceptance criteria checklist / 验收清单

- [x] backend-go `POST /api/cd/application` 请求模型、handler 和 service 能正确处理 `route_managed`。
- [x] 前端删除应用传参方式与 backend-go `DELETE /api/cd/application/{app_id}` 接口一致。
- [x] 删除工作目录相关中文文案不再出现旧路径 `data/apps/{code}`，并与 backend-go `DataRoot/cd/{code}` 工作目录模型一致。
- [x] 英文工作目录文案同步修正为 `data/cd/{code}`。
- [x] backend-go `PUT /api/cd/application/{app_id}/service-config/{service_name}` 支持 `image: null` 表达移除 image 覆盖。
- [x] backend-go 同一接口支持空字符串 `image: ""` 表达移除 image 覆盖。
- [x] 前端代码变更后运行 `yarn lint --fix && yarn typecheck` 并通过。
- [x] 补充 Go 接口测试覆盖本次主要契约变化。

## Command results / 命令结果

### backend-go 相关测试

命令：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/backend-go && go test ./internal/transport/http ./internal/service/cd
```

结果：通过。

```text
ok  	backend/internal/transport/http	(cached)
ok  	backend/internal/service/cd	(cached)
```

### 前端 lint 与 typecheck

命令：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn lint --fix && yarn typecheck
```

结果：通过。

```text
yarn run v1.22.22
warning package.json: No license field
$ eslint . --cache --fix
Done in 38.06s.
yarn run v1.22.22
warning package.json: No license field
$ vue-tsc --noEmit
Done in 17.21s.
```

### 文案残留检查

检查范围：`frontend/src/i18n/locales`。

搜索 `data/apps` 结果：未发现匹配。

## Missed or expanded scope / 范围偏差

- 未运行全量 `go test ./...`，本次只运行与 CD 应用接口和 service 直接相关的 backend-go 包测试。
- 未做浏览器端手工验证；当前验证覆盖接口测试、前端 lint 和 typecheck。
- 未修改旧 Python backend，符合本次用户明确要求“后端是 backend-go”。
- 未实现应用管理整体易用性改进，只处理分析文档第 3 节的契约问题，符合 light 模式范围。

## Risks / 风险

- `image: null` / `image: ""` 重置覆盖后，当前实现可能保留一条 `application_service` 配置记录但 `image` 为 null；本次接口视图验证显示会展示为未覆盖并回退基础 image。若后续列表或导出希望完全删除记录，需要另开需求处理。
- 删除应用工作目录的自动化测试覆盖了 `remove_dir=true` 与 `data/cd/{code}` 路径，但未覆盖真实部署目录中复杂文件权限或占用场景。
- 前端没有新增单元测试；当前依赖 TypeScript 类型检查和 backend-go 接口测试验证契约。

## Incomplete items / 未完成项

无阻塞交付的未完成项。

可选后续工作不属于本次范围：

- 为前端应用删除弹窗或 service image 重置交互补充组件测试。
- 为服务镜像覆盖记录的 null 状态制定更明确的数据清理策略。
- 继续推进 `docs/analyze/application-management-usability.md` 中 P0/P1 的可部署性可见化和应用创建向导能力。

## Conclusion / 结论

本次 light / 轻量模式 Verification 通过。实现与 `docs/requirement/20260626-cd-application-contract-fixes.md` 中定义的目标、非目标和验收标准一致；相关 backend-go 测试、前端 lint 和 typecheck 均通过；未发现需要阻塞交付的范围偏差。
