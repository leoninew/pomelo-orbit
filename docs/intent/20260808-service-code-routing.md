# 不可变 Service Code 与组件级网关域名需求
最后修改时间: 2026-08-08 15:16:30

Review status: Accepted

流程模式: 标准 / standard

## Background

当前 `gateway_http` 的 Host 由 `{application_code}.{gateway.base_domain}` 派生。同一 Service 内的所有组件因此共享 Host，只能依赖可选的 `path_prefix` 区分；同一 Application 的多个 Service 也会生成相同 Host。像 MinIO 这样的辅助组件需要独立 HTTP 路由时，这个模型无法表达组件级访问地址。

当前 `Service` 只有可编辑的 `instance_key`，没有稳定的路由身份。`application.code` 同样可以修改，因此二者都不能作为已发布域名的长期标识。

## Goal

1. 为每个 Service 增加不可变的 `service_code`。创建界面以 `<application_code>-<instance_key>` 预填，用户可在创建前修改并提交最终编码。
2. 将 `service_code` 定义为 Service 的稳定外部路由身份：创建后不支持用户编辑，也不因 Application code、instance key、Version 或组件运行时配置变更而重新计算。
3. 将 Service 组件的 `gateway_http` Host 统一派生为 `{component_code}.{service_code}.{base_domain}`；现有模型中的组件 code 指 Version Component 的 `name`。
4. 让 Gateway 的公开暴露列表、客户端连接提示和部署渲染使用同一 Host 派生规则，避免展示地址与实际 Traefik labels 不一致。
5. 在创建时严格校验并原子拒绝重复 `service_code`，不使用自动追加后缀等隐式改写。

## Non-goal

- 不提供创建后修改、重命名、别名、旧域名重定向或新旧 Host 并存能力。
- 不提供创建后修改 `service_code` 的能力；创建请求接收 Application、Version、instance key 与 service code。
- 不新增独立的 Component code 字段；本需求使用现有 Version Component `name` 作为组件 code。
- 不改变 local / host 端口发布语义，不改变平台 `Route` 的 REST provider 路径。
- 不以域名解决无 TLS 的 `gateway_tcp` 分流；该场景仍遵从既有 TCP entrypoint 语义。

## User Scenarios

1. 用户为 Application `ragflow` 以 instance key `default` 创建 Service。创建窗预填 `service_code=ragflow-default`，用户可在提交前修改；系统持久化用户提交的合法且唯一编码。Service 基本信息编辑仍可修改 instance key 与 Version，但 code 保持不变。
2. 该 Service 的组件 `minio` 配置 `gateway_http` 后，部署渲染的 Traefik Host 规则和 Gateway 公开暴露信息均为 `minio.ragflow-default.example.com`。
3. 同一 Service 的组件 `ragflow` 配置 `gateway_http` 后，地址为 `ragflow.ragflow-default.example.com`，与 MinIO 的路由天然隔离，不要求二者通过 path prefix 共用 Host。
4. 用户把 Service 的 instance key 从 `default` 改为 `blue`，既有 `service_code=ragflow-default` 与其公开域名不变；后续创建的其他 Service 才会得到由创建窗默认值或用户覆盖值确定的 code。
5. 创建 Service 时，若提交的 code 不是合法 DNS label、超过长度限制或已被其他 Service 使用，创建失败并在创建窗显示明确校验/冲突错误；系统不截断、不规范化为其他名称，也不自动加序号。
6. 用户查看 Service 列表、详情或基本信息编辑窗时可见 `service_code`；基本信息编辑窗没有可修改控件，更新 API 也不接受该字段。

## Acceptance

- [ ] `Service` 持久化、领域模型、读取投影、Proto/HTTP 响应与 Web 显示均包含只读 `service_code`。
- [ ] Service 创建窗以 `<application_code>-<instance_key>` 预填可编辑 code；创建在持久化前校验提交的 code 必须是小写 DNS label，仅含 `a-z`、`0-9`、`-`，首尾不能是 `-`，长度不超过 63。
- [ ] 数据库以与公开 Host 作用域一致的唯一约束保障 `service_code` 全局唯一；应用层在创建前查询已存在 code，并将常规重复提交映射为明确的业务冲突错误。
- [ ] Service 基本信息更新、应用更新及重新选择 Version 均不得更新 `service_code`；更新契约和前端表单只读展示该字段，不提供可编辑输入。
- [ ] 已存在的 Service 在引入非空唯一字段时得到确定性回填；任何无法生成合法且全局唯一 code 的历史数据必须显式处理，不能以后缀、静默截断或运行时 fallback 掩盖。
- [ ] 每个 `gateway_http` endpoint 的 Docker Router `Host(...)` 规则为 `{component_name}.{service_code}.{gateway.base_domain}`；同一 Service 的不同组件不再共享 Host。
- [ ] 在同一 Host、entrypoint 和 path prefix 下产生重复 `gateway_http` 路由的有效配置被拒绝，避免 Traefik 路由选择不确定。
- [ ] Gateway 公开暴露查询中的 `public_host`、HTTP `client_hint` 与实际渲染 Host 完全一致。
- [ ] TLS `gateway_tcp` 的 SNI Host 也采用 Service/组件级派生规则；无 TLS 的 `HostSNI(*)` 行为保持现状。
- [ ] 部署渲染、Gateway 暴露投影、Service 创建/更新、仓储唯一性和 Web 只读呈现均有回归测试。
- [ ] 活文档更新为 Service 级组件域名规则，并说明两级子域名对证书匹配的影响。

## Open Questions

1. 现有 Application code 可被编辑且其更新校验允许首字符为 `-`，现有 instance key 也没有 DNS-label 校验。历史数据回填仍按二者拼接，实施前需要盘点现有 Service 数据；对不满足回填规则或产生重复 code 的记录，需确认是先人工修正数据再执行迁移，还是在开发库重建后重新创建这些 Service。当前不接受自动改名。
2. 托管 Gateway Dashboard 目前不是普通组件 endpoint，其 Host 单独使用 Application code 推导。本需求是否同时将它迁到 `traefik.<gateway_service_code>.<base_domain>`，需要在计划阶段结合 Gateway 组件约定确认。
3. `{component}.{service}.{base_domain}` 是两层通配标签：TLS 证书 `*.{base_domain}` 不匹配该地址。`letsencrypt` 可按实际 Host 申请证书；手工通配证书需要每个 `*.{service_code}.{base_domain}` 或覆盖精确 SAN。证书发放与 DNS 解析策略需要在实施前确认。

## Decisions

- 用户最新确认：`service_code` 创建后不支持修改；创建窗预填 `<application_code>-<instance_key>`，但允许用户在持久化前修改。
- `service_code` 是路由身份，不是展示名称，也不是可随 `instance_key` 变动的派生显示值。
- 为与单个 active Gateway 的共享 `base_domain` 对齐，`service_code` 采用全局唯一，而不是仅在 Application 或 Project 内唯一。
- Host 派生集中在部署/Gateway 共享的业务辅助逻辑中；渲染器和 Gateway 投影只消费派生结果，不各自拼接字符串。
- 路由规则继续由 Traefik Docker provider labels 下发，不通过平台 `Route`/`providers.rest` 创建业务组件路由。
- 用户确认：Service code 的常规重复提交通过 `ServiceByCode` 预检给出明确冲突；数据库唯一索引仍保证并发安全，但不为罕见的并发穿透引入 SQLite/MySQL 专属错误码或错误文本解析。

## Risk

- 这是数据库、SQLC、领域模型、Service API、部署渲染、Gateway 投影、Proto 和 Web 的跨层变更；Service code 的唯一性必须由数据库约束兜底，不能只依赖先查后写。
- Application code 与 instance key 的现行规则不足以保证组合后始终是合法 DNS label。严格拒绝会使部分创建/迁移暴露已有不规范数据，但这比生成不可预测或不可访问的地址更可靠。
- Component `name` 会随 Version 选择或组件重命名变化，因此组件级 Host 可能发生变化；本需求仅保证 `service_code` 稳定，不为组件 Host 提供兼容别名。
- 两级子域名需要调整 TLS 证书覆盖范围；否则路由虽生成正确，HTTPS 客户端仍可能因证书主机名不匹配而失败。
- 两个并发创建同一 code 的请求可能同时通过预检；唯一索引会拒绝其中一个，但该少见路径只返回通用创建失败，不承诺精确的重复 code 文案。

## User Review Notes

- 2026-08-08：用户要求检查 `gateway_http` 的域名生成和路由，指出一个 Service 可包含 MinIO 等多个需要路由的组件。
- 2026-08-08：用户提出组件级域名形态 `{component_code}.{service_code}.{base_domain}`，并要求 Service 创建时由 application code 与 instance key 填充 code，校验合法且不可重复。
- 2026-08-08：用户最终确认 `service_code` 不支持修改。
- 2026-08-08：用户确认创建 Service 时展示预填编码并允许修改；创建提交校验编码唯一性，基本信息编辑窗只读展示编码。
- 2026-08-08：用户要求开始计划，需求视为接受。
- 2026-08-08：用户接受以 `ServiceByCode` 处理常规重复提交，不保留跨 SQLite/MySQL 精确解析唯一约束错误的逻辑。
