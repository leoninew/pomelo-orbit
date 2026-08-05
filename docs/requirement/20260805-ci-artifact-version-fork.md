# CI 制品关联应用版本
最后修改时间: 2026-08-05 09:44:39

Review status: Accepted

流程模式: 标准 / standard

## Background

Pomelo Orbit 的 CI 已能声明并记录 `docker_image` 制品，但当前制品只有通用的 `type`、`name` 和 `path` 字段，镜像标签、构建所得本地镜像 SHA 与源码 commit SHA 没有结构化记录。

Application 包含多个 Version，Version 由 Component 镜像及其运行配置组成。现有 `ForkVersion` 可从任意已有 Version 复制出未发布 Version，并保留 `created_from_version_id` 血缘；Application 本身没有 current、default 或 base Version 概念。Version 的 label 不是语义化版本，也不能作为关联标识。

本需求面向同一宿主机上的本地构建与运行。镜像 tag 可以被后续构建覆盖；记录的 SHA 用于解释创建 Version 时对应的构建产物，不用于限制后续部署或阻止 tag 覆盖。

## Goal

1. 规范化记录构建产生的本地容器镜像制品，至少可追溯镜像 tag、本地镜像 SHA 和实际源码 commit SHA。
2. 允许构建阶段关联一个 Application，并配置从哪个已有 Version fork 新的未发布 Version。
3. 在关联构建阶段生成镜像制品后，基于已解析的来源 Version 创建未发布 Version，并建立制品与新 Version 的关联。
4. 提供以下两种 fork 策略：
   - `latest`：从该 Application 最后创建的 Version fork；Application 至少需要一个 Version。
   - `fixed`：从用户选择的、属于该 Application 的已有 Version fork。

## Non-goal

1. 不引入或要求语义化版本（Semantic Versioning）。
2. 不将 Version label 用作制品关联、fork 选择或版本排序的依据。
3. 不要求推送镜像到 registry，不要求 OCI manifest digest，也不对本地 tag 覆盖做部署前校验。
4. 不改变 Service、Deployment、MCP 的职责或生命周期，也不在 CI 成功后自动部署。
5. 不在本需求中定义发布门禁；新建 Version 的状态为未发布，但现有部署准入语义不在范围内。

## User scenarios

1. 用户为一个构建阶段选择 Application 和 `latest` 策略。运行创建时应用已有 Version；构建成功后，系统从本次运行已确定的来源 Version fork 出未发布 Version，并关联该镜像制品。
2. 用户为一个构建阶段选择 Application 和 `fixed` 策略，并选择该 Application 下的一个 Version。构建成功后，系统始终从该指定 Version fork，不受后续其他 Version 创建影响。
3. 用户查看新 Version 时，可以查看其来源 Version、关联的镜像制品、本地镜像 SHA 与 commit SHA；Version label 仅作展示。

## Acceptance

1. 构建阶段可以保存 Application 关联和 fork 策略；`fixed` 策略保存的 Version 必须存在且属于该 Application。
2. 镜像制品以结构化字段保存本地镜像 tag、SHA 和实际 commit SHA，不再将这些语义混入通用 `path`。
3. `latest` 策略选择该 Application 最后创建的 Version；Application 不存在 Version 时返回明确错误，且其选择的来源 Version 在本次运行内保持确定。
4. 关联构建阶段成功生成镜像制品后，会创建一个未发布 Version，且该 Version 可追溯到其 fork 来源与镜像制品。
5. 关联关系使用资源 ID，不依赖 Version label 或语义化版本格式。
6. 本需求不会增加自动部署、registry 依赖或本地 tag 覆盖拦截行为。

## Open questions

1. `latest` 在 PipelineRun 创建、阶段开始还是制品生成时解析，以及创建时间相同时的稳定排序规则，待在 Plan 阶段确定。
2. 镜像制品如何映射到 fork 后 Version 的具体 Component：由阶段配置组件名、由制品声明提供映射，还是只支持单组件 Application，尚未确定。
3. Version 引用的完整枚举、删除错误文案和数据库约束方式待在 Plan 阶段确定。
4. 同一阶段生成多个镜像制品或任务重试时，是否复用同一个新 Version、如何确保不重复 fork，尚未确定。
5. 自动生成的未发布 Version label 格式尚未确定；该格式不应被赋予语义化版本或关联键含义。

## Decisions

1. 制品关联和 Version 血缘使用资源 ID；不等待语义化版本设计。
2. 镜像 tag 覆盖属于允许的本地构建行为。保存镜像 SHA 与 commit SHA 的目的为构建来源追溯，不是冻结部署输入。
3. Application 不增加 base Version 或 current Version 字段。
4. fork 策略限定为 `latest` 和 `fixed` 两种；`latest` 要求应用已有 Version，`fixed` 要求用户选择已有 Version。
5. 新 Version 通过现有 fork 语义创建为未发布 Version；本需求不扩展为部署工作流。
6. 删除 Version 前必须断言不存在任何引用；引用包括既有 Service、Deployment 和本需求新增的固定阶段关联、制品关联及 fork 血缘，不再静默断开引用。

## Risk

1. `latest` 策略在并发构建下可能选择与用户预期不同的最后创建 Version，必须定义解析时点并在运行记录中保存实际来源 ID。
2. 本地 tag 可覆盖，因此 Version 的镜像引用不能单独证明任意一次后续运行实际使用的镜像；制品记录仅证明 Version 创建时的构建来源。
3. 固定 Version 引用和多制品/重试场景需要明确引用完整性与幂等性，否则可能出现无法删除的 Version 或重复生成的未发布 Version。

## User review notes

1. 用户确认本地构建、本地运行的场景不需要 registry；commit SHA 的记录具有自然的追溯价值。
2. 用户确认不需要因本地 tag 覆盖而阻止部署；记录镜像 SHA 已足以说明构建来源。
3. 用户强调当前讨论仅覆盖制品与 Version 的关联，不包含部署行为。
4. 用户确认 `latest` 使用 Application 最后创建的 Version 作为 fork 来源。
5. 用户确认删除 Version 时必须断言没有被引用。
