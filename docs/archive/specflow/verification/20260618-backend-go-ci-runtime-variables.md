# backend-go CI 运行时变量修复验证
最后修改时间: 2026-06-18 15:55:35

Review status: Draft

## Requirement alignment

按 `docs/requirement/20260618-backend-go-ci-runtime-variables.md` 核对：

- 已对齐手动触发、webhook 触发、retry 路径的变量构建入口：创建 `PipelineRun` 时构建完整变量快照，worker/task payload 不再作为变量权威来源。
- 已实现 runtime variable 合并链路：repository builtins、template builtins、runtime overrides、repository overrides、template declarations、stage declarations。
- 已实现 `value` 优先、无 `value` 时使用 `default`。
- 已阻止 runtime/repository/template/stage 来源覆盖系统内置变量。
- 已将新 run 的 `VariablesSnapshot` 写为完整 `VariableDeclaration[]`，并保留旧 map 格式读取能力。
- 已实现 secret 变量存储快照时脱敏为 `***`；执行阶段不把脱敏值当作真实执行值。
- 已按用户确认就地修改 backend-go 初始化迁移脚本，使“通用容器镜像流水线”声明 `working_dir` 与 `repository_dockerfile` 默认值。
- 已同步 frontend `TriggerModal` 使用 `value ?? default` 初始化变量输入与 placeholder。

## Spec alignment

不适用。当前流程是轻量模式 / light，本需求未创建独立 spec 文档，验证按 requirement / 需求核对。

## Plan alignment

不适用。当前流程是轻量模式 / light，本需求未创建独立 plan 文档，验证按 requirement / 需求核对。

## Actual diff summary

实际改动文件：

- `backend-go/internal/service/ci/runtime_variables.go`
  - 新增 backend-go CI runtime variable 合并、声明补全、校验、secret 脱敏、旧格式变量快照兼容逻辑。
- `backend-go/internal/service/ci/runtime_variables_test.go`
  - 新增默认值合并、内置变量不可覆盖、repository override 优先级、secret 脱敏测试。
- `backend-go/internal/service/ci/pipeline_run.go`
  - 手动触发与 retry 创建 run 时构建完整变量快照，task payload 仅传 `pipeline_run_id`。
- `backend-go/internal/service/ci/repository.go`
  - webhook 触发创建 run 时构建完整变量快照，task payload 仅传 `pipeline_run_id`；执行 store 接口补充 `PipelineTemplate`。
- `backend-go/internal/service/ci/execution.go`
  - 执行阶段基于 repo/template/snapshot/run 重新解析变量，不再信任 payload variables。
- `backend-go/internal/service/ci/execution_test.go`
  - 新增执行阶段从变量声明默认值解析脚本的测试；更新 fake store 支持读取 template。
- `backend-go/internal/migrations/mysql/v0.1.4__business_data.sql`
  - “通用容器镜像流水线”增加 `working_dir` 与 `repository_dockerfile` 变量声明默认值。
- `backend-go/internal/migrations/sqlite/v0.1.4__business_data.sql`
  - 与 MySQL 初始化数据保持一致。
- `frontend/src/views/ci/components/TriggerModal.vue`
  - 变量初始化、placeholder、内置变量展示 fallback 使用 `value ?? default`。
- `docs/requirement/20260618-backend-go-ci-runtime-variables.md`
  - 记录并接受本次轻量模式需求。

## Expected vs actual changed files

| 预期范围 | 实际结果 | 结论 |
| --- | --- | --- |
| backend-go CI runtime variable 合并逻辑 | 新增 `runtime_variables.go` 与测试，触发/执行路径接入 | 符合 |
| backend-go 手动触发、webhook、retry、执行路径 | 修改 `pipeline_run.go`、`repository.go`、`execution.go` | 符合 |
| backend-go 单元测试 | 新增 `runtime_variables_test.go`，扩展 `execution_test.go` | 符合 |
| 初始化迁移脚本 | 修改 MySQL/SQLite `v0.1.4__business_data.sql` | 符合用户明确决策，但有 checksum 风险 |
| frontend TriggerModal 默认值显示 | 修改 `TriggerModal.vue` | 符合 |
| spec/plan 文档 | light 模式不需要 | 符合 |
| verification 文档 | 新增本文档 | 符合 |

## Acceptance criteria checklist

- [x] backend-go 创建 PipelineRun 时构建完整 runtime variables。
- [x] 覆盖 repository builtins：`repository_id`、`repository_name`、`repository_code`、`repository_url`、`repository_ref`。
- [x] 覆盖 template builtins：`template_id`、`template_name`、`template_version`、`runtime_datetime`。
- [x] 支持 runtime overrides、repository `variable_overrides`、template declarations、stage declarations。
- [x] 合并优先级与需求一致，系统内置变量不被覆盖。
- [x] 新 run 的 `VariablesSnapshot` 保存完整 `VariableDeclaration[]`。
- [x] 执行阶段不因 task payload 中 `variables:{}` 丢失默认值。
- [x] “通用容器镜像流水线”声明 `working_dir` 与 `repository_dockerfile` 默认值。
- [x] backend-go 测试覆盖默认值合并、内置变量不可覆盖、执行路径变量解析、secret 脱敏。
- [x] frontend `TriggerModal` 初始化变量输入时使用 `value ?? default`。
- [x] backend-go 相关测试通过。
- [x] frontend lint/typecheck 通过。

## Test results

### Backend formatting

命令：

```powershell
gofmt -l "backend-go/internal/service/ci/runtime_variables.go" "backend-go/internal/service/ci/runtime_variables_test.go" "backend-go/internal/service/ci/repository.go" "backend-go/internal/service/ci/pipeline_run.go" "backend-go/internal/service/ci/execution.go" "backend-go/internal/service/ci/execution_test.go"
```

结果：通过。命令无输出，表示目标 Go 文件无需重新格式化。

### Backend tests

命令：

```powershell
go -C backend-go test ./internal/service/ci ./internal/worker/handler/ci ./internal/transport/http
```

结果：通过。

```text
ok  	backend/internal/service/ci	(cached)
ok  	backend/internal/worker/handler/ci	(cached)
ok  	backend/internal/transport/http	(cached)
```

### Frontend lint/typecheck

命令：

```powershell
yarn --cwd frontend lint --fix && yarn --cwd frontend typecheck
```

结果：通过。

```text
yarn run v1.22.22
warning package.json: No license field
$ eslint . --cache --fix
Done in 1.03s.
yarn run v1.22.22
warning package.json: No license field
$ vue-tsc --noEmit
Done in 8.31s.
```

## Missed or expanded scope

- 未发现超出本次 requirement 的业务范围扩张。
- 未创建 spec 或 plan 文档，符合 light / 轻量模式。
- 未执行 git 写操作，例如 `git add`、`git commit`、`git push`。
- 注意：当前 `git diff --cached` 显示这些改动处于 staged 状态，但本次验证阶段未执行任何 git 写操作。

## Risks

- 已按用户确认就地修改 `v0.1.4__business_data.sql`，对已经执行过该 migration 的环境可能存在 checksum 不一致风险。
- backend-go 仍未完整重构为 Python backend 的 DDD/snapshot 架构；本次只补齐 runtime variables 相关核心行为。
- secret 变量的执行值依赖持久声明/来源重新构建；如果某个 secret 只存在于已脱敏 run snapshot 中，执行阶段不会把 `***` 当作真实值使用。

## Incomplete items

- 无实现未完成项。
- 若目标环境已经应用旧版 `v0.1.4` migration，需要在部署/升级流程中单独处理 checksum 或数据修正策略。

## Conclusion

验证通过。本次实现满足已接受的轻量模式 requirement，相关 backend-go 测试、Go 格式检查、frontend lint 与 typecheck 均通过。
