# 版本组件结构化表单
最后修改时间: 2026-07-28 18:30:43

Review status: Accepted

## Background

Version Component 当前将端口、环境变量、挂载、网络、依赖、健康检查、资源、tmpfs 和 ulimit 以 `*_json` 字段跨 API、模型、数据库和 Compose Renderer 传递。前端的表单或查看页一旦自行解析这些 JSON，只是在界面层掩盖未结构化的领域模型。

## Goal

- 将 Component 配置替换为结构化 Proto/API 模型和持久化结构，不保留 `*_json` 作为客户端或领域接口。
- 版本详情仅作为组件目录，提供新增、查看与进入组件独立详情页的入口。
- 组件详情页以表单管理端口、环境变量、挂载、健康检查、资源、tmpfs、ulimit、网络和依赖；已发布 Version 使用同一分组只读展示。
- Compose Renderer 直接消费结构化 Component 数据，不再解析 Component 配置 JSON。

## Non-goal

- 不改变 Component 仍从属于 Version 的数据关系。
- 不引入用户编辑 raw compose YAML。
- 不把运行时状态、日志等 Service/Container 职责移入 Component。
- 不保留旧 JSON Component API 或持久化模型的兼容读取、回填或双写层。

## User scenarios

- 用户在版本详情查看组件目录，进入某个组件的独立详情页。
- 用户在未发布 Version 的组件页中直接编辑所有 Component 配置，并一次保存。
- 用户以测试方式、命令、间隔、超时、重试次数、启动宽限期等字段编辑健康检查，不接触 JSON。
- 用户以行表单维护挂载、端口、环境变量、tmpfs、ulimit、网络和依赖，不接触 JSON。

## Acceptance

- `VersionComponentReq` / `VersionComponentResp` 不包含 `*_json` 字段，改为嵌套结构化消息和 repeated 项。
- 数据库存储不包含 Component 配置 JSON 列，改为一对一/一对多配置表；迁移后的数据可按 Component 关联。
- Compose Renderer、领域校验和 HTTP mapper 不再调用 Component 配置 JSON 解析器。
- 版本详情不再解析、汇总或展示组件内部 JSON 配置；每行组件可进入独立详情页。
- 组件详情页按基础信息、端口、环境变量、挂载、健康检查、资源和运行时配置分区展示；未发布状态为表单，发布状态为同布局只读。
- 空配置统一使用共享空状态和中性表述。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

组件继续作为 Version 内的规格单元保存，并用独立路由承载其查看和编辑体验。结构化模型以嵌套 Proto 消息表达、以组件关联表持久化；旧组件编辑模态窗和 JSON 字段一并移除。`000023_application` 的建表 SQL 原地更新，开发 SQLite 数据库通过 SQL 原地完成结构与数据转换，不在业务代码中执行迁移。

配置文本和值以用户输入或存储值为准：前端不通过 `trim`、默认值、静默过滤或自动规范化修正配置；字段缺失或不合法时显示校验错误。后端同样保留已验证的原值，组件改名等业务语义变更仅显式更新引用项。

## Risk

这项改造涉及破坏性 API 和数据库结构变更。开发库当前有 7 条组件记录，并实际使用了端口、环境变量、挂载、命令、依赖条件、健康检查、资源、tmpfs 和 ulimit；SQL 转换必须保留这些字段。健康检查、资源、网络和依赖的精确字段/表结构在规格阶段确定。
