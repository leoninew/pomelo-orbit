# Verification / 验证：环境主机身份以稳定指纹固定
最后修改时间: 2026-11-11 17:54:54

## Review status / 审查状态

Draft

## Intent alignment / 意图对齐

依据 [意图](../intent/20261111-environment-host-identity-pin.md) A01–A09 核对。只新增 `host_identity_policy`，两种策略均复用 `host_key_fingerprint`；默认保留协商对象指纹，稳定模式取证书内公钥指纹。首次记录、权限、受管凭据、探测诊断和目标修订规则沿现有流程。

用户在 Implementation 后要求“主机身份校验”与类型同行，并将下方长说明改为标题旁问号 Tooltip。本次一并核对最终共享组件与中英文文案。

## Plan alignment / 计划对齐

依据 [计划](../plan/20261111-environment-host-identity-pin.md)。用户于 2026-11-11 明确要求以 `best-practices:dev-check` 和 `specflow:specflow` 完成验证，授权本阶段。工作基线为 `HEAD 1498b88a` 上当前工作树；核对范围包含暂存与未暂存内容，不改动用户索引。

输入定义、普通更新、完整快照恢复、两条实际 SSH 握手入口、三库迁移、HTTP/MCP 与共享表单均按计划落地。验证阶段使用项目现有只读检查入口，不执行格式修复、依赖安装、生成命令、服务启动或外部数据库写入。

## Task deliveries and integration / 子任务交付与集成

不适用（单任务，由主会话实施及核对）。

## Actual diff summary / 实际差异摘要

- 环境模型、DTO、SQLC 查询与仓储读写携带策略；创建缺省值在应用边界赋予，更新省略字段时保留原策略，无内部空值兼容逻辑。
- 三个普通保存入口通过共享 target/identity 判断处理策略切换：清空 pin、递增修订，凭据不变，保存不连接 SSH；快照恢复保留包内策略、指纹、修订及探测记录。
- runner 内集中提取指纹，Probe 和 Runtime 的 strict/首次观察回调均使用策略；终端、命令、转发、会话及其余执行入口复用 Runtime。
- HTTP 初始化/编辑及 MCP 输入输出完整传递字段；项目交接包格式为 3。共享表单包含选择、回填、请求转换、dirty 判断及简短问号提示。
- 新增三库 `000057` up/down，仅新增/删除策略列；更新环境、升级、项目交接手册。数据库合并脚本同步把策略纳入目标比较。

## Expected vs actual changed files / 预期与实际修改文件对比

验证开始时，相对基线共有 79 个文件变化：`docs/` 5、`internal/` 55、`proto/` 1、`scripts/` 2、`sql/` 9、`web/` 7；其中包括生成代码与测试夹具。本阶段另新增本报告，并在计划中记录授权和结论。

| 计划范围 | 实际落点与核对 |
| --- | --- |
| 模型、用例与仓储 | `internal/model/environment.go`、环境 DTO/usecase、环境仓储及相关测试，符合预期 |
| SSH 算法与实际握手 | `environment_probe.go`、`runtime.go`；回调和本地协议测试覆盖两条入口 |
| 迁移、schema、SQLC | 三库共六个新迁移，两个 schema，环境查询及相应生成物，未修改历史迁移 |
| HTTP、MCP、proto | environment/project_initialization mapper、MCP 环境工具和输出 mapper、共享 proto 与 Go/TS 生成物，符合预期 |
| 项目交接 | 格式常量、codec 往返、服务导入及环境快照恢复测试，符合预期 |
| 前端 | 共享字段与表单工具/测试、详情、两份语言包，包含用户后续布局与 Tooltip 调整 |
| 活文档 | 环境、项目交接与升级手册，以及本功能过程文档，符合预期 |
| 预期之外的必要同步 | 数据库合并脚本与原有单测：防止不同策略复用探测状态，已在计划登记 |
| 预期检查但无需修改 | `ssh_initialization.go`、CLI、部署/流水线目标和修订快照直接复用共享模型/Runtime，无需新增生产接口 |

`.openidea/` 为已有未跟踪目录，不属于交付范围。

## Acceptance criteria checklist / 验收标准检查清单

| 标准 | 结果与证据 |
| --- | --- |
| A01 默认与升级保持行为 | SQLite 历史升级测试从 56 迁移至当前，原 SSH pin/revision 保留，local/SSH 策略均为 negotiated；初始化和生成命令测试断言创建默认值 |
| A02 单一指纹存储 | model、DTO、SQL、三库新增迁移均只增加策略；指纹仍由已有字段保存 |
| A03 重签与默认严格比较 | `TestHostIdentityCallbacksHandleCertificateRenewalAndKeyChange` 生成同 key 的不同证书，stable 接受重签及裸 key，negotiated 拒绝；普通 key 的 SHA256 结果相同 |
| A04 换 key 拒绝 | strict 与首次观察回调均拒绝不同 key 的证书；既有普通 key 回调测试保留换 key 拒绝和同连接 rekey 固定值不被替换的断言 |
| A05 各入口一致与部署前置 | `TestProbeAndRuntimeUseHostIdentityPolicyDuringHandshake` 以本地 SSH 服务器验证 Probe、Runtime strict/observation 和首次记录；终端/命令/转发直接调用该 Runtime，TargetResolver 的空 pin 拒绝规则保留 |
| A06 三库迁移 | 三库 up/down、默认值与约束静态核对；迁移文件对齐检查及 SQLite up/down、非法值约束通过。MySQL/PostgreSQL 真库执行未启用，不能据此声称三库实测通过 |
| A07 契约、MCP 与交接 | HTTP 两组 mapper 保留省略/空串/stable 区别；MCP 协议测试实际调用 get/update，断言输出、字段省略与 stable 输入；包 codec/服务导入测试保持策略和 pin，格式 2 导入被拒绝；SQLite 快照用例验证完整状态持久化 |
| A08 生命周期与回归 | 三个保存入口覆盖省略、相同、双向切换，检查 pin/revision、凭据、探测历史及不调用 prober；仓储测试拒绝旧修订观察写入；前端轻量测试覆盖默认、回填、两种请求和 dirty |
| A09 项目质量入口 | 已执行下列检查，运行结果在命令表记录；外部目标沿既有显式开关跳过 |

## Export/import and MCP / 导出导入与 MCP 专项核对

| 用户关注的入口 | 核对结论 |
| --- | --- |
| 导出交接包 | `ExportDocument → Export → TargetDefinitionForUser → configurationFromEnvironment` 读取完整策略与 pin；codec 输出 `environment.ssh.host_identity_policy`，当前格式为 3 |
| 导入并覆盖环境 | `ImportDocument → Import → SaveTargetDefinitionForHandover → environmentFromConfiguration` 恢复完整快照，不套用普通编辑清空规则；原有密钥解密/新凭据归属流程保留 |
| 导入不覆盖环境 | 仍跳过环境和凭据恢复，保留目标配置；现有 `TestImportDocumentSkipsEnvironmentAndCredentialWhenNotOverridden` 覆盖该边界 |
| MCP 读取 | `orbit_get_project_environment` 的 `ssh` 输出包含 `host_identity_policy` 与现有 pin |
| MCP 修改 | `orbit_update_project_environment` 的嵌套 `ssh.host_identity_policy` 可选，支持 negotiated/stable；省略保留当前值，非法值由共享用例拒绝。更新 SSH 对象仍需提交其完整配置字段 |
| MCP 探测 | `orbit_probe_project_environment` 调用同一个环境 Probe 用例，runner 按已存策略验证指纹 |

本次核对的是交接包的环境字段和 MCP 环境工具；未新增独立的 MCP 包导入/导出工具。上述两类既定入口已经具备新增策略，无需额外补生产代码。

## Test / command results / 测试 / 命令结果

工具链依据根 `Taskfile.yml`、`web/package.json`、`go.mod`、项目现有 lint 配置及 Python unittest 脚本。

| 工作目录与命令 | 结果 |
| --- | --- |
| 根：`task check` | 全部通过：前端类型、lint、Prettier、Go 格式和 lint，`0 issues` |
| 根：`task test` | 通过：前端 43 个文件/194 个测试，Go 核心 unit tag 入口通过（Go 结果包含缓存，不作计时依据） |
| 根：`go test -count=1 ./cmd/... ./internal/... ./sql` | 通过，包含本地 SQLite、SSH 协议、交接包、HTTP/MCP 及其余默认启用范围 |
| `web/`：`corepack pnpm test --reporter=default --silent=false --no-color` | 43 个文件、194 个测试通过，无 warning/stderr |
| 根：`python -m unittest discover -s scripts -p test_merge_orbit_databases.py` | 18 个测试通过 |
| 根：`git diff HEAD --check` | 通过，覆盖用户暂存和未暂存差异 |
| 根：`go test -v ./internal/test/e2e -run 'Test(MySQL\|PostgreSQL)MigrationE2E\|TestWindowsSSH'` | 6 个外部联调用例均按既有开关跳过，命令成功；未启用 MySQL/PostgreSQL 配置或 Windows OpenSSH 开关 |

`task sqlc`、`task proto`、前端 `lint:fix` 和格式修复已在 Implementation 执行并记录于计划，本阶段不重跑会改写文件的命令。未执行镜像构建、部署、浏览器 E2E 或真实代理连接，这些不属于计划的本地检查范围。

## Missed or expanded scope / 遗漏或扩大的范围

未发现既定功能遗漏。数据库合并脚本属于必要字段同步，已登记；后续 UI 变更来自用户明确要求。验证阶段未修改产品代码、测试、依赖、配置、真实项目或远端系统。

## Risks / 风险

- 稳定模式固定公钥，保留原有 TOFU；不建立 CA 信任，不新增证书有效期/主机名验证。底层 key 变化仍拒绝连接。
- 切换策略需要重新建立 pin，不能用首次新记录证明历史主机连续性；旧故障项目未自动切换，也未进行线上修复或部署。
- MySQL/PostgreSQL DDL 只有源码核对，真实实例尚未执行；真实 SSH 代理证书轮换联调未运行。
- 交接格式 3 拒绝旧包；降级前需切回 negotiated 并重新记录指纹，按手册保留匹配版本备份。

## Incomplete items / 未完成事项

本次实现与计划内本地验证没有未完成项。外部数据库与真实 SSH 代理联调为明确未运行项，不以本地通过替代。

## Conclusion / 结论

整体验证完成，在计划约定的本地检查范围内通过，未发现阻塞交付的问题。交接包导出、导入和 MCP 环境读写/探测的策略链路已具备，相关测试通过，无需额外补生产代码。A06 的三库结论明确限于源码核对和 SQLite 执行证据；MySQL/PostgreSQL 与真实代理联调仍未实测。
