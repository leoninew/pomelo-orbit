# 版本组件结构化表单规格
最后修改时间: 2026-07-28 18:30:43

Review status: Accepted

## Requirement basis

依据 `docs/requirement/20260728-version-component-tabs.md`：Version Component 不再以 `*_json` 作为 API 或持久化模型；迁移通过 SQL 就地完成，业务代码不承担兼容或数据搬迁。

## Overview

`version_component` 保留组件身份和标量策略字段：`id`、`version_id`、`name`、`image`、`pull_policy`、`restart_policy`、时间戳。其余 Component 配置使用关联表，API 以嵌套结构化消息读写，Compose Renderer 直接使用领域结构。

Version 详情保留组件目录；组件独立详情路由承载未发布版本的表单和已发布版本的同布局只读展示。

## Structured model

| 配置 | 持久化结构 | API 结构 |
|---|---|---|
| command / args | `version_component_argument(component_id, kind, position, value)` | `repeated string command` / `args` |
| 环境变量 | `version_component_env(component_id, key, value, position)` | `repeated ComponentEnv` |
| 端口 | `version_component_port(component_id, host_port, container_port, position)` | `repeated ComponentPort` |
| 挂载 | `version_component_mount(component_id, source_type, source, target, read_only, content, content_mode, position)` | `repeated ComponentMount` |
| 网络 | `version_component_network(component_id, name, position)` | `repeated string networks` |
| 依赖 | `version_component_dependency(component_id, depends_on_name, condition, position)` | `repeated ComponentDependency` |
| 健康检查 | `version_component_healthcheck(component_id, test_mode, test, interval, timeout, retries, start_period, start_interval, disabled)` | `ComponentHealthcheck` |
| 资源 | `version_component_resource(component_id, limit_cpus, limit_memory, reservation_cpus, reservation_memory)` | `ComponentResources` |
| tmpfs | `version_component_tmpfs(component_id, target, size_bytes, mode, position)` | `repeated ComponentTmpfs` |
| ulimit | `version_component_ulimit(component_id, name, soft, hard, position)` | `repeated ComponentUlimit` |

`test_mode` 为 `CMD` 或 `CMD-SHELL`；`disabled` 表达 Compose 的 `healthcheck.disable`。测试命令以单一文本维护；`CMD` 在渲染 Compose 时解析为 argv，`CMD-SHELL` 原样传递。

资源表单范围为 CPU 和内存的 limits / reservations；现有开发数据只使用 `limits.memory`，迁移后保持原值。未被表单模型支持的 Component JSON 形状在转换前应视为迁移失败，不能静默删除。

## API and domain

`VersionComponentReq` / `VersionComponentResp` 以嵌套消息替换所有 `*_json` 字段。HTTP mapper 转换为领域 DTO；领域校验检查：

- Component 名称在 Version 内唯一，端口、环境变量、挂载目标、tmpfs 目标和 ulimit 名称不重复。
- 依赖必须引用同一 Version 的 Component，条件仅允许 `service_started`、`service_healthy`、`service_completed_successfully`。
- 健康检查启用时必须有测试模式和至少一个测试参数；重试次数非负。
- 资源数值作为 Compose 字符串保存，空字段不输出。

Compose Renderer 根据结构化值生成 `command`、`environment`、`ports`、`volumes`、`networks`、`depends_on`、`healthcheck`、`deploy.resources`、`tmpfs` 与 `ulimits`；不再调用 Component JSON parser。

## Schema and SQLite conversion

按用户决定，SQLite 和 MySQL 的 `000023_application` 建表 SQL 原地改为新结构。开发 SQLite `data/db/pomelo-orbit.db` 当前 schema migration 为 30；其数据转换由专用脚本 `scripts/migrate_sqlite_version_component.py` 执行，不在会话中临时拼接 SQL，也不在业务代码或服务启动阶段执行。

脚本以显式数据库路径运行，并具备以下固定流程：

1. 确认目标库 schema migration 为预期版本、`version_component` 仍包含旧 JSON 列；否则退出，不尝试猜测状态。
2. 使用 Python 标准库解析所有现存 JSON，按结构化模型逐项验证；任何无法表示的值均报出 Component 与字段并退出。
3. 在目标库同目录创建一次性备份，再以 `BEGIN IMMEDIATE` 开启单个事务。
4. 创建所有新 Component 关联表及索引，将预检后的数据按顺序写入；随后重建仅含标量字段的 `version_component`。
5. 在提交前核对 Component 数量、每类关联项数量、Component 引用和 `PRAGMA foreign_key_check`；失败则回滚。
6. 脚本提供 `--check`，只执行预检和转换统计；实际写入需要显式 `--apply`。

MySQL 建表 SQL 同步更新；不在 Go usecase、HTTP handler 或服务启动流程写入迁移、兼容读或回填逻辑。该 Python 脚本只处理本地开发 SQLite，不作为业务交付中的运行时依赖。

## UI

`/version/:versionId/component/:componentId` 为 Component 详情页；`new` 作为创建态。

- 基础、命令与参数、端口、环境变量、挂载、网络与依赖、健康检查、资源、tmpfs、ulimit 各自为表单分区。
- 未发布 Version 可编辑，在页面底部一次保存；已发布 Version 同位置显示只读字段，无编辑或删除。
- 版本详情组件区只显示名称、镜像和进入详情动作；新增直接进入创建态。
- 删除组件在组件详情页确认；若有 Expose 或其他 Component 依赖，直接显示引用项并阻止删除。

## Risks and alternatives

保留 JSON 并只建前端适配层会继续让校验和渲染依赖隐式数据形状，已排除。将每项都改为 Component 宽表字段无法表达端口、挂载、环境变量和命令参数的有序多值关系，已排除。

SQL 原地转换失败必须整体回滚并保留旧表；不可使用“忽略无法解析项”的降级策略。

## Confirmed decisions

1. 仅移除 `VersionComponent.*_json`；`Version.env_json` 和 `Service.runtime_config_json` 不在本轮处理。
2. Component 提供独立 CRUD API，以 Component ID 定位；不再由组件页面整份覆盖 Version。
3. Component 改名由后端事务更新 Expose 和依赖引用；删除被 Expose 或 Component 依赖引用的 Component 时阻止并列出引用项。
4. SQLite 开发库使用 Python 脚本原地转换；MySQL 没有存量数据，只更新 `000023_application` 的新库建表 SQL。
5. 未知 JSON 形状或约束冲突即失败并保留原库；`--apply` 在数据库同目录创建备份，备份加入忽略规则。
6. command 和 args 是有序参数行，保留当前 Compose 语义；不支持猜测型单行 shell command。
7. 挂载改为显式目录、文件、命名卷和 special 类型；只有文件挂载可填写内容及 `seed/sync`。
8. 健康检查支持 `CMD`、`CMD-SHELL`、禁用、测试参数行和既有时长/重试字段；时长使用 Compose duration 文本，不扩展其他字段。
9. 资源仅支持 CPU/内存的 limits/reservations；tmpfs 和 ulimit 保留当前范围，不添加 `pids`、设备或更多 ulimit 名称。
10. 网络为可编辑的有序名称列表；依赖支持三种条件、禁止自依赖和环。
11. Component `pull_policy` 仅接受 `always`、`missing`、`never`，非空时覆盖 Application 的拉取策略；重启策略仍限 `no` / `unless-stopped`。
12. 组件详情页默认查看，未发布 Version 可切换编辑，两个状态保持相同分区；`new` 路由直接编辑，仅显式保存。
13. 表单文本按原值保存和回显。可选字段仅在用户明确留空时省略；动态行不自动丢弃。非法输入由前后端显式校验拒绝，不通过 fallback、默认填充或 normalize 改写。
