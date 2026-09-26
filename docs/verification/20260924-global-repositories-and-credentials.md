# 代码仓库与仓库凭据跨项目复用验证
最后修改时间: 2026-09-26

## 已验证

- `task sqlc`、`task proto` 已在实现过程中执行，生成代码随 SQL 和 Proto 定义更新。
- `task check` 通过；`yarn --cwd web lint:fix` 和 `yarn --cwd web typecheck` 通过。新增选择器测试文件有 7 条非阻断 lint warning，0 个 lint error。
- `TMPDIR=/private/tmp go test ./cmd/... ./internal/...` 通过。macOS 默认临时目录 `/var/folders` 会被源码路径校验规范化为 `/private/var/folders`，使既有 `repositorysource` 测试按原始路径比较时失败；设置规范化的临时目录后该测试通过。
- `yarn --cwd web test src/components/RepositorySelect.test.ts` 通过：远端搜索、翻页、已选仓库回显与 Project 切换后丢弃旧响应。
- `yarn --cwd web test` 通过：31 个测试文件、135 条测试。Gateway 未配置页测试已按当前共享空态行为修正。
- `TestSharedRepositoryAndCredentialInstantiateInAnotherProject` 通过：在 Project A 创建仓库与凭据，Project B 成员读取并用同一仓库实例化本 Project 的 Application Pipeline。
- `TestSharedRepositoryRunsInEachProjectWithSharedCredential` 通过：两个 Project 的 Application Pipeline 使用同一仓库和凭据分别创建、执行 Run；Run 仍按 Project 隔离，执行器取得共享凭据。
- `go test ./internal/application/pipeline/usecase -run TestDeleteApplicationPipelineRequiresFinishedRuns -count=1` 通过：等待中的 Run 阻止删除 Application Pipeline，终态 Run 允许删除并保留历史。
- `TestNonRepositoryCredentialCannotBeReadOrBoundAsRepositoryCredential` 通过：即使非仓库类型的历史凭据被误设为全局，按 ID 的详情、导出和仓库绑定也不可使用它。
- `TestLockApplicationPipelineReportsDeletedRow` 与 MySQL 连接配置测试通过：锁操作识别已删除的 Pipeline；MySQL 使用匹配行数和 `READ COMMITTED`，避免无变化 UPDATE 误判或锁后读取旧事务快照。
- 现有 SQLite `MigrateUp` 测试随 Go 全量测试通过；`git diff --check` 通过。

## 旧库转换前核对

就地修改迁移文件不会升级已经执行相应版本的数据库。执行旧库转换前，需先备份，并在目标库核对跨 Project 重复的 `repository.code`、仓库凭据名称、仓库与 Pipeline 引用，以及旧凭据的 `type` 与 `project_id`。迁移 000042 只是把旧 `credential` 表整体重命名为 `repository_credential`；它不会把历史部署私钥搬到 `environment_credential`。仅将 `git_ssh`、`github_token`、`gitee_token`、`gitea_token`、`registry_token` 转为全局仓库凭据；其他类型须确认用途和引用并保持隔离，不能统一清空 `project_id`。冲突须显式处理，保留原 ID、密文和历史引用，再施加全局唯一约束。转换结果与回退依据须在实际操作时记录。

目标库已执行迁移 000042 时，可先运行以下只读审计；更早版本将表名 `repository_credential` 换为 `credential`：

```sql
SELECT code, COUNT(*) AS total FROM repository GROUP BY code HAVING COUNT(*) > 1;
SELECT name, COUNT(*) AS total FROM repository_credential GROUP BY name HAVING COUNT(*) > 1;
SELECT id, project_id, name, type FROM repository_credential
WHERE type NOT IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token');
SELECT r.id AS repository_id, r.git_credential_id, c.type AS credential_type
FROM repository r LEFT JOIN repository_credential c ON c.id = r.git_credential_id
WHERE r.git_credential_id IS NOT NULL
  AND (c.id IS NULL OR c.type NOT IN ('git_ssh', 'github_token', 'gitee_token', 'gitea_token', 'registry_token'));
```

## 未完成的环境验证

- 本机没有 Docker、`psql` 或 `mysql` 可执行程序，也没有本任务可用的 PostgreSQL/MySQL 测试实例；这两种数据库的完整迁移链、回退和并发删除行为尚未实跑。
- 未发现可供本任务转换的现有业务数据库，因此尚未执行旧库类型审计、冲突处理与数据转换，也未验证真实跨 Project Run 的工作区执行。就地修订已执行迁移只影响新建库；不能把修改迁移文件视为已完成旧库升级。
- 按用户要求不做浏览器 E2E，也不以覆盖率为目标扩展测试分支。
