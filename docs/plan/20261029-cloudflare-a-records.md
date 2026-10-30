# Plan / 计划：Cloudflare A 记录管理
最后修改时间: 2026-10-30 11:31:25

## Review status / 审查状态

Accepted

## Intent basis / 意图依据

依据 [意图](../intent/20261029-cloudflare-a-records.md)。用户已明确要求实施核心展示及修改能力，整体 Verification 已执行，结果见同名验证记录。

## Technical decisions / 技术设计与决策

- DNS application/usecase 负责 Project 成员授权、Gateway Token 读取、输入校验与 A 类型限制；通过 port 访问 Cloudflare，bootstrap 注入 infrastructure/external/cloudflare。
- 保持 Token 现有 Gateway 归属，无数据库迁移。列表/编辑均不回传 Token，不依赖 Docker/SSH 或 Traefik 状态。
- Cloudflare GET /zones 列出所有可访问 Zone；GET /zones/{zone}/dns_records?type=A 分页；保存前 GET 单条核对 A 类型，再以原名称、type=A 和 content/ttl/proxied 进行 PATCH；名称和类型使用读取值，省略备注、标签等未编辑字段。
- HTTP 采用单数路径 /api/dns/zone、/api/dns/zone/:zone_id/record、/api/dns/zone/:zone_id/record/:record_id；DTO 由 proto 生成。DNS 路由在请求写事务中间件前注册，外部 HTTP 请求不持有写事务。
- MCP 增加 Zone/A 列表及修改工具，按当前已选项目调用同一用例，不开启数据库写事务。
- Cloudflare 请求固定基址、短超时、不跟随重定向、不自动重试；外部错误使用固定的可理解信息，不输出原始错误体或 Authorization。
- IPv4 用 net/netip 校验；TTL 支持 1（自动）或 30–86400 秒，30 秒可否使用由 Cloudflare Zone 套餐判定；代理开启时使用自动 TTL。
- 前端使用现有 AppInput、SelectControl、AppDialog、ListPagination 等共享组件与当前字体规范；名称只读。请求以项目、Zone 和序号隔离，保存期间锁定表单并防重复提交。

## Implementation steps / 实现步骤

1. 完成 DNS 端口、用例和 Cloudflare 适配器及聚焦测试。
2. 接入 proto、HTTP 和 MCP；核对鉴权与事务边界。
3. 新增 DNS 页面和中英文本，接入网络接入菜单；补充表单规则轻量单测。
4. 执行前端 lint/typecheck/test、task check 与 Go 测试，检查 diff；记录实际结果与偏差。

## Files to change / 需修改文件

- internal/application/dns/{dto,port,usecase}、internal/infrastructure/external/cloudflare。
- proto/orbit/v1/dns、对应 Go/TS 生成文件、internal/api/http/{handler/dns,routes}。
- internal/bootstrap/http.go、internal/api/mcp/delivery。
- web/src/{api/dns,views/dns,navigation.ts,router/index.ts,i18n/locales}。
- 本功能意图与计划；必要时补充 docs/README.md 的功能约定。

## Tasks and coordination / 子任务与协调

不适用，主会话顺序实施同一纵向结果。

## Verification plan / 验证计划

- 用例：成员隔离、Token 缺失、无效 IPv4/TTL、非 A 拒绝、正常列表及修改。
- 外部协议：Authorization、type=A/分页、PATCH 字段保留、权限/限流/服务故障、不泄漏 Token。
- 前端：纯表单规则和导航测试；不新增整页或端到端测试。
- 运行 corepack yarn lint:fix、corepack yarn typecheck、前端轻量测试、task check、go test ./cmd/... ./internal/...；仅在现有服务可用时进行浏览器开发检查，不启动或重启服务器。
- 本阶段执行必要开发检查，整体 Verification 按 SpecFlow 阶段授权推进。

## Risks / blockers / assumptions / 风险 / 阻塞项 / 假设

继承意图中的 Token 权限、共享 Zone、远端并行修改及 DNS 传播风险。真实 Cloudflare 写入需要用户指定目标并授权；本次通过适配器测试验证协议，不修改用户 DNS。

## Rollback / recovery / 回滚 / 恢复

无持久化结构变更；移除新增入口及装配可回退功能。已提交给 Cloudflare 的修改不随代码回退，需要用原值再次显式修改；不自动回滚外部记录。

## Implementation notes / 实施记录

以下为 2026-10-29 首版实施记录，后续范围调整见文末；尚未进入独立整体 Verification 阶段。

- 交付：网络接入 DNS 页面、Zone 读取、A 记录分页、编辑 IPv4/TTL/代理状态；HTTP 与三个 MCP 工具共用 DNS 用例。名称只读，未增加新建/删除或其他记录类型。
- 架构核对：应用通过窄端口访问 Project、Gateway 存储与 Cloudflare；Cloudflare 适配器不承载成员权限或业务规则；bootstrap 完成注入；HTTP/MCP 外部写入不持有数据库写事务。
- 凭据与写入：DNS 响应不包含 Token；仅后端发送 Authorization；Cloudflare PATCH 带原名称和固定 A 类型以满足官方请求契约，备注、标签等未编辑元数据不发送。
- 范围与差异：按计划新增域、协议、页面和轻量测试；具体外部适配器落在既有 external 目录。无迁移、依赖、证书配置语义或凭据归属变更。
- 前端：corepack yarn lint:fix、corepack yarn typecheck 通过；corepack yarn test --reporter=default --silent=false --no-color 通过（43 个文件、189 个用例），未出现测试警告。
- 项目检查：task check 通过，Go lint 0 issues；go test ./cmd/... ./internal/... 通过；git diff --check 通过。
- 关键覆盖：成员权限与 Token 缺失、IPv4/TTL、非 A 拒绝、代理自动 TTL、Cloudflare Zone 分页与 A 过滤、PATCH 字段保持、权限/限流/失败脱敏、不跟随重定向、不自动重试、HTTP 认证与显式 false、MCP 项目和事务边界、前端请求与表单规则。
- 工具情况：task proto 首次受 Windows 目录占用影响，使用原配置去除 clean 的临时模板生成成功；所有既有生成文件保持无差异，仓库生成配置未改动。首次 task check 发现 HTTP Body.Close 的 errcheck 问题，修正后完整检查通过。
- 未执行：当前浏览器连接被占用，替代浏览器连接无可用实例，因此未完成浏览器视觉/交互实测。现有前端服务 HTTP 200；未启动或重启服务。未使用真实 Token 做 Cloudflare 联调或修改真实 DNS。
- 遗留限制：仍沿用 Gateway 的 ACME Token 保存/清空语义；可见 Zone 依 Token 授权，多个环境可共享；远端并行修改无条件写入保证；超时后先刷新核对再重试；保存成功不代表 DNS 传播完成。

官方接口依据：[Zone 列表](https://developers.cloudflare.com/api/resources/zones/methods/list/)、[DNS 列表](https://developers.cloudflare.com/api/resources/dns/subresources/records/methods/list/)、[修改 DNS](https://developers.cloudflare.com/api/resources/dns/subresources/records/methods/edit/)。

## User review notes / 用户审查记录

- 2026-10-29：沿用「先实现核心的 A 记录展示和修改能力」的实施授权，未要求新增/删除记录或 Git 写入。

## 后续变更：新增解析与搜索

- Zone 使用可搜索下拉框；记录名称搜索通过 Cloudflare `name.contains` 与分页配合。
- 通过独立创建模态窗新增 A 记录；HTTP 与 MCP 复用应用用例，保持原有权限、Token 和事务边界。
- 新增记录直接写入 Cloudflare；失败后先搜索核对，不自动重试。

- 操作列增加删除及独立确认框；HTTP 和 MCP 调用同一用例，删除前核对记录类型。

- 应用列表及 DNS 列表新增按钮统一为服务列表样式；操作列按钮间距与现有列表一致。
