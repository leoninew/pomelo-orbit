# 系统设置实际配置读取与持久化修改验证
最后修改时间: 2026-10-09 19:55:36

Review status: Draft

Mode: standard

当前阶段：验证 / Verification。依据 [Intent](../intent/20261006-settings-persistent-overrides.md) 与 [Plan](../plan/20261006-settings-persistent-overrides.md)。用户反馈“测试了没大问题”，并明确要求完成本验证文档；后续要求复用解析类库，并接受其约 64 KiB 单行上限，本轮已落实并重新检查。

## 意图与计划对齐

本任务实现进程配置的实际值读取、持久化修改和重启后生效。场景为原生 Windows、原生 Linux 和容器运行，业务数据库中的环境与部署记录不参与评估。

- 保留 YAML + ENV；固定读取启动工作目录的 `overwrite.env`，没有新增 JSON 配置文件或文件定位变量。
- 加载优先级为默认 YAML < profile YAML < 启动 dotenv < 原始进程 ENV < overwrite.env。文件加载不修改 OS ENV。
- typed Config 的 mapstructure 标签自动提供字段、类型及 ENV 映射；设置列表与更新校验复用同一份字段定义。新增现有类型字段无需维护额外名单。
- 系统设置返回实际运行快照、启动基线、保存覆盖、下一配置及服务端待重启状态；批量更新与重置使用完整配置校验和 revision 冲突检查。
- 所有配置统一可编辑，secret 标记只控制显示；修改数据库和 JWT 不隐含迁移数据或重新加密凭据。
- 容器使用现有可写 `controlled_file` 与 `ignore_if_exists=true`，同时持久化 data 目录。后端 Compose renderer 的 `env_file` 尚未实现，本次没有扩展它。
- 保存采用跨进程文件锁、prepared/committed 恢复记录和原地写入，保留单文件 bind mount 的 inode。
- 覆盖文件使用 `hashicorp/go-envparse v0.1.0` 解析，不进行变量插值；codec 只保留键名约束、BOM 处理、稳定排序和标准库字符串转义。保存前确认编码结果能被同一类库读取。
- 页面按用户后续要求保留原四列表格、搜索与行内文字操作，移除新增的“下次启动值”说明；待重启状态使用原有顶部提示。

标准模式没有独立 Spec 文档；规格对齐按已接受的 Intent、Plan 及后续用户指令核对。

## 实际变更

核对范围包括用户已暂存的 37 个功能文件，以及后续解析器替换、保存前解析检查、Go 依赖、相应测试与文档修订。用户原有暂存状态保留，没有执行暂存或提交；最终提交应包含工作区中的这些追加修改。

| 范围 | 预期与实际 |
| --- | --- |
| 配置加载 | `internal/config/config.go`、新增 `fields.go` 与 `runtime.go`；自动发现字段、合并规则、校验和运行快照，配套配置测试 |
| ENV 存储 | 新增 `internal/common/envfile` codec、Windows/Linux 锁与文件事务；原 infrastructure Store 改为适配器，旧存储测试由 common 测试替代 |
| 设置应用层 | DTO、Store 端口、usecase 与集成测试；批量保存、批量重置、revision、待重启状态 |
| HTTP 契约 | settings handler、mapper、单数路由、Proto 和生成 Go/TS；保留权限与错误契约 |
| 启动装配 | `internal/bootstrap/http.go` 仅调整设置服务构造及固定覆盖文件 Store |
| 前端 | Settings.vue、API、生成类型及中英文文案；原布局内实现类型编辑、批量草稿、错误反馈与后端状态 |
| 配置与活文档 | `.env.example`、`.gitignore`、默认 YAML、后端架构、Docker、卷挂载、MCP 指南及 Intent/Plan |
| 解析依赖 | `go.mod`、`go.sum` 新增 go-envparse 直接依赖，没有更新其他模块版本 |

没有业务数据库、SQL、迁移、通用 Compose 模型或部署 writer 的变更。启动 dotenv 仍使用已有 `github.com/joho/godotenv v1.5.1`；固定覆盖文件使用 go-envparse。

## 验收清单

| 验收项 | 结果与依据 |
| --- | --- |
| 真实 ENV 值及统一类型 | 通过；`TestActualValuesSaveRefreshRestartAndReset` 及配置加载测试 |
| 所有普通配置与敏感项可编辑 | 通过；`TestAllConfigurationTypesAndSecretsAreEditable` 覆盖数据库、JWT、MCP、布尔、整数、列表及 duration |
| 保存不改变当前值，刷新保留待重启 | 通过；上述 settings 集成测试验证保存、重新读取和重新加载配置 |
| 固定覆盖文件且只保存修改项 | 通过；Store 与加载测试，启动 dotenv 和原始进程 ENV 保持原值 |
| 字段自动发现 | 通过；`TestFieldDiscoveryIncludesNewNestedFieldsWithoutRegistration` 使用新增嵌套字段 fixture，不依赖固定字段数量 |
| 候选配置完整校验与批量原子操作 | 通过；`TestBatchValidationAndResetAreAtomic` 验证 LLM 关联字段及拒绝写入；磁盘写入错误路径经代码核对，没有模拟真实磁盘故障 |
| 字符串保存与解析 | 通过；类库往返测试覆盖空串、前导零、美元符号、引号、Windows 路径、换行、控制字符及 Unicode；`TestStoreRejectsOversizedAssignmentWithoutChangingFile` 验证超限不修改原文件 |
| 并发、恢复及无关字段保留 | 通过；子进程并发测试、prepared/committed 恢复测试、拒绝修改保留原内容测试；不等同于真实断电测试 |
| 重启覆盖与重置恢复基线 | 通过；settings 集成测试及 `TestOverwritePrecedenceResetAndEnvironmentIsolation` |
| 受控文件后续保留 | 通过；既有 `TestMaterializeControlledFileRespectsIgnoreIfExists` 验证初始化与后续跳过，不宣称执行过完整生产重新部署 |
| HTTP/worker/MCP 统一入口 | 代码核对通过；入口共用 `config.Load`，已有进程仍需分别重启 |
| Windows/Linux/容器场景 | 已取得隔离文件事务运行证据；文件位置、权限、可写挂载与恢复方式已写入活文档 |
| 原页面与权限边界 | 代码核对及前端固定检查通过；用户反馈人工测试无大问题，未细分其测试场景 |

## 检查证据

| 检查 | 结果与范围 |
| --- | --- |
| `task check` | 本验证阶段通过；包含前端 typecheck、lint、format 及 Go 格式与 lint，0 issues |
| `go test ./cmd/... ./internal/...` | 本验证阶段通过；包括配置、设置、ENV 事务及既有后端测试 |
| `git diff --cached --check`、`git diff --check` | 本验证阶段通过 |
| `task proto` | 实现阶段已通过，生成 Go/TS 纳入本次 diff；本轮未重新生成 |
| `yarn --cwd web lint:fix`、`yarn --cwd web typecheck` | 实现阶段已通过；本轮 task check 再次检查前端，未改页面 |
| Windows 配置/settings/ENV Store 测试 | 实现阶段与本轮后端测试通过 |
| WSL Ubuntu 配置及 ENV 事务测试 | 本轮替换解析器后通过；使用当前 Windows Go 工具链交叉编译测试程序，并在 WSL Ubuntu 实际执行。验证类库往返、配置加载、文件锁、恢复、并发和超限保留原文件；挂载专属测试在此场景跳过 |
| Linux 实际单文件 bind mount | 实现阶段使用隔离 Alpine 容器、UID/GID 1000、`/app/overwrite.env` 可写文件挂载及持久化 `/app/data` 通过；验证 inode 保留和宿主文件同步观察 |
| 用户人工测试 | 用户明确反馈“测试了没大问题”；没有具体操作清单，因此不推测其覆盖了所有场景 |

用户要求删除的 `web/src/views/settings/Settings.test.ts` 不存在。历史 UI 测试结果不作为当前持续覆盖的证明；本轮没有增加 UI 测试，也没有操作用户浏览器、开发服务器或业务数据库。

## 范围偏差

前期实现曾扩大设置页面展示，后续按用户要求恢复原结构并删除额外的下次值说明。后端仍提供覆盖与下一值元数据供正确编辑和状态判断，但不据此增加页面说明区域。

新增文件锁与写入恢复机制对应计划中的 Linux 单文件挂载、并发写入及进程中断风险。受控文件首次初始化沿用已有能力，没有新增专用于 Orbit 的部署逻辑。

用户指出手写 ENV 解析器没有复用类库，本轮已改用 go-envparse。用户明确没有大值写入需求，因此移除 2 MiB 值支持测试，保留正常值往返与超限保存不损坏原文件的必要验证。语法遵循所选类库，重复键取最后值，没有保留自定义重复键拒绝逻辑。

## 风险与限制

- 覆盖文件及 `<启动工作目录>/data/config` 必须可写且持久化；共享文件的进程必须共享恢复目录。手工改文件前应停止相关写入进程。
- 保存或重置不热更新运行资源；独立 HTTP、worker 与 MCP 进程分别重启后加载覆盖。预览使用固定原始 ENV 与当前基线文件，不预测未来部署 ENV。
- 保存校验不验证新的数据库可连接、外部服务可用或 JWT 能解密既有凭据；修改相关值的实际影响由使用者负责，不附带数据迁移或重新加密。
- 恢复测试验证标记状态与进程合作写入，不保证硬件损坏、真实断电或不支持操作系统文件锁的文件系统行为。
- Linux 容器验证为隔离文件事务验证，没有执行用户运行实例的升级或完整生产部署。
- go-envparse 单行上限约 64 KiB，计算对象是包含键名和字符串转义的整行；用户已接受。启动时手工写入的超限行会失败，设置保存则在写目标文件前拒绝超限内容。

## 未完成事项

没有阻碍本任务交付的未完成事项。解析器复用已落实，Windows 固定检查与 Linux 相关测试已通过。当前未追加真实断电、生产重新部署或浏览器自动化测试；这些证据边界已在上文注明。

## 结论

本任务的实现与已接受意图、计划及后续用户指令一致，主要功能验收、仓库固定检查及 Windows/Linux 相关运行验证通过。用户已提供人工测试无大问题的反馈，解析类库复用问题也已解决。验证文档完成，Review status 保持 Draft 供用户审阅；没有执行 Git 提交或操作用户运行实例。
