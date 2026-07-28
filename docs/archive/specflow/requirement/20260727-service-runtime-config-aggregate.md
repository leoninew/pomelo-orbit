# Service 聚合拥有运行时配置
最后修改时间: 2026-07-27 14:01:00

Review status: Accepted

## Flow mode

标准模式 / `standard`

## Background

现有实现将 `runtime_env` 作为 `Credential` 的一种类型保存；Version Component 通过 `secret_env_refs` 保存 `env_key`、`credential_id` 与 `data_key`，Service 详情和部署时再解密 Credential 并动态解析引用。

这与现行领域模型冲突：Service 是运行态 SoT，代表一个 Application 在目标环境和实例上的运行绑定；运行时配置应是 Service 聚合的属性。Credential 与持续部署无关，不是 Service 配置的录入、引用、解析或部署来源。

当前模型使修改一个 Credential 可在不修改 Service 的情况下改变详情接口和下一次部署的结果，同时运行中容器仍保持旧值。这绕过了 Service 聚合，也使“服务配置”“待部署配置”和“已运行配置”无法清晰区分。

## Goal

1. 将运行时配置定义为 Service 聚合内的普通 `key/value` 集合，由 Service 成为部署所需运行时配置的唯一业务事实来源。
2. 将 Credential 完全移出持续部署链路；Credential 的创建、修改、删除和轮换均不得影响任何 Service 配置或部署行为。
3. 让 Service 配置的创建、修改、查看和部署均经过明确的 Service 用例；修改仅改变待部署配置，直到部署完成才影响容器。
4. 让 Service 详情准确表达已保存的 Service 配置与运行中容器环境之间的关系，不把其中任何一方误称为另一方。

## Non-goal

1. 本任务不从 Docker 容器或 `docker inspect` 读取进程实际环境变量，也不尝试将容器环境作为持久化 SoT。
2. 本任务不改变非 `runtime_env` 类型 Credential 的职责、存储格式或使用路径。
3. 本任务不将配置值持久化到 Version；Deployment 按需求保存配置快照供 worker 使用。配置不是 Credential 或秘密，不施加加密、脱敏或额外的秘密访问控制。
4. 本任务不保留旧的 `Credential -> VersionComponent.secret_env_refs -> Service` 运行时配置链路作为兼容层；迁移后只保留新的 Service 聚合路径。
5. 本任务不扩展 Docker、MCP、Gateway、Expose 或 Application 生命周期的产品范围。

## User scenarios

1. 运维人员创建或编辑 Service 时，以 `key/value` 维护运行时配置。保存后，该值成为该 Service 的待部署配置；其他 Service 和任何 Credential 编辑均不受影响。
2. 运维人员可在项目权限范围内查看 Service 当前保存的运行时配置，并明确知道该页面不是容器环境实时快照。
3. 运维人员通过 Service 配置工作流直接维护运行时环境变量；该工作流不列出、读取或导入 Credential。
4. 运维人员发起部署或重启时，系统立即将当前 Service 配置写入 Deployment 快照；worker 后续只使用该快照，即使 Service 配置已被修改。
5. Version 的环境变量占位符发生修改时，系统校验所有关联 Service 的配置：缺少所需 key 时拒绝修改并说明原因；多余 key 保留但忽略，并打印警告。

## Acceptance

1. Service 聚合持久化普通 `key/value` 运行时配置；持久化模型不依赖 `credential_id`、`data_key` 或组件级秘密引用才能解析部署值。
2. 配置以普通明文 `key/value` 保存和返回，不使用 Credential、加密、脱敏或秘密专用控制；既有 Service 项目访问规则继续适用。
3. Credential 的创建、修改、删除和轮换不会改变任何既有 Service 的运行时配置或下一次部署结果；持续部署的 API、用例、数据模型和页面不读取、列出或导入 Credential。
4. 移除 `runtime_env` Credential 类型与 `VersionComponent.secret_env_refs` 运行时引用模型；Version 仅保留静态组件规格。Service 配置与目标 Version 不兼容时，创建、更新或部署必须失败并给出可理解的错误。
5. 发起部署或重启时，将完整 Service 配置写入 Deployment 快照；Compose Render 和 worker 只从该快照解析 Version 的 `${KEY}` / `${KEY:-default}` 占位符。快照后 Service 配置的修改不能影响该任务。
6. Service 详情 API 和页面展示当前保存的 Service 配置，不展示或暗示它是当前容器的实际环境；接口契约不暴露 Credential、`data_key` 或组件级秘密引用。
7. 现有数据有明确、可重复的迁移/清理路径，且迁移完成后旧的运行时环境引用链路不再参与读写或部署。
8. 后端、HTTP/Proto 契约和 Web 的相关测试覆盖：Service 配置隔离、普通 K/V 读写、Credential 与持续部署的完全隔离、部署快照不受后续修改影响、Version 匹配拒绝和多余 key 警告、部署物化来源。
9. MCP 与受管 RAGFlow 初始化流程同样通过 Service 创建、配置和 `service_id` 部署，不得提供或调用 `runtime_env` Credential、`secret_env_refs` 或调用方直传 K/V 的部署接口。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 运行时配置属于 Service 聚合，是部署运行态的业务 SoT；Version 仅保存静态组件规格。
2. Credential 与持续部署完全解耦，不作为 Service 配置的录入、引用、解析或部署来源。
3. 不建立新旧运行时配置链路并行的兼容层；迁移以一次性、可验证的方式完成。
4. 服务详情的运行时配置视图是 Service 持久化配置读模型，不是 Docker 容器环境快照。
5. 配置可独立于部署保存；保存后仅在下一次成功部署或重启后影响运行中容器。
6. Deployment 在创建时快照完整 Service 配置，worker 不读取任务创建后的 Service 配置。
7. Version 修改必须匹配关联 Service 的必需占位符；缺 key 拒绝，多余 key 忽略并打印警告。
8. 运行时配置是普通 `key/value`，不是秘密；不需要加密、脱敏或 Credential 专用控制。

## Risk

1. 这会触及 Service、Version Component、Credential、Deployment、Compose Render、数据库迁移、Proto 和 Web 多个边界，必须先设计迁移顺序和部署一致性，避免半迁移状态执行部署。
2. Deployment 快照包含完整 K/V 配置；持久化和读取模型必须使 worker 能稳定使用该快照，但不应将其误解为运行中容器的实时值。
3. Version 修改需要检查所有关联 Service；查询与错误汇总必须在 Service 数量较多时保持可控，并在多余 key 情况记录明确警告。
4. 用户已明确授权本任务直接修改既有建表迁移并同步当前运行中的 SQLite 库；旧表和数据以事务方式清理，新库与当前运行库均得到一致结构。

## User review notes

用户于 2026-07-27 明确指出：原 Credential 表没有被预期作为运行时配置 SoT 使用；从 DDD 看 Service 是聚合，运行时配置是 Service 的众多属性之一，现有实现不正确。随后进一步重申：Credential 与持续部署没有关系。

用户确认：运行时配置可修改但到部署后才生效；Deployment 创建时快照配置；Version 修改必须匹配关联配置，缺失必需 key 时拒绝、多余 key 忽略并打印警告；配置仅是普通 `key/value`，不需要加密或秘密控制。

用户于 2026-07-27 明确授权：涉及表结构时无需新增增量迁移，直接修改既有建表迁移，并同步更新运行中的 SQLite 数据库。
