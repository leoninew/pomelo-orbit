# 组件运行身份、共享挂载与 Orbit 迁移验证
最后修改时间: 2026-10-06 17:16:29

Review status: Draft

Mode: standard

## Intent alignment

依据已接受的 [Intent](../intent/20261006-component-user-group-add.md)，实现 Version 运行身份、Service 继承/覆盖/显式清空、通用共享目录声明及 Orbit 停机迁移说明。Orbit 使用普通应用模型，沿用已有 DooD 容器识别与路径映射。

验证期间用户补充的独立运行身份区域、独立保存接口及两个单行文本框已落实。用户再次要求验证并修复发现的问题；本轮修复失败测试并补足高风险路径的回归覆盖。

## Spec alignment

不适用。标准模式没有单独 Spec，按 Intent 与 Plan 核对。

## Plan alignment

与已接受的 [Plan](../plan/20261006-component-user-group-add.md) 一致：身份字段贯通模型、SQL、HTTP/MCP、Web、定义传递、有效计划、hash 与 Compose；共享属性限绝对目录源，并依据实际运行容器的标签和 bind 判断占用。

Service 根目录、未共享 bind、嵌套私有 bind 和父目录写入仍受保护。共享规则与 worker 或应用名称无关。此前 owner 标记移除已按当前代码同步到迁移说明。

## Actual diff summary

- Version `user` / `group_add`、Service 三态覆盖及增量迁移 `000052`；新增 schema、query 与相应生成代码。
- 独立 `PUT /api/version/:version_id/component/:component_id/identity` 和 MCP `orbit_update_version_component_identity`；Basic 更新的契约与 SQL 不包含身份字段。
- Version mount `shared` 声明、Compose 标签 `io.pomelo-orbit.shared-mounts`、实际挂载与标签匹配及通用目录占用判定。
- Version 身份编辑为 480px 响应式弹窗，两个纵向等高单行输入框，附加组按逗号回填与输入；Service 保留覆盖、清空和恢复继承。
- 活产品/运行时/挂载/MCP 文档与 Tencent 停机迁移、持续部署验收及回退说明。
- 本轮新增 local DooD/SSH 完整目录检查及 CI 派生配置保留测试，修复凭据更新时间测试的系统时钟精度依赖。

## Expected vs actual changed files

| 预期范围 | 实际变更及核对 |
| --- | --- |
| 模型、身份校验、有效配置和渲染 | `internal/model/`、application/service/deployment usecase；符合预期 |
| 三种数据库增量迁移、schema/query、持久化 | `000052`、application/service SQLC Repository；未修改已执行迁移 |
| HTTP/MCP、Proto/DTO、生成代码 | 独立身份接口、Service 附加组存在性 wrapper；生成物与新增字段对应 |
| 定义传递、fork/CI 派生 | 定义创建/普通 fork/CI fork SQLite 集成及接管包 JSON 保留测试 |
| 前端配置和展示 | Version 身份区域/表单、Service 覆盖、shared 编辑、locale/API/helper；符合用户调整 |
| 文档与迁移步骤 | 活文档、迁移指南及同名 Intent/Plan/Verification |
| SQLC 多包 models.go | 公共 schema 变更产生的生成物同步，不是手工扩展其他领域 |
| 凭据更新时间测试 | 用户授权修复验证失败后追加；仅测试夹具，不改变凭据业务实现 |
| ServiceDetail runtime_directory | 此前独立工作，保留用户变更，不计入本任务功能验收 |

## Acceptance criteria

- [x] Version 运行身份保存/读取/清除；身份与 Basic 独立保存且相互保留字段。SQLite 集成、MCP 工具和页面 DOM/API mock 测试通过。
- [x] Service 继承、整体替换、显式清空和恢复继承；SQL NULL/空字符串及 NULL/空数组往返保留，HTTP Proto wrapper 和 Web helper 保留存在性。
- [x] 以 `1000:1000`、`["988"]` 验证合并、Compose YAML 和 hash；渲染代码核对确认空身份不输出相应字段，没有宿主机 UID/GID 默认值。
- [x] 定义创建、普通 fork、CI 派生保留身份和 shared；接管包 JSON 往返保留 Service 显式空覆盖。
- [x] 身份写入校验覆盖用户/主组、附加组与实际支持的格式；前端字段错误关联控件并独立消除。
- [x] SQLite 51 到当前版本迁移保留原有默认身份和非共享语义，回退到 51 成功；MySQL/PostgreSQL 新迁移做结构核对。
- [x] 标签只标记声明为共享的 target；实际 bind source 匹配，未共享/不同 target/嵌套私有 bind/父写入/Service 根目录继续拒绝。
- [x] local DooD 与 Linux SSH 完整目录检查使用各自路径：共享根允许准备子服务目录，移除共享标签后拒绝；原有 Windows/Desktop 路径与映射回归通过。运行时为测试替身，不是实际 Docker/SSH 部署。
- [x] 迁移指南包含绝对 data/logs/.env/socket 挂载、HOME/工作区配置、版本发布、外部 worker、停旧 project、空目录部署、Gateway 运行绑定、Route 同步、持续部署与回退顺序。
- [x] 工程固定检查、前端全量测试和 Go 全量测试通过；过程文档及迁移说明未加入真实凭据值。
- [ ] 实际容器进程身份、socket 权限、真实 Docker DooD 持续部署、Tencent 停机切换及两个域名验收。未执行，属于后续部署验收。
- [ ] 页面视觉及用户真实会话中的保存验收。用户禁止操作其浏览器，保留人工验收。

## Test results

命令从仓库根目录执行；前端命令通过 `--cwd web` 使用现有工具链。

| 命令或证据 | 结果 |
| --- | --- |
| `yarn --cwd web lint:fix` | 前轮 UI 修正后通过；本轮 `task check` 再次验证 lint |
| `yarn --cwd web typecheck` | 通过，包含本轮 `task check` |
| `yarn --cwd web test` | 50 个测试文件、224 项测试通过 |
| `task check` | Web 类型/lint/格式与 Go 格式/静态检查通过，0 issues |
| `go test ./cmd/... ./internal/...` | 修复后全量通过；新增回归后再次通过 |
| `go test ./internal/application/credential/usecase -run TestCredentialServiceTracksUpdatesWithoutChangingCreationTime -count=20` | 20 次通过 |
| `go test ./internal/application/application/usecase ./internal/application/deployment/usecase` | 新增 CI 派生及 local/SSH 共享目录回归通过 |
| `git diff HEAD --check` | 通过 |
| `task sqlc`、Go `buf generate` | 前轮已成功生成并检查；本轮未修改 SQL/Proto |
| TypeScript Proto 生成 | 前轮 Buf 远程插件不可用，使用本地同版本 ts-proto 2.11.8、相同配置生成；不将 `task proto` 全流程记录为通过 |
| 实际 MySQL/PostgreSQL、Docker/SSH、域名及浏览器 | 未验证；包测试通过不等于这些环境验收通过 |

## Issues and fixes

1. 前轮运行身份 UI 使用两列布局、单行与多行混用。已收敛为两个纵向单行控件与紧凑弹窗，按钮为“保存”。附加组响应数据改为逗号回填，避免单行输入剥离换行导致组值合并；页面回填与独立 API 保存测试通过。
2. 凭据更新时间测试要求极短间隔的连续 `time.Now()` 必然严格递增，Windows 上反复失败。测试以明确的过去时间作为持久化基线，分别验证名称和内容更新刷新更新时间、保留创建时间及列表返回值；保留严格更新断言，没有加入 sleep、重试或业务时间兜底。连续 20 次和全量测试通过。
3. 共享规则原有测试主要验证底层挂载判定，CI 身份保留主要靠普通 fork 覆盖。本轮补充完整目录检查的 local DooD/SSH 正向与未标记拒绝回归，以及经过 SQLite 保存再读取的 CI 派生版本配置保留测试。

## Missed or expanded scope

独立身份保存与简化 UI 为用户明确补充的需求，已同步 Intent/Plan。凭据测试修正为本次验证发现问题后的授权修复，可与本功能分开提交。保留原有 runtime_directory 展示变更。

没有新增控制面特例、DooD 路径映射、兼容接口、业务默认 UID/GID 或额外第三方依赖。没有执行远端迁移、库存写入、开发服务器管理或 Git 写操作。用户明确禁止浏览器操作后没有继续相关验证。

## Risks and incomplete items

- 自动化证据覆盖模型/SQLite/Proto/Compose/占用规则和 DOM/API mock，不证明真实 Tencent、Docker 权限、Gateway、域名或停机回退已通过。
- MySQL/PostgreSQL 迁移未用真实数据库执行；后续部署前按目标数据库运行增量迁移并验收。
- 已运行容器需要重新部署才带有共享标签；声明共享不释放 Service 根目录或嵌套私有挂载。
- `988` 只是 Tencent 核查值；迁移时重查 socket GID、进程身份及 data/logs 写权限。
- Orbit 更新自身仍需外部 worker。迁移后须退出临时 worker，用新 Orbit 在线执行其他服务的新部署、重部署和 restart，核对 CI/CD/日志/Route 路径。
- 页面视觉验收由用户完成。Verification 保持 Draft，待用户人工接受。

## Conclusion

实现与已接受的 Intent/Plan 及验证期间用户调整一致。发现的失败测试和回归覆盖缺口已修复，自动化检查通过，可以进行人工验收；真实迁移与部署验收仍按迁移指南后续执行。
