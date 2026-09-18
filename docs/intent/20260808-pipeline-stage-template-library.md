# 可复用流水线阶段
最后修改时间: 2026-08-08 16:39:12

Review status: Draft

流程模式: 严格 / strict

## Background

当前 `PipelineStage` 是 `Pipeline` 独占的 DAG 节点，阶段的创建、编辑和删除都嵌入在流水线详情中。用户无法预先集中管理可复用的执行阶段；每条流水线都需要重复填写镜像、脚本和说明。

此前将阶段从跨流水线共享资源改为 Pipeline 私有节点，是为了隔离 Application、Component 映射、来源 Version 策略、DAG 依赖和排序。恢复“全局共享 `PipelineStage`”不能把这些应用实例上下文写回可复用资源，但制品声明本身是构建阶段的可复用执行定义，适合由模板阶段保存。

本需求保留统一领域名 `PipelineStage`，通过 `kind=template|application` 区分两类阶段：项目级 `template` 阶段是可复用的通用执行定义，`application` 阶段是可执行 Pipeline 的私有节点。产品界面名称仍为“阶段”，阶段管理只管理 `kind=template` 的阶段。

Template Pipeline 编排的是模板阶段的引用及其 DAG/排序；Application Pipeline 由 Template Pipeline 实例化或直接引入模板阶段后，物化自己的 `kind=application` 阶段。模板阶段不直接保存 DAG、Application、Component、Component 绑定或 Version 策略，但保存完整制品声明。Application 与来源 Version 的选择、Docker 制品到 Component 的绑定在流水线实例化为 Application Pipeline 时完成。

## Goal

1. 提供项目范围的阶段管理界面，恢复与原构建模板一致的列表、详情、创建、编辑、复制和删除管理方式。
2. 允许 Template Pipeline 与 Application Pipeline 在编排时从阶段库搜索并引入模板阶段，而不是重复手工创建通用执行定义。
3. Template Pipeline 保存模板阶段引用和自身 DAG/排序；Application Pipeline 物化独立的 `PipelineStage(kind=application)`，且每个应用阶段必须记录非空的来源模板阶段 ID、名称和版本。
4. 继续让 Pipeline 编排承担 DAG 依赖和排序；模板阶段拥有制品声明，Application Pipeline 私有阶段拥有 Application/Component 制品绑定及来源 Version 策略。
5. 保持当前 Template/Application Pipeline、不可变 Snapshot、Run 执行和一次成功 Run 最多 fork 一个 Version 的模型不变。
6. 允许用户显式将模板阶段引用或已引入的应用阶段更新为指定模板阶段版本，并在更新前审阅差异和处理不再有效的私有配置。

## Non-goal

1. 不让 `PipelineStage(kind=template)` 保存或共享多条 Pipeline 的 DAG、排序、Application、Component、来源 Version 策略或运行时绑定；制品声明除外。
2. 不做模板阶段更新对 Pipeline 引用或 Application 阶段的自动同步、静默覆盖或自动重编排。
3. 不改变 Pipeline 的手动 Trigger、Retry、变量解析、Snapshot 创建或 CI 成功后的 Version fork 行为。
4. 不修改已执行迁移文件，不在业务代码中保留旧/新 Pipeline 模型兼容分支。
5. 不在本需求中实现跨项目或平台全局的模板阶段共享。
6. 不在业务逻辑中转换、容忍或适配既有无来源 `PipelineStage` 数据；主业务完成后由独立离线脚本处理旧数据，不属于本功能的业务实现。

## User scenarios

1. 项目成员进入“阶段”管理页，创建一个 `kind=template` 的镜像构建阶段，配置名称、执行镜像、脚本、制品声明和说明；创建后可进入独立详情页管理完整定义。
2. 项目成员在阶段列表中搜索、查看详情、编辑、复制或删除模板阶段；删除模板阶段不会删除已物化到 Application Pipeline 的私有阶段，也不会影响历史执行记录。
3. 项目成员编辑 Template Pipeline 时，从本项目阶段库选择一个模板阶段并引入。系统创建独立的编排引用，用户再配置该引用的排序和依赖关系。
4. 项目成员从 Template Pipeline 创建 Application Pipeline 时，系统将模板阶段引用物化为新的 `kind=application` 私有 Stage ID，并保留来源模板阶段的 ID、名称和版本。
5. 项目成员在模板阶段详情中管理 `file`、`command` 或 `docker_image` 制品声明。直接引入和模板流水线实例化均复制该声明；实例化为 Application Pipeline 时，用户选择 Application、来源 Version 策略，并将每个 `docker_image` 声明绑定至目标 Component。
6. 同一模板阶段可被同一 Pipeline 多次引入。每次编排引用和应用私有阶段都有不同 ID，用户自行以私有阶段名称区分。
7. 模板阶段被更新后，直接新引入的阶段使用更新后的定义；Template Pipeline 实例化时使用其阶段引用中已保存的定义。已存在的 Pipeline 引用、Application Stage 和已有 PipelineSnapshot 保持原记录，用户不会在未确认的情况下改变运行结果。
8. 当模板阶段有更高版本时，用户在阶段编辑器右上角按钮组看到警告色“更新至模板 vN”命令。点击后系统展示名称、镜像、脚本、制品声明和说明的差异；用户确认“更新并保存”后替换通用定义，保留 Pipeline 私有名称、DAG、排序和组件绑定。

## Acceptance

1. 系统存在项目归属的 `PipelineStage(kind=template)`，具有名称、镜像、脚本、说明与版本等完整定义；同项目内名称唯一，所有操作执行项目成员校验。
2. Web 在 CI 导航内提供“阶段”入口，支持仅 `kind=template` 阶段的列表、搜索、创建、详情、编辑、复制、删除和字段级表单校验。详情页将基本信息、脚本、制品声明和危险操作分区呈现，使用项目共享表单与选择组件。
3. 模板阶段保存并校验 `file`、`command`、`docker_image` 制品声明；声明不包含 `component_name`。DAG 依赖、排序、Application 绑定、Component 绑定和 Version fork 策略不得持久化到 `kind=template` 阶段。
4. Template Pipeline 的编排创建模板阶段引用，并在引用中保存导入时的执行定义和制品声明、DAG 与排序；Application Pipeline 的 `kind=application` Stage 由该引用复制为独立可执行定义。关联表或模型的具体命名由 Spec 确定，但不得让 DAG/排序写入模板阶段本身。
5. Pipeline 编辑页以“引入阶段”替代重复填写通用执行定义；用户能选择本项目 `kind=template` 阶段，并在引入后编辑该 Pipeline 的依赖、排序和说明。
6. 直接引入必须由服务端读取模板阶段完成；Application Pipeline 创建时必须由服务端读取 Template Pipeline 已关联的阶段引用完成。每次应用私有 Stage 创建、模板版本更新与 Pipeline 版本递增在同一写事务中完成；客户端不能伪造模板定义或跨项目模板 ID。
7. Template Pipeline 和 Application Pipeline 都可以引入模板阶段。Application Pipeline 的 Application 在实例化时选择，并在同一实例化流程选择来源 Version 策略、将 `docker_image` 制品绑定到该 Application 的 Component；现有唯一上游 `git_object_id`、组件唯一性和来源 Version 策略约束继续生效。
8. 每个 Application Pipeline 私有 Stage 必须保存非空的来源模板阶段 ID、名称与已应用版本；创建、引入、更新和 Snapshot 读取路径拒绝无来源 Stage。既有无来源数据不由业务逻辑兼容，按离线迁移或重建策略在切换前处理。
9. 模板阶段的更新、删除或物理不存在不得改变已物化的 Application Stage、PipelineSnapshot、PipelineRun、Artifact 或生成 Version 的执行/展示历史。
10. Pipeline 编辑页提供显式的模板阶段更新操作：仅在来源模板阶段仍存在且版本高于已应用版本时，于阶段编辑器右上角显示警告色“更新至模板 vN”。更新前显示名称、镜像、脚本、制品声明和说明的差异，确认对话框使用“更新并保存”；未确认不改变 Pipeline，确认成功后递增 Pipeline 版本并使后续 Run 创建新 Snapshot。该操作不得覆盖私有 Stage 名称、DAG、排序或 Component 制品绑定。
11. 新增 API、Proto、Go 与 TypeScript 生成物、路由、导航、模型、Repository、Usecase、迁移、文档和测试均只表达该新模型，不增加旧接口别名或运行时降级路径。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 代码领域名统一使用 `PipelineStage`，通过 `kind=template|application` 区分阶段类型；界面名称使用“阶段”，阶段管理只管理 `kind=template`。
2. `PipelineStage(kind=template)` 保存可复用的执行定义及制品声明。Template Pipeline 的引用和 DAG、排序，以及 Application Stage 的 Application/Component 绑定和来源 Version 策略均不属于模板阶段。
3. Application Pipeline 使用创建时复制的阶段定义执行，而不是运行时读取模板阶段。现有 `PipelineSnapshot` 继续记录每次运行使用的输入，模板阶段变更不会隐式改变 Pipeline 或既有 Snapshot。
4. 每个 Application Stage 必须有模板阶段来源；来源 ID、名称和已应用版本是非空逻辑引用及展示字段，不使用会阻断模板删除的物理外键。Template Pipeline 阶段引用保存可用于创建 Application Pipeline 的执行定义，模板删除后仍可使用。无来源旧数据不由业务逻辑处理。
5. 模板阶段保存 `file`、`command`、`docker_image` 制品声明；阶段引用与应用阶段在导入或创建时复制该声明。Docker 制品的目标 Component 映射仅属于 Application Pipeline，并在流水线实例化时配置。
6. 模板阶段仅在项目内共享。跨项目的标准阶段库需要组织级权限、所有者与凭据/变量边界，作为独立需求评估。
7. 同一模板阶段可被同一 Pipeline 引入多次；每次创建独立编排引用和 Application Stage，用户自行保证私有 Stage 名称可辨识且满足 Pipeline 内唯一性。
8. 已引入阶段更新模板版本必须是显式、可审阅的操作，不允许自动同步；确认更新后使用更新后的通用定义并重新校验 Pipeline。
9. 模板有更新时，在阶段编辑器右上角显示警告色“更新至模板 vN”命令；普通本地保存不得触发模板更新。

## Risk

1. 把依赖、排序、Application 或 Component 映射放入 `PipelineStage(kind=template)` 会复发跨 Pipeline 配置污染，且可能使模板阶段无法通用；实现必须在模型、HTTP 契约和前端表单三层拒绝这些字段。制品声明是模板阶段的受支持定义。
2. 若引入只在前端复制字段，客户端可绕过模板归属和版本审计，也可能产生非原子 Pipeline 版本更新；复制和校验必须由后端事务承担。
3. 当前活文档声明的 Pipeline 迁移版本与仓库实际迁移目录存在不一致；在创建新迁移、更新测试基线和生成 SQLC 前必须先校准当前迁移链，不得回改既有迁移。
4. 动态制品、依赖关系和 Application 条件字段较多。前端必须将本地错误定位到具体控件，切换制品类型、删除制品或取消组件映射时清理已失效错误，避免将字段错误只显示为 Toast。
5. 模板版本更新只能影响通用定义。实现必须保留 Application Stage 的本地名称、DAG、排序和组件映射，不能通过覆盖或清除私有配置完成更新。
6. 强制来源是破坏性切换。旧数据由主业务完成后的独立脚本处理；业务接口必须拒绝无来源数据，不得用空来源、默认模板或旧阶段编辑入口降级运行。

## User review notes

1. 用户明确要求新增可复用的模板阶段，界面名称仍叫“阶段”，并在 Pipeline 编辑过程中引入它们。
2. 用户确认同一模板阶段可以在同一 Pipeline 中重复引入，由用户自行区分。
3. 用户确认每个 Application Stage 必须有模板阶段来源；既有无来源数据不在业务逻辑中迁移或兼容。
4. 用户确认需要显式将已引入阶段更新至模板版本的能力，并要求在模板有更新时于阶段编辑器右上角使用警告色按钮显式执行更新。
5. 用户要求统一使用 `PipelineStage(kind=template|application)`，阶段管理只管理 `kind=template`，不新增独立的 `PipelineStageTemplate` 领域名。
6. 用户确认制品声明应保存在模板阶段；技术上不存在阻碍。声明不含 Application/Component 绑定，并在导入和实例化时复制到对应对象。
7. 用户确认 Application 关联在模板流水线实例化为 Application Pipeline 时完成；因此目标 Component 和来源 Version 策略也只能在该实例化流程确定。
8. 用户确认 Template Pipeline 创建 Application Pipeline 时使用其关联阶段引用中保存的数据，不重新读取阶段库当前定义；阶段模板删除后，该引用仍支持继续创建。
9. 用户确认旧数据在主业务完成后由独立脚本处理，业务代码不迁移、不兼容。
10. 用户要求 `PipelineStage` 的模板/应用字段形态由业务层校验，不使用数据库 `CHECK` 约束表达该领域规则。
