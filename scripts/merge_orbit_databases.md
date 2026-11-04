# Orbit 双库合并脚本用法

将两套 PostgreSQL 中指定 Orbit 项目的业务数据合并并写回两边。同一对象采用最新配置，不同记录取并集，统一主键及关联。

## 准备

在仓库根目录运行，需要 Python ≥3.12 和 `dbtalk`。涉及加密凭据时安装 `cryptography`：

```bash
python -m pip install cryptography
```

DSN 使用 `.env` 或进程中已有的 `DBTALK_DSN_*` 变量。下面的 `LEFT_ORBIT_KEY`、`RIGHT_ORBIT_KEY` 是存放对应实例实际 Fernet 密钥的进程变量名；项目没有加密凭据时省略两个密钥参数。

## 生成方案

```bash
python scripts/merge_orbit_databases.py plan \
  --left-dsn-env DBTALK_DSN_LEON_POSTGRES_ORBIT \
  --right-dsn-env DBTALK_DSN_WANGM_POSTGRES_ORBIT \
  --left-key-env LEFT_ORBIT_KEY \
  --right-key-env RIGHT_ORBIT_KEY \
  --project orbit.think.preflite.cn \
  --output data/orbit-db-merge/think.plan.json
```

此命令只读数据库。检查方案中的 `before_rows`、`merged_rows`、`changes_by_side`、`decisions` 和 `id_mapping`，分别查看两边完整原值、完整合并结果、各边变更、选择依据和主键映射。方案的 `status` 为 `ready` 后可执行。

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

## 独立核验

```bash
python scripts/merge_orbit_databases.py verify \
  --left-dsn-env DBTALK_DSN_LEON_POSTGRES_ORBIT \
  --right-dsn-env DBTALK_DSN_WANGM_POSTGRES_ORBIT \
  --project orbit.think.preflite.cn \
  --output data/orbit-db-merge/think.verify.json
```

报告包含两边完整业务数据、逐行差异、`different_tables` 和 `identical`。核验不需要密钥参数。

## 参数

| 参数 | 命令 | 说明 |
| --- | --- | --- |
| `--left-dsn-env` | 全部 | 左库已有 DSN 变量名，必填 |
| `--right-dsn-env` | 全部 | 右库已有 DSN 变量名，必填 |
| `--cwd` | 全部 | dbtalk 查找 `.env` 的目录，默认当前目录 |
| `--project` | plan、verify | 任一侧的精确项目编码或 ID，必填，一次一个项目 |
| `--output` | plan、verify | 本地 JSON 文件路径，必填 |
| `--plan` | apply | 已生成的方案文件路径，必填 |
| `--left-key-env` | plan、apply | 左实例实际 Fernet 密钥的进程变量名 |
| `--right-key-env` | plan、apply | 右实例实际 Fernet 密钥的进程变量名；密钥相同时可与左侧相同 |
| `--allow-idle-deployment ID` | plan | 允许原样合并指定历史 running 部署，可重复；要求双方部署及关联任务相同、任务成功且无锁、完成已超过 7 天 |

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
