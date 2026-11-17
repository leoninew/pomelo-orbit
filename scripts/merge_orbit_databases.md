# Orbit 双库合并脚本用法

将两套 PostgreSQL 中指定 Orbit 项目的业务数据合并并写回两边。同一对象采用最新配置，不同记录取并集，统一主键及关联。Environment 的部署目标配置和 SSH 凭据保留各自数据库中的原值，不在两边同步。

部署、流水线运行和后台任务的状态不影响合并；包括 `running` 在内的状态及锁字段按选中记录原样保留。每条记录只依据自身的时间字段选择：依次取 `updated_at`、`finished_at`、`started_at`、`created_at` 中首个非空值；不以关联记录的时间比较不同来源的记录。时间相同但内容不同则报告冲突。版本、服务的关联配置随选中的父记录整体选取。

## 准备

在仓库根目录运行，需要 Python ≥3.12 和 `dbtalk`。涉及加密凭据时安装 `cryptography`：

```bash
python -m pip install cryptography
```

DSN 使用 `.env` 或进程中已有的 `DBTALK_DSN_*` 变量。下面的 `LEFT_ORBIT_KEY`、`RIGHT_ORBIT_KEY` 是存放对应实例实际 Fernet 密钥的进程变量名，用于校验合并的仓库凭据；没有加密仓库凭据时省略两个密钥参数。Environment SSH 凭据按库保留，不需要通过这两个参数解密或迁移。

本地开发密钥位于 `.env.development` 的 `POMELO_ORBIT_JWT__SECRET_KEY`。将该变量加载到当前进程后，可将两个密钥参数都设为此变量名；脚本会验证它能否解密两边的仓库凭据，不输出密钥或解密内容。脚本不会自动加载 `.env.development`。

## 生成方案

```bash
python scripts/merge_orbit_databases.py plan \
  --left-dsn-env DBTALK_DSN_LEON_POSTGRES_ORBIT \
  --right-dsn-env DBTALK_DSN_WANGM_POSTGRES_ORBIT \
  --left-key-env LEFT_ORBIT_KEY \
  --right-key-env RIGHT_ORBIT_KEY \
  --project-code orbit.think.preflite.cn \
  --output data/orbit-db-merge/think.plan.json
```

此命令只读数据库。检查方案中的 `before_rows`、`merged_rows_by_side`、`changes_by_side`、`decisions` 和 `id_mapping`，分别查看两边完整原值、每边最终结果、各边变更、选择依据和主键映射。Environment 的 target 字段和 SSH 凭据在 `merged_rows_by_side` 中按库保留，仅 Environment 的 ID、项目关联和项目编码可随关联数据统一。方案的 `status` 为 `ready` 后可执行。

## 执行方案

```bash
python scripts/merge_orbit_databases.py apply \
  --left-dsn-env DBTALK_DSN_LEON_POSTGRES_ORBIT \
  --right-dsn-env DBTALK_DSN_WANGM_POSTGRES_ORBIT \
  --left-key-env LEFT_ORBIT_KEY \
  --right-key-env RIGHT_ORBIT_KEY \
  --plan data/orbit-db-merge/think.plan.json
```

执行事务回滚演练后，分别提交两库并回读核验。结果写入 `data/orbit-db-merge/think.plan.execution.json`，包含逐库状态、完整 `actual_rows` 及与预期结果的 `verification_differences`。

一边已完成、另一边失败时，重跑相同命令续接。数据、schema 或数据库实例在生成方案后发生变化时，重新生成方案。

状态不阻断合并，但执行时仍检查数据指纹；业务在生成方案后写入新数据会使旧方案失效，避免覆盖新记录。

## 独立核验

```bash
python scripts/merge_orbit_databases.py verify \
  --left-dsn-env DBTALK_DSN_LEON_POSTGRES_ORBIT \
  --right-dsn-env DBTALK_DSN_WANGM_POSTGRES_ORBIT \
  --project-code orbit.think.preflite.cn \
  --output data/orbit-db-merge/think.verify.json
```

报告包含两边完整业务数据、逐行差异、`different_tables` 和 `identical`。Environment 配置差异会报告，但不会由 `apply` 覆盖。核验不需要密钥参数。

## 参数

| 参数 | 命令 | 说明 |
| --- | --- | --- |
| `--left-dsn-env` | 全部 | 左库已有 DSN 变量名，必填 |
| `--right-dsn-env` | 全部 | 右库已有 DSN 变量名，必填 |
| `--cwd` | 全部 | dbtalk 查找 `.env` 的目录，默认当前目录 |
| `--project-code`、`--project-name` | plan、verify | 二选一；精确项目编码或名称，必填，一次一个项目。名称有歧义时失败；选中的项目必须能在两库间通过 ID 或编码对应 |
| `--output` | plan、verify | 本地 JSON 文件路径，必填 |
| `--plan` | apply | 已生成的方案文件路径，必填 |
| `--left-key-env` | plan、apply | 左实例 Fernet 密钥的进程变量名，用于校验仓库凭据 |
| `--right-key-env` | plan、apply | 右实例 Fernet 密钥的进程变量名；密钥相同时可与左侧相同 |

## 退出码

| 退出码 | 含义 |
| --- | --- |
| `0` | 成功 |
| `1` | 输入、连接、检查或执行失败，查看终端错误及输出报告 |
| `2` | verify 发现两边数据存在差异 |

查看单个命令帮助：

```bash
python scripts/merge_orbit_databases.py plan --help
python scripts/merge_orbit_databases.py apply --help
python scripts/merge_orbit_databases.py verify --help
```
