# 移除服务实例语义
最后修改时间: 2026-09-16 17:26:38

Review status: Accepted

Mode: standard

## Background

当前 `Service` 通过 `instance_key` 区分同一 `Application` 下的多条运行绑定，首条通常使用 `default`。但实例键本身不是用户需要管理的配置维度；部署和界面中由它产生的 `xxx / default` 标识也泄露了内部约定。`Environment.state` 同样是已不参与业务决策的遗留存储字段，环境可用性由目标配置及 Probe 结果决定。

`Service.code` 是用户可见且稳定的运行标识：部署工作目录、Compose project、日志目录及部分路由主机名均使用它。Project 内编码保持唯一；同一 Application 可以按用户需要创建多条 Service。端口、挂载等运行时资源冲突不属于 Service 创建时的预防职责。

## Goal

1. 移除 `Service` 的实例键建模和界面上的“服务实例”概念，同时保留一个 Application 下多条 Service 的能力。
2. 删除 `instance_key` 及围绕 `default` Service 的模型、API、MCP、前端和部署元数据语义。
3. 创建 Service 时不显示实例字段。前端在选中 Application 后预填服务编码 `<application-code>-default`，用户可改为未被当前 Project 使用的编码；服务端继续以 Project scope 检查编码唯一性。
4. 保留 `Service.code` 作为运行时稳定标识；既有服务编码不因本次变更改名。
5. 移除不再使用的 `Environment.state` 字段及其模型、API 和持久化契约。

## Non-goal

- 不改变 `default_entrypoint`、变量默认值、Compose `default` network 等与 Service 实例无关的概念。
- 不检查或预先阻止同一 Application 的多个 Service 因端口、挂载等运行时资源产生冲突；部署执行结果如实反映运行时条件。
- 不改变 Project、Environment、Application、Version、Component 或 Route 的归属与授权模型。
- 不删除仍用于初始化、探测结果和环境就绪判断的 `last_probe_status`、`last_probe_revision` 等字段。
- 不为旧 HTTP、Proto 或 MCP 字段保留兼容 alias、fallback 或双路径。

## User scenarios

1. 用户在服务创建页面选择 Application 和 Version 后创建运行绑定；页面不显示实例键，服务编码预填为 `<application-code>-default`，并在保存时检查当前 Project 内未被使用。
2. 用户可为同一 Application 创建多条 Service；每条 Service 通过唯一 code 区分，并可独立选择 Version、维护覆盖配置、部署、停止、查看日志和删除。
3. Gateway 只有唯一的受管 Service；Gateway 读取、部署、停止和返回结果不再按 `default` Service 筛选，但其既有 `traefik-default` 服务编码保持不变。
4. Web、HTTP 与 MCP 的运行时操作按 Service ID 定位目标，不再接收 `instance_key` 或要求调用者提供 `default`。

## Acceptance

- [ ] `Service`、SQL schema 和 repository 查询不再包含 `instance_key`；一个 Application 可保留多条 Service，`service.code` 在同一 Project 内不可重复。
- [ ] Service 创建、基础更新、列表/详情响应、部署记录和 Gateway 响应不暴露实例键或“默认服务”字段。
- [ ] Web 不显示“服务实例”、实例键或 `xxx / default` 格式的 Service 标识；Service 创建仅移除实例字段，保留预填且可编辑的服务编码。
- [ ] 选中 Application 时，前端将新 Service code 预填为 `<application-code>-default`；服务端拒绝当前 Project 内已使用的 code，并保持 Service code 的运行目录、Compose project、日志和路由职责。
- [ ] Gateway 的唯一 Service 仍使用现有服务编码，但实现不再以 `instance_key == "default"` 识别它。
- [ ] MCP tool schema、输入和输出不再包含 `instance_key`；运行时查询以 Service ID 解析目标。
- [ ] 不新增兼容字段、Proto 保留声明、默认值兜底或旧新接口并存。
- [ ] `Environment` 不再持久化、读取或对外输出无业务用途的 `state`；环境目标和 Probe 行为保持不变。
- [ ] 覆盖多 Service 创建、Project 内 code 冲突、Version 更新、部署目标解析、Gateway 唯一 Service 以及前端创建请求的最小有效测试。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- `instance_key` 是待删除的内部多 Service 区分方式，不是 Application、Project、Environment 或用户可管理的业务字段。
- 一个 Application 可以关联多条 Service；每条 Service 保持独立的 Version 选择、覆盖配置、部署状态和唯一 `service.code`。
- `Service.code` 保留为持久化运行标识。前端仅以 Application code 加 `-default` 提供创建时的初始建议，用户可修改；服务端以 Project scope 保证唯一。该字面量不再表示可选择的实例或默认 Service。
- Gateway 也只通过唯一 Service 关联运行态，不再维护 DefaultService 选择或 instance-key 校验。
- `Environment.state` 没有生命周期或就绪性语义，应从 schema、领域模型和 HTTP/Proto 契约移除；`last_probe_status` 仍保留为探测结果。
- 本任务采用标准模式 / standard：跨越数据模型、API 契约、部署执行、MCP 与前端，但现有业务结论已足够明确，无需单独 Spec 阶段。

## Risk

- 删除 `instance_key` 是破坏性 schema 与 API 变更，Web、HTTP 和 MCP 必须同版本交付。
- 既有 Service code 会继续包含 `-default`，但它是稳定运行标识的一部分；不应在迁移时重命名，否则会改变工作目录、Compose project、日志和派生主机名。
- 多 Service 允许创建与部署，但端口、挂载和其他 Docker 资源冲突仍可能在部署时失败；它们不作为创建前拒绝的条件。
- 删除 `Environment.state` 是破坏性 API 与 schema 变更，环境调用方必须随本版本升级。
- 当前工作区已有三个 `000033_seed_pipeline.up.sql` 改动，与本任务无关，实施时必须保留并排除在任务变更范围外。

## User review notes

- 2026-09-16：用户要求移除部署中的“实例”与 `default` Service 概念。
- 2026-09-16：用户明确 `Service.code` 保留；创建 Service 不再出现实例字段，code 由所选 Application code 与 `-default` 拼接。
- 2026-09-16：用户确认同一 Application 可以按需求创建多条 Service，不以端口或挂载冲突阻止创建；当前 Project 内 Service code 必须保持唯一。
- 2026-09-16：用户澄清 Application code 加 `-default` 是前端创建时的编码建议，服务端仍检查该 code 是否已在当前 Project 使用。
- 2026-09-16：用户要求在本任务中一并删除未使用的 `Environment.state` 字段。
