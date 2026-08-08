# `cd_baseline.py` — 导出与导入持续部署 SQLite 控制面

Doc role: local script reference。与代码冲突时以代码为准。

## 用途

`export` 从 SQLite 数据库的只读连接导出持续部署控制面 SQL 基线；`import` 将该 SQL 顺序导入已有的 SQLite 控制面数据库。脚本对预置表清单逐表查询全部记录，再生成对应的 `INSERT` 语句；不按 Project、Application、Gateway 或其他业务字段筛选、转换或脱敏。

两个子命令的 `--database` 默认值均为相对于当前工作目录的 `data/db/pomelo-orbit.db`。

导出表覆盖 Project、Application、Version、Version Component 的完整配置、Gateway 配置、Service、Service Component、Route，以及各层环境变量。唯一明确排除的是 `deployment`，因此不包含部署历史、日志或执行状态。

环境变量值会原样写入 SQL 输出，应按敏感数据处理。

## 表清单

脚本的预置表清单是唯一导出范围：

`project`、`application`、`version`、`version_component`、`version_component_dependency`、`version_component_env`、`version_component_healthcheck`、`version_component_mount`、`version_component_endpoint`、`version_component_resource`、`version_component_tmpfs`、`version_component_ulimit`、`version_component_device`、`gateway_config`、`service`、`service_env`、`service_component`、`service_component_env`、`service_component_mount`、`service_component_resource`、`service_component_endpoint`、`route`。

这不是 CI、身份权限或任务队列的全库导出；`credential`、Repository、Pipeline、Artifact、用户和后台任务均不在持续部署控制面范围内。

## 用法

```bash
# 使用默认数据库导出完整持续部署控制面
python scripts/cd_baseline.py export --output data/exports/pomelo-orbit-cd.sql

# 生成可清空同范围 CD 数据后再导入的 SQL
python scripts/cd_baseline.py export --output data/exports/pomelo-orbit-cd.sql --relace

# 将 export 生成的 SQL 导入已有数据库
python scripts/cd_baseline.py import  --input data/exports/pomelo-orbit-cd.sql
```

`export` 通过 SQLite `mode=ro` 打开数据库，先执行完整性和外键检查。输出使用普通 `INSERT`，不会静默跳过重复记录。`--relace`（参数名按脚本接口拼写）会在所有 `INSERT` 前按依赖反序写入该导出范围内每张表的 `DELETE`，用于恢复同一份 CD 基线。输出文件仅在导出检查通过后原子替换。

`import` 不创建事务，按文件顺序执行 SQL。为处理 Version 自引用及 Service 对 Version 的引用，生成的 SQL 在导入期间暂时关闭 SQLite 外键检查，并在末尾重新开启；导入命令随后执行完整性和外键检查。SQL 出错时，已经执行的语句不会回滚，必须从导入前备份恢复数据库。

## 边界

导出内容仅限 SQLite 控制面记录，不包含容器、卷、工作区、运行日志或外部服务中的运行时数据。
