# 系统设置实际配置读取与持久化修改计划
最后修改时间: 2026-10-09 19:55:36

Review status: Accepted

Mode: standard

当前阶段：验证 / Verification。依据已接受的 [Intent](../intent/20261006-settings-persistent-overrides.md)。用户已反馈人工测试没有大问题，并要求完成验证文档；验收证据与未完成事项见 [Verification](../verification/20261006-settings-persistent-overrides.md)。

## 已确认方案

- 保持 YAML + ENV，固定 `<启动工作目录>/overwrite.env`；不新增 JSON 配置格式、定位变量、只读配置项或配置名单登记。
- 原生 Windows/Linux 从该文件读取应用覆盖；当前容器镜像 WORKDIR 是 `/app`，因此读取 `/app/overwrite.env`。
- 容器基线沿用组件 ENV 渲染的 `environment`。后端没有 `env_file` 能力，本任务不扩展该能力。
- 容器覆盖文件使用已有 `controlled_file`：可写、空初始内容、mode `0600`、`ignore_if_exists=true`。首次初始化，后续部署保留应用修改。
- 原生及容器的保存、加载、重置使用相同绝对路径；工作目录和文件位置在启动时固定，不随配置修改改变。
- 配置项由 typed Config 的 mapstructure 标签自动发现；普通新字段只需要正常声明、默认值和必要的业务校验，不维护额外 ENV/UI 名单。

## 当前代码问题

| 位置 | 问题与处理 |
| --- | --- |
| `internal/config/config.go` | godotenv.Load 将 dotenv 写入 OS ENV；手工 BindEnv 列表会遗漏新字段；没有读取 overwrite.env。改为捕获 ENV、显式合并，并由 Config 自动发现字段 |
| `internal/application/settings/usecase/service.go` | 从 YAML Base 和一份 dotenv 拼装页面，未展示真实运行值；改为运行快照、基线和下一次配置 |
| `internal/infrastructure/storage/local/envfile` | 手写解析和直接截断写入，没有完整校验或并发控制；替换为统一字面量 ENV codec 和文件事务 |
| `internal/bootstrap/http.go` | 原设置 Store 指向启动 dotenv；改为固定 overwrite.env |
| `Settings.vue` | 本地 needsRestart 刷新后丢失，类型依赖默认值，只能一次编辑一项；改为服务端状态、显式类型及批量草稿 |
| 后端 Compose renderer | 只有 environment 与 volumes，没有 env_file；复用现有受控文件挂载，不写不存在的多 env_file 场景 |

## 加载规则

优先级从低到高：默认 YAML、profile YAML、启动 `.env[.<profile>]`、启动进程 ENV、overwrite.env。

1. 捕获启动工作目录及原始进程 ENV，读取固定覆盖文件。
2. profile 先取覆盖的 APP__ENV，否则沿用原进程 APP__ENV；dotenv 的 APP__ENV 不递归切换文件。选择器不能包含路径分隔符。
3. YAML 使用现有 Viper；启动 dotenv 保留现有 godotenv 解析。ENV 和文件合并不修改进程环境。
4. 覆盖文件使用相同 ENV 键名、字面量值，由 `hashicorp/go-envparse v0.1.0` 解析，不进行变量插值；空串、false、0 和空列表不是缺失。重复 ENV 键按类库规则取最后值。
5. 覆盖值最后通过 Viper 显式值进入唯一解析流程，按字段类型解析，然后复用规范化与完整 Validate。
6. 没有覆盖文件或文件为空时没有应用覆盖；读取、格式或校验失败不能被当成空文件。
7. 保留规范化且运行派生前的比较快照，再执行空 worker.id 的 hostname/PID 派生。当前实际配置是启动快照，保存不更新运行资源。
8. 预览使用固定原始 ENV 和当前 YAML/dotenv；不能预测未来容器 ENV。修改启动基线需新进程；独立 worker/MCP 分别重启。

覆盖文件按键排序写为 `KEY="value"`，只借助 `encoding/json` 实现类库支持的字符串转义，文件格式仍为 ENV。接受类库约 64 KiB 的单行上限，按键名与转义后的值合计计算；保存前用同一类库解析编码结果，超限时返回错误并保留原文件，不扩展为大文本存储。

字段发现复用 mapstructure 标签，跳过 `-` 元数据，识别 duration、string、bool、int、string list。字段描述是可选说明，不是注册前提。设置项列表、ENV 映射、保存白名单、类型和读取值都用同一份字段描述。

## 原生与容器操作

| 场景 | 基线修改 | 应用覆盖修改 | 生效与保留 |
| --- | --- | --- | --- |
| Windows 原生 | 改启动 dotenv 或服务 ENV | 设置页写工作目录 overwrite.env | 重启进程；升级保留文件及 data/config |
| Linux 原生 | 同上 | 同上 | 同上；运行用户对文件和 data/config 可写 |
| 容器 | 改组件 ENV，重新部署创建容器 | 设置页写 `/app/overwrite.env` 的可写挂载 | 普通重启读取覆盖；重建继续使用同一宿主文件与 data |

受控文件示例：source `./data/config/overwrite.env`，target `/app/overwrite.env`，Content 空，ReadOnly=false，Mode=0600，IgnoreIfExists=true。data 目录同时普通持久化挂载到 `/app/data`，保存恢复材料不会落入容器临时层。这里使用通用挂载，不查询业务数据库、不识别或回写自身 Service。

## 文件保存、并发与恢复

保持覆盖目标文件的 inode，适配 Linux 单文件 bind mount。启动读取和设置操作共用文件存储实现。

- 锁与恢复材料固定在 `<启动工作目录>/data/config`，按目标文件名命名。Linux flock、Windows LockFileEx，等待最多 5 秒并响应 context。
- 所有读取与修改先加跨进程锁、恢复中断事务，再读取完整文件。内容及 revision 来自同一次读取；revision 为按稳定顺序编码的 ENV 值指纹，不把注释或排版差异当成配置修改。
- 写事务锁内比较 revision、合并用户更新与重置、完整候选配置校验。失败不开始目标写入。
- 将原文件内容以 ENV 文件备份到恢复目录，通过临时文件、Sync 和 rename 发布 prepared 恢复记录。
- 使用同一目标文件原地写入、调整长度、Sync。随后将 prepared 记录 rename 为 committed，作为提交点；提交后清理记录。
- 中断留下 prepared 时，下次读取先恢复原内容；留下 committed 时保留已写入内容再清理。读取者不读取半写文件。
- 提交前写入失败尝试恢复原内容；恢复失败仍保留原内容记录并返回错误，后续访问必须先恢复成功。提交后清理失败不把已提交报告为未保存。
- ctx 在写入开始前取消则不提交；写入开始后完成提交或恢复，不中途留下未标记的半文件。
- 文件锁和 journal 所在 data/config 必须持久化；多进程必须共享同一覆盖文件及同一恢复目录。手动改文件时先停止共享写入进程。
- 保证进程中断和合作写入的恢复；不宣称损坏磁盘或不支持 OS 锁的文件系统有额外保证。

ENV codec、锁与事务放在无 application/infrastructure 依赖的 common 包，config 启动与 infrastructure Store 共用；启动不引入反向层依赖。

## 设置接口与页面

接口遵循单数 `/api/setting/config`，GET/PUT/DELETE 保留 `setting:read`、`setting:write` 权限及当前 HTTP 错误契约。

- 项目返回：key、type、当前 value、基线 default、override_value、next_value、is_overridden、secret、pending_restart；响应含 revision、pending_restart 和下一配置错误。
- 当前值来自启动最终 Config，保存值来自 overwrite.env，下次值由同一配置解析器计算。来源表达 YAML/dotenv/ENV/override/derived，不能把 Compose 的某一文件名推测为 ENV 来源。
- PUT 使用 revision、updates[{key,value}]、reset_keys；DELETE 使用 revision、keys。一批操作只有一次完整校验和文件提交，满足 LLM 关联项一起修改/重置。
- 未知 key、错误类型、重复操作、完整校验失败为 validation；过期 revision 为 conflict；文件忙为 unavailable；存储错误按现有 internal 错误处理。失败不泄露值。
- 保持已有 Proto 字段编号；新状态追加字段，批量请求替换单项字段并 reserve 原编号/名称。Go/TS 使用 task proto 生成。
- 页面保留原四列表格、列宽、搜索栏及行内“编辑/保存/取消/重置”文字操作；当前值单元格只展示生效值，待重启状态使用原有顶部提示，不新增下次启动值说明、工具栏操作区或额外列。布尔沿用 RawValueSelect，整数使用 numeric input，列表逐行输入。
- 支持同时编辑多行或暂存重置；只有一项时行内“保存”，多项时行内“保存全部”，按一批提交以满足关联校验。保留 SensitiveValue 与权限控制。
- 保存/重置失败保留草稿和实际错误，成功使用整个响应；刷新后待重启仍由后端返回。搜索只过滤，不丢隐藏行草稿。
- 无热更新或自动重启。敏感标记只影响显示，所有应用配置均可编辑。空 worker.id 不因新 PID 不可预测而永久标为待重启。

## 实施范围

1. 修订本 Intent/Plan，并校准活文档中的配置、容器和受控文件说明。
2. common ENV codec 和文件事务；config 自动字段发现、唯一合并/校验和运行快照。
3. settings Store 端口及 usecase 批量保存/重置；bootstrap 固定 overwrite.env。
4. Proto、HTTP DTO/路由和前端设置页；补齐必要 i18n，生成契约。
5. 聚焦加载优先级、真实值、重启/重置、字符串保留、文件恢复/并发与批量完整校验的后端测试。
6. 运行仓库固定检查，报告实际 diff、检查结果、平台运行缺口；停在 Implementation，未明确要求时不自动进入 SpecFlow Verification。

不修改业务库存、SQL/schema、已有迁移、通用 Compose 模型或部署 writer；不创建真实覆盖文件、不修改用户 dotenv、不启动/停止/重启用户服务器。

## 检查与恢复

工程检查：`task proto`、`yarn --cwd web lint:fix`、`yarn --cwd web typecheck`、`task check`、`go test ./cmd/... ./internal/...`。补充配置行为及可用平台的文件事务测试，不新增复杂 UI 测试。

测试使用临时目录及隔离 ENV，避免固定字段数量断言。新增普通字段的自动发现通过可扩展 fixture 验证。Linux 单文件 bind mount、Linux 文件锁和 Windows 文件锁分别需要平台运行证据，交叉编译不能替代运行证据。

应用无法启动时：停止共享写入者，保留 overwrite.env 与 data/config 恢复材料，先完成 prepared 恢复，再修正对应覆盖项并重新启动。不能通过删除恢复记录跳过半写文件。删除全部覆盖只恢复启动基线，不自动恢复数据库数据或重加密凭据。

## 用户审查

用户已明确：YAML + ENV、固定 overwrite.env、取消定位变量、自动配置发现、复用 controlled_file、业务环境数据排除。用户要求“更新文档，按此实现”，本计划据此 Accepted。

用户指出界面改动过大，并要求移除单元格内的“下次启动值”展示；据此恢复原页面结构和行内操作，将界面变更限制在实际值、原有重启提示与正确保存所需的最小范围。

用户要求复用 ENV 解析类库，并明确“单行 64 KiB 已经很大，没有这么大的写入需求”；据此采用 go-envparse，不保留手写解析器和 2 MiB 值支持测试。
