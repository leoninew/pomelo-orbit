# 流水线模板与应用流水线分离
最后修改时间: 2026-08-07 18:55:18

Review status: Accepted

流程模式: 严格 / strict

## Background

当前 `PipelineStage` 同时承担可复用执行步骤和 Application Version fork 配置：`BuildVersionBinding` 直接绑定 Application、目标 Component 与 fork 策略。一个 Stage 被多个 PipelineTemplate 编排或被模板复制后，所有引用都会共享该 Application 绑定。

这与通用镜像构建模板的定位冲突。用户需要从通用模板创建面向单一 Application 的流水线，并在创建或配置该流水线时完成一次 Application 绑定；随后每次运行不再选择或修改 Application，也不需要重新调整通用构建 Stage。

当前手动触发以 `repository_id + template_id` 为输入。Webhook 也直接关联 `repository_id + template_id`。二者没有独立的、已绑定 Application 的流水线实体。

## Goal

1. 明确区分不可直接承担 Application 绑定的 PipelineTemplate 与可执行的非模板 Pipeline。
2. PipelineTemplate 作为可复用编排与通用构建 Stage 的来源；其 Stage 不持久化 Application Version 绑定。
3. 非模板 Pipeline 在同一 Project 内固定绑定一个 Repository；Application 可选，只有制品需要进入 Application Version 时才绑定 Application 并配置 `latest`/`fixed` 来源 Version 策略。
4. `docker_image` 制品声明决定其是否绑定目标 Component；运行时不再要求或允许选择 Application、Repository 或 Component 映射。
5. 保留当前镜像制品、来源 commit、Run 级来源 Version 冻结、Version fork 与 Component 制品血缘的能力；一次成功 Run 最多生成一个 Application Version。
6. 手动触发与 Retry 使用同一 Run 创建逻辑，确保来源 Version 与所有制品绑定完整写入。
7. Template 保留独立版本；Application Pipeline 沿用现有 Snapshot、Run 与 Artifact 历史链路，Template 本身不再创建 Snapshot。
8. 复用既有运行时变量解析与覆盖优先级：Application Pipeline 详情先展示已解析的默认运行时值，运行对话框仅按当前 ref 和用户覆盖重新计算，不产生 Run 或 Snapshot。

## Non-goal

1. 不在 CI 成功后自动部署 Service 或 Version。
2. 不引入 registry、OCI digest 或本地镜像 tag 覆盖拦截。
3. 不改变 Application Version fork 的未发布状态和现有制品追溯字段。
4. 不修改已执行的迁移文件，也不通过兼容层同时保留两套业务模型。
5. 不在本轮迁移或实现 PipelineWebhook、Webhook 配置和公开 Webhook 接收入口。
6. 不增加归档、停用等 Pipeline 状态；Pipeline 使用物理删除。

## User scenarios

1. 用户创建通用镜像构建模板，包含 clone、build 等 Stage 和制品声明，但不选择 Application。
2. 用户从该模板创建一个非模板 Pipeline，必须选择 Repository，可按是否需要生成 Application Version 选择 Application；仅在配置目标 Component 时选择 `latest` 或 `fixed` 来源 Version 策略。
3. 用户在 Application Pipeline 的 `docker_image` 制品声明中为需要进入应用版本的镜像选择目标 Component；其他制品不参与 Version 生成。
4. 用户运行非模板 Pipeline 时，只提供 ref 和变量；系统从 Pipeline 已绑定的 Repository 解析源码并冻结到 Run。存在目标 Component 映射时，系统再从绑定的 Application 解析来源 Version，并在成功后将全部已绑定制品一次性写入一个 fork 出的新 Version。
5. 多个非模板 Pipeline 可从同一模板创建，分别绑定不同 Application；修改其中一个 Pipeline 的制品 Component 映射不会影响模板或其他 Pipeline。
6. Template 更新到新版本后，后续创建的 Pipeline 记录新的来源版本；既有 Application Pipeline 不受影响，并在其自身版本变化时创建新的 Snapshot。
7. 用户打开 Application Pipeline 详情时，变量声明表直接展示由绑定 Repository、Pipeline 声明和既有覆盖规则解析出的默认值（例如 `repository_dockerfile`）。点击运行后，用户修改 ref 或可覆盖变量，运行对话框重新展示同一解析规则下的值；两次预览都不创建 Snapshot、Run、任务或 Version 绑定。

## Acceptance

1. 通用 Stage、PipelineTemplate 的读写模型和 UI 中不再出现 Application Version 绑定。
2. 非模板 Pipeline 必须有同 Project 的 Repository；Application 是可选的，但 Component 绑定制品必须有同 Project 的 Application。模板不得直接运行。
3. 每个需要进入 Application Version 的 `docker_image` 制品声明恰好配置一个目标 Component；同一 Pipeline 内目标 Component 不重复，来源 Version 策略由 Pipeline 统一配置。
4. PipelineRun 和 Artifact 历史记录能展示已冻结的 Application、来源 Version、生成 Version、Component、镜像制品与 source commit。
5. 手动触发与 Retry 都通过同一绑定解析与 Run 持久化路径；一个成功 Run 最多生成一个包含全部已绑定制品的未发布 Version。
6. 从同一模板派生的两个 Pipeline 可独立绑定不同 Application，且一次运行不需要手动改写通用 Stage 或制品声明中的 Application 身份。
7. Application Pipeline 的 Run 必须引用同版本的不可变 PipelineSnapshot；Template 不得拥有或生成 Snapshot。
8. Pipeline 详情和运行对话框都通过同一无副作用的变量预览路径展示运行时值，并保留 Repository 覆盖、Pipeline 自定义值与用户本次覆盖的既有优先级。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. Repository 身份从可复用 Stage 移至非模板 Pipeline；Application 身份也可在需要版本制品绑定时固定到 Pipeline。来源 Version 策略属于 Pipeline，目标 Component 映射属于 `docker_image` 制品声明。
2. 从模板创建 Pipeline 时物化完整流水线定义；模板后续修改不隐式影响已创建 Pipeline。
3. Run 级 Version 绑定继续保存实际 Application 和来源/生成 Version；新 Version Component 通过 `artifact_id` 与对应制品关联，作为不可变执行历史。
4. 业务代码不保留旧流水线模型的兼容分支。空库迁移链在 version 30 直接创建最终结构和种子数据，不提供旧结构就地转换。
5. Template 与 Application Pipeline 各自维护 `version`。Application Pipeline 创建时记录不可变的 `source_template_version`；只有 Application Pipeline 的版本驱动 Snapshot 创建与复用。
6. Pipeline 允许物理删除，不增加生命周期状态；既有数据库数据的处置属于后续独立迁移任务。
7. Webhook 尚未投入使用，本轮不迁移、不重建也不实现其新模型。
8. 跨生命周期的展示和历史引用使用逻辑外键：保存目标 ID 以及必要名称/版本标签，不用数据库外键阻断 Template、Application、Repository、Version 或 Pipeline 的物理删除。只有同一聚合内的组成关系使用物理外键。
9. 运行时变量值的展示与运行前校验共用一个预览解析入口；详情使用默认 ref 和空覆盖，运行对话框以当前 ref 与覆盖重算。预览严格只读，不创建 Snapshot、Run、任务或 Version 绑定。

## Risk

1. 若在制品归档时为每个绑定制品分别 fork Version，多组件构建会生成互相覆盖的零散 Version；实现必须在整个 Run 成功后只 fork 一次并原子更新所有目标 Component。
2. 手动触发与 Retry 若各自维护 Run 创建逻辑，会再次出现来源 Version 冻结不一致；改造必须收敛入口。
3. 开发数据库的就地迁移脚本必须在业务代码切换前完成；脚本失败或未执行时，系统不得试图通过旧模型降级运行。
4. 变量预览依赖绑定 Repository 的可访问配置；前端必须展示加载或失败状态并禁止在预览失败时提交运行，不能以空白值伪装成已解析结果。
