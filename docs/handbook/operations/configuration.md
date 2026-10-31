# 系统配置

实例配置控制平台的数据库、监听、日志、认证、LLM 和 Worker。业务容器配置在[服务](../guides/services.md)中管理。

## 配置来源

启动工作目录必须包含 `configs/config.yaml`。`POMELO_ORBIT_APP__ENV` 选择 profile，来源从低到高依次覆盖：

1. `configs/config.yaml`。
2. `configs/config.<profile>.yaml`。
3. `.env` 或 `.env.<profile>`。
4. 进程启动环境。
5. `overwrite.env` 应用覆盖。

有 profile 时只读对应 dotenv，不叠加 `.env`。profile 在启动环境或 `overwrite.env` 中选择；配置文件不向父目录自动查找。

| 写法 | 日志级别示例 |
| --- | --- |
| YAML | `logging.level` |
| 系统设置键 | `logging__level` |
| 环境或覆盖文件 | `POMELO_ORBIT_LOGGING__LEVEL` |

时长使用 `60s`、`2m` 等形式；环境列表用 CSV，页面列表每行一项。`overwrite.env` 按字面值保存，不展开 shell 或业务变量。

## 在页面中修改

打开“系统管理 → 系统设置”。查看需要 `setting:read`，修改需要 `setting:write`。

| 页面信息 | 含义 |
| --- | --- |
| 默认值 | 启动文件与进程环境提供的基线，不含应用覆盖 |
| 当前值 | 当前后端进程实际使用的值 |
| 应用覆盖 | 已存入 `overwrite.env` 的值 |
| 待重启提示 | 保存结果与当前运行值有差异，或下次配置有错误 |

1. 编辑目标字段，关联项一起编辑，例如 LLM 地址、Key 和模型。
2. 点击“保存全部”，提交本页所有待保存修改，包括被搜索隐藏的项。
3. 查看校验结果与重启提示，安排相关进程重新加载配置，再核对当前值。

编辑框优先带入已保存覆盖，否则使用默认值。保存校验类型与组合，不测试外部服务连通性。

## 重置与冲突

重置只删除应用覆盖，点击后需“保存全部”才生效，随后回到下层配置。页面取消只撤销未保存编辑。

| 错误 | 处理 |
| --- | --- |
| `settings_revision_conflict` | 重新加载，比较他人修改后再编辑 |
| `settings_store_busy` | 稍后重试，持续失败则检查协同进程与文件访问 |
| 下次配置无效 | 检查类型、关联项、profile 及实际配置文件后再重启 |

系统设置读取可包含秘密原值。保护 `overwrite.env` 和工作目录下 `data/config/` 的锁及恢复材料，密钥用途见[密钥管理](./secrets.md)。初始化默认值只影响向导，不更新已有环境或网关。
