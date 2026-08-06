# `export_cd_baseline.py` — 导出持续部署 SQLite 控制面

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

从 SQLite 数据库的只读连接导出持续部署控制面 SQL 基线。脚本对预置表清单逐表查询全部记录，再生成对应的 `INSERT` 语句；不按 Project、Application、Gateway 或其他业务字段筛选、转换或脱敏。

导出表覆盖 Project、Application、Version、Version Component 的完整配置、Gateway 配置、Service、Service Component、Route，以及各层环境变量。唯一明确排除的是 `deployment`，因此不包含部署历史、日志或执行状态。

环境变量值会原样写入 SQL 输出，应按敏感数据处理。

## 表清单

脚本的预置表清单是唯一导出范围：

`project`、`application`、`version`、`version_component`、`version_component_dependency`、`version_component_env`、`version_component_healthcheck`、`version_component_mount`、`version_component_endpoint`、`version_component_resource`、`version_component_tmpfs`、`version_component_ulimit`、`version_component_device`、`gateway_config`、`service`、`service_env`、`service_component`、`service_component_env`、`service_component_mount`、`service_component_resource`、`service_component_endpoint`、`route`。

这不是 CI、身份权限或任务队列的全库导出；`credential`、Repository、Pipeline、Artifact、用户和后台任务均不在持续部署控制面范围内。

## 用法

```bash
# 使用默认数据库，将完整持续部署控制面导出到新文件
python scripts/export_cd_baseline.py --output /tmp/pomelo-orbit-cd.sql

# 指定数据库，并在所有检查通过后替换已有输出
python scripts/export_cd_baseline.py \
  --database data/db/pomelo-orbit.db \
  --output /tmp/pomelo-orbit-cd.sql \
  --replace
```

脚本通过 SQLite `mode=ro` 打开数据库，先执行完整性和外键检查。输出在事务内使用普通 `INSERT`，不会静默跳过重复记录；同一事务内延迟外键检查，以保留原始表记录的导出顺序。已有输出文件必须显式传入 `--replace` 才会在完整导出成功后原子替换。

## 边界

导出内容仅限 SQLite 控制面记录，不包含容器、卷、工作区、运行日志或外部服务中的运行时数据。
