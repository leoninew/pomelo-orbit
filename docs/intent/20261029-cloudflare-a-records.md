# Intent / 意图：Cloudflare A 记录管理
最后修改时间: 2026-10-30 11:31:25

## Review status / 审查状态

Accepted

## Background / 背景

环境通过 Gateway 保存 Cloudflare DNS API Token，当前仅用于 Traefik DNS-01 证书签发。「网络接入」已有网关、自定义路由和 Traefik 路由入口，缺少 DNS 记录管理。用户要求先实现核心 A 记录展示和修改能力。

## Goals / 目标

- 在网络接入新增 DNS 页面，使用当前 Project 对应环境绑定 Gateway 的 Token。
- 列出 Token 可访问的 Zone，分页展示所选 Zone 的 A 记录。
- 搜索、新增和删除 A 记录，并修改已有 A 记录的 IPv4 地址、TTL、Cloudflare 代理状态，展示保存结果。
- 前端与 MCP 使用同一应用用例，校验 Project 成员权限，凭据不进入 DNS 响应。

## Non-goals / 非目标

- 批量修改、其他记录类型、自动关联 Route 或部署、导入导出。
- 独立 DNS 凭据体系、迁移 Token 归属、DNS 本地镜像、传播完成检测。

## User stories / scenarios / 用户场景

- 用户进入网络接入 → DNS，选择可访问的 Zone，查看 A 记录及当前地址、TTL、代理状态。
- 编辑一条已有记录并保存；失败保留表单，成功显示 Cloudflare 返回的实际值。
- 未配置 Token、权限不足或 Cloudflare 不可用时，显示明确提示；切换项目不保留前一项目记录。

## Acceptance criteria / 验收标准

- A01：菜单和路由可访问；加载、无 Token、空记录和失败状态可区分。
- A02：只展示 A 记录，支持 Zone 切换与分页，切换项目/Zone 时隔离旧请求结果。
- A03：只修改已有 A 记录的地址、TTL、代理状态；IPv4/TTL 本地与服务端校验；保留名称、类型、备注、标签等其他字段。
- A04：非成员不能调用 Cloudflare；凭据只由后端使用；修改不持有长数据库事务，不要求 Gateway 容器运行。
- A05：HTTP/MCP 共用用例；必要的规则、外部协议和轻量前端测试及项目检查通过。
- A06：支持搜索、新增和确认后删除 A 记录；新增与编辑使用独立模态窗，列表按钮和间距遵循现有风格。

## Workload assessment / 工作量评估

工作量中等：新增 DNS 用例、Cloudflare HTTP 适配器、HTTP/MCP 入口和单个前端页面，复用 Gateway 存储，无迁移。复杂度主要在权限、分页、外部失败及项目切换状态隔离。风险是 Token 权限范围、外部记录被并行修改和远端请求结果不确定性。一个纵向交付结果、接口耦合紧密，由主会话顺序完成，保持单任务。

## Candidate tasks and dependencies / 候选子任务与依赖

不适用；不创建 task。

## Open questions / 待定问题

暂无需要用户确认的未决事项。新增 A 记录时填写名称；编辑已有记录时名称只读。

## Decisions / 决策

- Cloudflare 官方 API 已确认支持 Zone 列表、按 type=A 的记录分页和 PATCH 部分修改。
- 复用现有 Gateway Token，不扩展其配置入口或 ACME profile 行为。
- Token 所有者须授予目标 Zone 的 Zone Read 与 DNS Edit 权限；Token 范围可能覆盖多个环境共享的 Zone，页面展示所选 Token 实际可见范围。

## Risks / assumptions / 风险 / 假设

- 当前只有 DNS ACME profile 保存 Token，改用 HTTP 或关闭 ACME 会清空该值；首版明确提示缺失，不改变证书设置语义。
- 不承诺条件写入和自动重试；PATCH 使用读取到的名称和 A 类型，仅改变所选编辑字段；失败后用户刷新核对，避免盲目重试写入。
- Cloudflare 接受不等于公共 DNS 已完成传播；代理启用时 TTL 为自动。

## User review notes / 用户审查记录

- 2026-10-29：用户先要求可行性调查，后明确要求「先实现核心的 A 记录展示和修改能力」，授权补齐记录并直接实施。

- 2026-10-30：用户要求增加 A 记录创建、名称搜索及 DNS 页面工具栏调整；创建和编辑使用独立模态窗。
- 2026-10-30：用户要求在操作列增加删除按钮，并在确认后删除 A 记录。

- 2026-10-30：用户确认功能有效并授权整体验收。
