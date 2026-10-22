# 环境凭据归属迁移（000054）
最后修改时间: 2026-10-21 13:51:29

000054 将当前受管 SSH 密钥改为 Environment 的子记录：`environment_credential.environment_id` 非空、唯一，并通过 `ON DELETE RESTRICT` 外键引用 `environment(id)`。删除凭据的 `project_id` 和环境的 `ssh_credential_id/ssh_credential_revision` 存储列。环境读取仍从子表投影凭据身份，Deployment/PipelineRun 的历史快照继续保留。

## 升级准备

1. 完成或恢复并撤销旧 schema 下尚未结束的工作区搬迁批次；升级后重新预演。
2. 在发布窗口停止业务写入、worker 和活动 CLI/终端连接，备份整个数据库及对应的凭据加密配置，验证备份可恢复。
3. 对版本 53 的库执行下面的只读预检查，保存仅含 ID、分类和数量的结果。不得输出公钥、密文或私钥。
4. 确认异常已按实际数据事实处理，部署应用与 schema 的同一版本，通过既有迁移入口执行 000054。不要让旧应用或旧 CLI 连接新 schema。

以下 SQL 适用于 SQLite、MySQL、PostgreSQL，必须在升级前运行。

```sql
SELECT e.id AS environment_id, e.project_id,
       e.ssh_credential_id AS credential_id,
       CASE
         WHEN e.ssh_credential_id IS NULL AND e.ssh_credential_revision IS NULL
           THEN 'no_credential'
         WHEN e.ssh_credential_id IS NULL THEN 'revision_without_credential'
         WHEN e.target_type <> 'ssh' THEN 'non_ssh_binding'
         WHEN c.id IS NULL THEN 'missing_credential'
         WHEN c.project_id <> e.project_id THEN 'cross_project'
         WHEN e.ssh_credential_revision IS NULL OR c.revision < 1
              OR c.revision <> e.ssh_credential_revision THEN 'revision_mismatch'
         ELSE 'valid_binding'
       END AS binding_state
FROM environment e
LEFT JOIN environment_credential c ON c.id = e.ssh_credential_id
ORDER BY e.id;

SELECT ssh_credential_id AS credential_id, COUNT(*) AS binding_count
FROM environment
WHERE ssh_credential_id IS NOT NULL
GROUP BY ssh_credential_id HAVING COUNT(*) > 1;

SELECT c.id AS unbound_credential_id, c.project_id
FROM environment_credential c
WHERE NOT EXISTS (SELECT 1 FROM environment e WHERE e.ssh_credential_id = c.id)
ORDER BY c.id;
```

`no_credential` 是 local 或未初始化 SSH 的正常状态。只有 `valid_binding` 能迁入新子表；其他状态或重复绑定须在发布前查明。迁移包含异常检查，异常绑定会在替换业务表之前失败，不会自动清空指纹或生成替代密钥。

## 数据映射与检查

有效绑定以旧 `environment.ssh_credential_id` 为唯一来源，复制原凭据 ID、公钥、加密私钥、revision、created_at，并填入主表的 environment ID。环境目标修订、指纹、Probe/Gateway 字段不改，因此有效任务不会仅因结构迁移失效，也不需要重新安装原公钥。

未被当前环境绑定的旧密钥不进入新表，包括仅被历史任务快照引用的密钥。升级前备份负责保留这些历史行；历史快照保留不意味着对应旧私钥仍可在线读取。

升级后检查子表行数是否等于升级前的有效绑定数，以下异常查询均应为空：

```sql
SELECT c.id, c.environment_id
FROM environment_credential c
LEFT JOIN environment e ON e.id = c.environment_id
WHERE e.id IS NULL OR e.target_type <> 'ssh' OR c.revision < 1;

SELECT environment_id, COUNT(*) AS credential_count
FROM environment_credential GROUP BY environment_id HAVING COUNT(*) > 1;
```

核对迁移版本为 54 且非 dirty；对照升级前保存的 ID/修订和环境状态验证保留结果。SQLite 另执行 `PRAGMA foreign_key_check;`，结果应为空；应用连接必须启用外键（现有数据库连接工厂已配置）。恢复业务后，重复生成初始化命令应复用原公钥和创建时间。

## 失败与回退

SQLite 迁移显式使用事务，驱动不再额外包装；检查失败可能使该连接仍处于事务内。先在同一连接回滚或关闭失败连接，再检查旧表及 migration dirty 状态。PostgreSQL 使用显式事务。MySQL DDL 隐式提交，后续 DDL 失败可能留下中间表或已变更结构；必须检查实际执行位置，按备份恢复，不能假定整份迁移自动回滚。

不要仅清除 dirty 标记或盲目重跑来掩盖失败。先核对表结构和数据；异常预检查失败时根据业务事实修复旧绑定，DDL 部分完成时优先恢复升级前完整备份和对应程序版本，再执行迁移。

down 恢复当前子记录的 `project_id` 以及环境的凭据 ID/修订，通过 Environment 找回 Project 归属，保留当前有效密钥。**down 无法恢复升级时排除的未绑定旧密钥**；需要完整回到升级前状态时使用数据库备份和当时的加密配置。回退期间同样停写，并在 schema 恢复后切换对应应用版本。
