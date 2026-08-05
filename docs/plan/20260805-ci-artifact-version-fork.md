# CI 制品关联应用版本计划
最后修改时间: 2026-08-05 17:37:49

Review status: Accepted

流程模式: 标准 / standard

## 输入与范围

- 依据已接受的 [Requirement](../requirement/20260805-ci-artifact-version-fork.md) 与 [Spec](../spec/20260805-ci-artifact-version-fork.md)。
- 仅实现 CI 制品记录、Application Version fork 和血缘查询；不触发 Service、Deployment、registry 或 MCP 的动作。
- `latest` 使用创建时 `id DESC` 的最后创建 Version；`fixed` 使用用户指定且属于 Application 的 Version。
- 同一 Repository 的触发与重试只断言不存在 `running` PipelineRun；`waiting_to_run` 不阻塞，不增加锁表、lease 或跨 worker 互斥。

## 实施步骤

1. 就地更新既有 SQLite 与 MySQL 的 `000022_pipeline_run`、`000023_application`、`000030_seed_pipeline` 及对应 down migration；不添加增量或兼容迁移，已有数据库另行迁移。
   - 建立 `pipeline_stage_build_version_binding`：以 `pipeline_stage_id` 唯一保存 `application_id`、`component_name`、`fork_strategy` 和可空的 `fixed_version_id`。
   - 建立 `pipeline_run_build_version_binding`：以 `(pipeline_run_id, pipeline_stage_id)` 唯一保存冻结后的 Application、组件、来源 Version，以及可空的 `generated_version_id` 和 `artifact_id`。
   - `artifact` 使用 `pipeline_stage_id`、`collector`、`location` 取代 `type`、`path`；直接保存 `value`、`value_format`、`image_ref`、`local_image_sha256` 与 `source_artifact_id`；`version_component.artifact_id` 关联 fork 后的 Component 与该 Artifact。
   - 为 Repository 的 `running` 查询和 Version 入向引用计数添加索引；对固定绑定、Run 来源/生成 Version 和 fork 血缘使用限制删除的外键。Component 制品关联可随 Component 删除。
   - 就地更新 seed Git clone Stage 为 `command`/`git_object_id` collector，并更新 Docker Stage 为 `docker_image`/`reference` collector；移除 `/artifacts/source_commit` 写入。

2. 扩展领域模型、SQL 查询、sqlc 适配器和 Repository 接口。
   - 在 `internal/model/pipeline.go` 与 pipeline DTO 中加入 `BuildVersionBinding`，并把它序列化进 `StageDefinition`，确保 Snapshot 包含不可变绑定。
   - 让 pipeline stage 的读取、创建、更新和复制同时读写绑定；更新 stage 内容或绑定时递增 stage version。
   - 在 PipelineRun/Artifact 模型中加入 Run 级绑定、镜像字段和 Component 制品来源的读模型；在 `sql/query/pipeline_run/`、`sql/query/application/` 增加写入、查询、引用计数和 `id DESC` 取最新 Version 的语句，并重新生成 sqlc。
   - 为 PipelineRunStore 增加“Repository 是否有 `running` Run”的查询，以及创建 Run 时持久化其绑定的接口。该接口是查询加业务校验，不实现锁。
   - 新增同一数据库事务内的持久化操作：插入完整 Artifact、fork Version 及其完整 Component 配置、更新目标 Component 镜像与 `artifact_id`，并回写 Run 绑定。使用现有 `tx.RunInTx` 上下文，使失败不留下部分数据库记录。

3. 在 Pipeline Stage 与 Template 编排处加入绑定和 DAG 校验。
   - 校验 Application 与 Stage 属于同一 Project，`component_name` 非空且符合现有 Component 命名约束；`latest` 不带 `fixed_version_id`，`fixed` 必须带 Version ID 且 Version 属于该 Application。
   - 有绑定的 Stage 必须且只能声明一个 `docker_image` collector；其 `image_ref` 来自该制品渲染后的 `reference`，不写入通用 Artifact 的 location。
   - 在 Template 编排保存、Snapshot 创建和 Run 创建三处验证：每个 `docker_image` Stage 的传递前置中恰有一个 `command`/`git_object_id` 制品生产 Stage。配置可被后续 Stage 使用，但并行的无依赖 Stage 不可作为来源。
   - collector 校验：`file` 和 `docker_image` 必须有 reference；`command` 必须有 command 和 `text`/`git_object_id` format；不以 `trigger_ref` 兜底。

4. 在触发和重试路径冻结 Version 来源并执行 Repository 运行任务断言。
   - 在 `TriggerRepository` 和 `RetryPipelineRun` 创建新 Run 前查询该 Repository 的 `running` Run；存在时返回明确业务校验错误，`waiting_to_run` 不参与判断。
   - 从 Snapshot 的绑定解析来源：`latest` 选择 Application 下 `id DESC` 的第一条 Version，`fixed` 重新验证 Version/Application/Component；缺少 Version 或组件时拒绝创建 Run。
   - 将来源 Version ID 与 Stage 绑定和 PipelineRun 一并写入。每次 retry 创建新的 Run，并按其创建时刻重新解析 `latest`；旧 Run 的来源记录保持不变。

5. 扩展执行器的制品归档与本地镜像读取。
   - `file` collector 沿用受声明路径约束的文件存在检查；`command` collector 通过 DockerRunner 在相同镜像、环境和挂载中捕获标准输出，直接持久化为 `artifact.value`。
   - 新增本地 Docker image inspect 端口及 `DockerRunner` 实现，针对渲染后的 `image_ref` 取得 Docker image ID（`sha256:` 标识）。tag 后续被覆盖不构成错误。
   - 在 Docker build Stage 脚本成功后，通过 Run/Stage/名称查询其唯一上游 `git_object_id` 值，再 inspect 本地镜像并写入 `source_artifact_id`。读取或 inspect 失败时将 Stage 标为失败，错误明确说明制品归档失败。
   - 有构建绑定时，在同一事务 fork 已冻结来源 Version，生成 `build-<runtime_datetime>` 展示 label，只替换指定 Component 的 `image`，保留其他 Component 与配置；记录新 Version、Component `artifact_id` 和 Artifact 关系。以 Run 绑定唯一键保证同一成功 Stage 不重复 fork。

6. 收紧 Version 删除及 Application 级级联删除的引用完整性。
   - 用统一的 Version 引用计数替代仅检查 Service/Deployment 的 `CountVersionRuntimeRefs`，涵盖 Service、Deployment、固定阶段绑定、Run 的来源/生成 Version 绑定和子 Version 的 `created_from_version_id`。
   - 删除 `ClearVersionForkRefs` 及 Application 删除路径中静默清空 fork 血缘的语义。存在任一引用即以业务错误拒绝删除，不删除或改写引用记录。
   - 调整 Application 删除的级联逻辑，使其不会绕过上述 Version 引用检查；存在受引用 Version 时拒绝整个 Application 删除。

7. 更新 API、传输层和 Web 视图。
   - 在 `proto/orbit/v1/pipeline/pipeline_stage.proto` 增加可选 `BuildVersionBinding` 请求/响应字段；在 `pipeline_run/artifact.proto` 与 `application/version.proto` 增加只读镜像元数据和 Component 制品来源字段。重新生成 Go 与 TypeScript Proto 客户端并更新 DTO、handler、HTTP 映射。
   - 在 Pipeline Stage 编辑页提供 Application、fork 策略、固定 Version 和 Component 选择；切换策略时清除不适用的固定 Version，并显示后端校验错误。
   - 在 Artifact 列表/Run 详情展示 `image_ref`、本地 image SHA 和 source commit；在 Application Version Component 详情展示其关联制品与来源 Version。label 只按展示处理。

## 预期文件范围

- `sql/migration/{sqlite,mysql}/000022_pipeline_run.{up,down}.sql`
- `sql/migration/{sqlite,mysql}/000023_application.{up,down}.sql`
- `sql/migration/{sqlite,mysql}/000030_seed_pipeline.up.sql`
- `sql/query/pipeline/pipeline_stage.sql`、`sql/query/pipeline/snapshot.sql`
- `sql/query/pipeline_run/{pipeline_run,artifact}.sql`、`sql/query/application/{application,version}.sql`
- `internal/model/{pipeline,pipeline_run,application}.go`
- `internal/repository/{pipeline,pipeline_run,application}.go` 与 `internal/repository/impl/sqlc/{pipeline,pipeline_run,application}/`
- `internal/application/pipeline/{dto,usecase}/`、`internal/application/pipeline_run/{dto,usecase,port}/`
- `internal/infrastructure/{runner/pipeline,storage/local/pipelineworkspace}/`
- `proto/orbit/v1/{pipeline/pipeline_stage.proto,pipeline_run/artifact.proto,application/version.proto}` 及生成产物
- 对应 HTTP handler/DTO 转换、`web/src/api/`、Pipeline Stage、Pipeline Run、Artifact 和 Application Version 的 Vue 视图与本地化文本
- 相关单元测试、SQLite/MySQL 集成测试与 seed 迁移断言

## 验证计划

1. 迁移验证：在 SQLite 与 MySQL 的全新数据库执行 up/down；确认外键、唯一键、索引和 seed clone 的 `source_commit` 声明正确。
2. Pipeline Stage/Template 测试：覆盖项目归属、`latest`/`fixed` 约束、单一 `docker_image`、固定 Version 所属校验、Snapshot 冻结与唯一传递 `source_commit` 生产者校验。
3. PipelineRun 测试：覆盖 `id DESC` 的 latest 选择、fixed 选择、无 Version/组件失败、retry 重新解析 latest，以及只有 `running` 状态阻止触发或重试；断言 `waiting_to_run` 不阻止且没有任何锁表/lease 行为。
4. Executor/Repository 测试：覆盖 command collector 的输出与 Git object ID 格式、Docker inspect 成功/失败、镜像 metadata 与 Artifact 自关联血缘保存、fork 后仅替换目标 Component、事务回滚和同一 Run/Stage 的幂等关联。
5. Version 测试：覆盖 Service、Deployment、固定绑定、Run 来源/生成绑定与子 Version 任一存在时删除拒绝；覆盖 Application 删除不会绕过该检查。
6. API/Web 验证：重新生成 Proto/sqlc 后执行 Go 单测、静态检查与前端 typecheck/lint；手工验证绑定配置、Artifact 追溯和 Version Component 来源展示。

## 风险与回滚

- 本地 image tag 可覆盖；界面与数据模型仅将其与记录的 `local_image_sha256` 并列展示，不把该 SHA 作为部署校验或 tag 覆盖拦截。
- 运行任务检查是用户选择的业务断言，不提供跨 worker 原子排他保证。
- 数据库 migration 下行只在尚未写入新关联数据的环境中可安全执行；已有制品或自动创建 Version 时，应先备份并按引用关系处理，不应通过删除引用来强制回滚。

## User review notes

1. 用户要求开始 Plan，前置 Requirement 与 Spec 视为已接受。
2. 用户确认只断言不存在 `running` 的业务任务，无须活动运行锁或同类实现。
3. 用户要求开始实现，Plan 视为已接受。
4. 用户要求不做旧制品模型兼容；基线 schema 与 seed 数据就地更新，已有数据库另行迁移，生成目录仅通过 Taskfile 再生。
