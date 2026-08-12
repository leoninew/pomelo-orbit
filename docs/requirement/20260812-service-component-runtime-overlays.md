# 服务组件运行时基础配置覆盖
最后修改时间: 2026-08-12 20:01:34

Review status: Accepted

Mode: standard

## Background

Service 是 Application 的一个运行实例，当前通过 `source_version_component_id` 绑定 Version 中声明的组件。Service 组件只保存环境变量、挂载、资源与端点的稀疏覆盖；启动命令、拉取策略和重启策略仍只能由 Version 编辑。

同一 Version 需要派生多个运行实例时，实例可能使用相同镜像、组件身份和接口契约，却需要不同容器启动参数、拉取行为或重启行为。例如，同一组件在不同 Service 中以不同参数执行常驻或任务角色。当前只能新建或修改 Version，无法将这类实例级差异保存在 Service 中。

## Goal

1. 在不改变 Service 与 Version Component 映射关系及镜像来源的前提下，允许 Service 对指定组件覆盖以下运行时基础配置：
   - `entrypoint`
   - `command`
   - `pull_policy`
   - `restart_policy`
2. Service 组件详情展示 Version 默认值、当前 Service 配置及其继承或覆盖来源，并允许编辑上述覆盖字段。
3. Service 的有效部署计划、预览、配置哈希和 Compose 渲染必须使用相同的合并结果；变更后标记为等待显式部署。
4. HTTP API、MCP `orbit_update_service_component_overlay` 与 Web 使用同一覆盖契约。

## Business Model

### 组件身份与职责边界

Service Component 继续是 Version Component 的运行实例映射，不是完整组件副本：

| 字段组 | 归属 |
| --- | --- |
| 镜像、组件名、`source_version_component_id`、依赖、健康检查、设备、`tmpfs`、`ulimits`、端点的协议及容器端口、挂载类型/目标/只读属性 | 仅 Version 声明 |
| 环境变量、挂载源、资源、端点可调字段 | 既有 Service 稀疏覆盖 |
| 入口点、启动命令、拉取策略、重启策略 | 新增 Service 稀疏覆盖 |

Service 不得重命名组件，也不得将覆盖关联到未由选定 Version 声明的组件。切换 Service 的 Version 时，既有的组件映射和覆盖仍按组件名重新映射并校验兼容性；无法匹配的 Version 必须被拒绝。

### 覆盖语义

新的基础配置覆盖按字段独立生效：没有覆盖记录时继承 Version；有覆盖记录时使用 Service 值。它们不是新旧配置并存，也不通过默认值隐式补齐。

| 字段 | 继承 | 覆盖 | 显式清空 |
| --- | --- | --- | --- |
| `entrypoint` | 使用 Version argv | 使用 Service argv | `[]`，使 Compose 不输出 `entrypoint`，采用镜像默认入口点 |
| `command` | 使用 Version argv | 使用 Service argv | `[]`，使 Compose 不输出 `command`，采用镜像默认命令 |
| `pull_policy` | 使用 Version 值 | `always`、`missing` 或 `never` | 不适用 |
| `restart_policy` | 使用 Version 值（可能为空） | 使用支持的策略值；`no` 显式禁用重启 | 不适用 |

`entrypoint` 与 `command` 从 UI/API 的 shell-quoted 文本输入解析为 argv；沿用 Version 组件的 `commandline.Parse` 规则，不执行 shell、不展开环境变量或命令替换。响应将 argv 格式化为可解析回同一 argv 的文本。

### 生效链路

```text
Version Component declaration
  + Service Component sparse overlays
  -> EffectiveServiceComponent
  -> EffectiveServicePlan / plan hash / pending deploy
  -> preview and Compose render
  -> explicit deployment
```

所有读取和部署路径必须调用同一合并逻辑。Service 详情不能由前端自行推断生效值；当已有 Service 配置无法形成有效计划时，返回声明和覆盖，并明确缺少有效值。

## User Scenarios

1. 操作者为一个实例覆盖 `entrypoint` 或 `command`，使其以不同角色运行；另一个引用同一 Version 的 Service 不受影响。
2. 操作者清空已在 Version 中声明的 `command` 或 `entrypoint`，让容器使用镜像默认值；该状态不同于继承 Version 的空/非空值。
3. 操作者为某实例调整 `pull_policy` 或 `restart_policy`，保存后只影响该 Service 的待部署状态与下一次部署。
4. 操作者重置任一覆盖字段后，该字段立即恢复继承 Version；其他字段的覆盖不被重置。
5. MCP 调用方读取 Service 后更新同一组件覆盖，获得与 Web API 相同的校验、持久化和有效部署行为。

## Non-goal

- 不允许 Service 重命名、增加、删除或替换其 Version 声明的组件映射。
- 不允许 Service 覆盖镜像、组件依赖、健康检查、设备、`tmpfs`、`ulimits`、端点协议或容器端口，以及挂载的类型、目标或只读属性。
- 不将 Service 覆盖升级为独立 Version，不复制 Version 的制品元数据，也不新建镜像构建/制品选择流程。
- 不在保存时自动部署、重启或拉取镜像；运行状态仍仅通过显式部署操作改变。
- 不修改已执行迁移；需要持久化变更时新增 SQLite 和 MySQL 迁移。

## Interaction Rules

- 服务组件详情将基础运行配置置于独立分组，使用 Version 默认值、当前 Service 值和来源状态表达继承或覆盖。
- 拉取策略、重启策略使用符合其类型的输入控件；`entrypoint` 和 `command` 使用命令文本输入。
- 每个字段可以单独重置为 Version 值；`entrypoint` 和 `command` 还必须能表示显式清空，`restart_policy=no` 表示显式禁用重启。
- 组件详情中所有 Service 覆盖卡片使用同一“重置为 Version 值”语义：清除该字段或声明项的 Service 覆盖并恢复 Version 声明；不提供将未覆盖的 Version 声明标记为删除的入口。已有删除覆盖显示同一重置动作。
- 保存采用现有“替换一个组件的全部稀疏覆盖”请求模型，必须保留未在当前界面修改的环境变量、挂载、资源和端点覆盖。
- 成功保存后重新读取服务组件详情，并提示等待显式部署。

## Acceptance

- [ ] Service Component 可持久化并读取 `entrypoint`、`command`、`pull_policy` 和 `restart_policy` 的字段级覆盖，未覆盖字段继承 Version；镜像始终使用 Version 声明。
- [ ] 拉取策略只能为 `always`、`missing` 或 `never`；重启策略采用 Version 组件已有的有效值集合。
- [ ] `entrypoint` 和 `command` 能区分继承、非空 argv 覆盖和显式空 argv；重启策略能区分继承、策略覆盖和 `no` 的显式禁用。
- [ ] 命令输入按既有安全 argv 解析规则校验，非法引号或命令文本返回字段级校验错误；不会执行 shell 展开或命令替换。
- [ ] `MergeServiceComponent`、Service 详情、服务列表有效配置、预览、计划哈希和 Compose 渲染对基础运行配置产生一致结果。
- [ ] 任一新增覆盖改变有效计划哈希，使 Service 显示等待显式部署；保存操作本身不触发部署。
- [ ] Web 服务组件详情展示并编辑 Version 默认值、当前 Service 值及覆盖来源，可逐字段重置，且不会丢失既有 overlay 组。
- [ ] HTTP API 和 MCP 使用同一请求/响应契约；MCP 结果包含新增覆盖字段。
- [ ] Service 切换 Version 时，新增覆盖与既有覆盖一起按映射组件校验；不兼容切换被拒绝，不产生部分写入。
- [ ] 新增迁移同时覆盖 SQLite 与 MySQL，已执行迁移不被修改。

## Open Questions

- Service 组件基础配置是否在一个分组中编辑四个字段，还是拆为“运行策略”和“启动配置”两个独立界面分组；两种方案的后端覆盖语义相同。本任务暂定为同一 Service Component overlay 请求内的两个 UI 分组。
- `restart_policy` 当前 Version 的有效值集合为 `no` 与 `unless-stopped`。本任务暂定 Service 覆盖严格复用该集合，不扩大为 Docker Compose 的其他策略。

## Decisions

- 用户确认 Service Component 不可覆盖镜像；镜像保持 Version 的制品与运行契约。
- 用户已确认启动命令、拉取策略和重启策略属于 Service Component 的可覆盖运行配置。
- 用户要求统一“重置”文案与行为：服务组件运行配置以及环境变量、资源、端点、挂载均重置为 Version 值；覆盖项不使用“删除”表示恢复继承。
- 用户确认 MySQL E2E 是环境依赖的非阻塞检查；详情沿用现有配置卡片的 Version 默认值、当前值和覆盖来源表达即可，无须独立展示服务端 effective 列。
- `entrypoint` 与 `command` 成对纳入范围，避免只覆盖其中之一而无法准确表达 Docker Compose 的启动语义。
- 组件身份与网络接口契约仍由 Version 管理；Service 只覆盖实例级运行行为。
- 覆盖字段直接定义在 `service_component`，与 `version_component` 对应字段同名；不建立独立的 runtime 子表或嵌套覆盖对象。

## Risk

- `entrypoint` 与 `command` 需要区分继承和显式空 argv；将空数组当作“继承”会导致无法恢复镜像默认启动行为。
- 此变更横跨数据库、SQLC、领域模型、有效计划、HTTP/MCP 协议和前端。前后端或 MCP schema 未同步时可能覆盖丢失或详情显示错误。

## User Review Notes

- 2026-08-12：用户确认 Service Component 覆盖范围包括启动命令、重启策略和拉取策略，并要求以标准模式记录任务；镜像不得覆盖。
- 2026-08-12：用户确认覆盖字段直接定义在 `service_component`，与 `version_component` 对应字段一致，并沿用现有 Service 覆盖的组织与处理逻辑。
