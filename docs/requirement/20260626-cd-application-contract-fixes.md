# CD 应用管理契约修复需求
最后修改时间: 2026-06-26 21:48:22

Review status: Accepted

## Background

`docs/analyze/application-management-usability.md` 中“3. 还发现几个会直接影响易用性的实现/契约问题”指出，当前应用管理存在若干前后端契约不一致或文案与实际实现不一致的问题。这些问题会让用户产生“配置已经保存但没有生效”“勾选了但行为没发生”“文案描述和实际目录不一致”等困惑。

本次后端目标明确为 `backend-go`，不处理旧 Python backend。

## Goal

修复以下直接影响应用管理易用性的实现/契约问题：

1. 新建应用时，前端传入的 `route_managed` 应在 backend-go 中正确接收并持久化。
2. 删除应用时，“同时删除应用工作目录”的前后端请求契约应一致，并确保用户勾选后 backend-go 实际收到 `remove_dir=true`。
3. 前端文案中的应用工作目录描述应与 backend-go 实际目录模型一致，避免继续提示旧路径 `data/apps/{code}`。
4. 服务镜像覆盖的“重置/留空移除覆盖”行为应与 UI 文案一致：前端发送空值或 `null` 时，backend-go 应移除该 service 的 image 覆盖，而不是报错。

## Non-goal

- 不实现完整应用创建向导。
- 不实现部署准备状态、部署前预检或失败原因解释。
- 不重构应用配置文件模型。
- 不处理 Python backend。
- 不修改已执行迁移文件。
- 不做兼容层或额外别名，只统一当前前后端契约到目标状态。

## User scenarios

- 用户新建应用时勾选“启用路由管理”，创建成功后进入详情页可以看到路由管理已启用，并可继续添加路由。
- 用户删除未运行的应用时勾选“同时删除应用工作目录”，后端实际按该选项删除对应工作目录。
- 用户查看删除确认文案时，看到的工作目录描述与 backend-go 的实际工作目录语义一致。
- 用户在服务镜像配置中重置覆盖，或按提示留空保存后，服务镜像回退到 compose 中的基础 image。

## Acceptance

- backend-go `POST /api/cd/application` 请求模型、handler 和 service 能正确处理 `route_managed`。
- 前端删除应用传参方式与 backend-go `DELETE /api/cd/application/{app_id}` 接口一致。
- 删除工作目录相关中文文案不再出现旧路径 `data/apps/{code}`，并与 backend-go `DataRoot/cd/{code}` 的工作目录模型一致。
- backend-go `PUT /api/cd/application/{app_id}/service-config/{service_name}` 支持 `image: null` 或空字符串表达移除 image 覆盖。
- 前端镜像覆盖重置和留空保存不会触发后端校验错误，并能刷新为基础 image 状态。
- 补充或更新必要的 Go/前端测试；如缺少既有测试入口，至少运行相关项目检查并记录结果。
- 前端代码变更后按项目要求运行 `yarn lint --fix && yarn typecheck`，如失败需修复或记录阻塞原因。

## Open questions

暂无必须阻塞实现的问题。

## Decisions

- 使用 light / 轻量模式。
- 本次后端仅处理 `backend-go`。
- 范围限定为 `docs/analyze/application-management-usability.md` 第 3 节列出的实现/契约问题。
- 不扩大到应用管理整体易用性优化。

## Risk

- 删除工作目录的请求契约调整需要避免与现有前端调用方式不一致；本次以后端现状或前端现状二选一统一，不保留双写兼容。
- 服务镜像覆盖移除行为可能涉及 repository upsert 后记录仍存在但 image 为 null，需确认 UI 能正确展示为未覆盖。
- 前端 lint/typecheck 可能暴露与本次无关的既有问题；如出现需如实记录。
