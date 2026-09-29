# Gateway 内外网域名与自定义路由域名辅助
最后修改时间: 2026-09-29 10:02:47

Review status: Accepted

Flow mode: standard

## Background

变更前 GatewayConfig 只有 `base_domain`：使用 `gateway` endpoint mode 的 Service Component 会据此生成 Docker label 域名，Gateway 自身相关地址也使用该值。自定义 Route 的域名独立保存为完整字符串；创建表单此前以 `base_domain` 预填域名。Route 以 `name` 表达用户所说的“路由编码”，创建页、列表编辑弹窗和详情编辑弹窗都有独立的域名控件。

## Goal

- 将现有 Gateway 域名概念和字段统一命名为 `internal_domain`，保持其当前的组件派生域名与 Docker label 语义。
- 在 GatewayConfig 增加可选的 `external_domain`，仅作为自定义 Route 表单的域名拼接后缀；配置它不创建 Gateway 对外路由或 DNS 记录。
- 用户配置 `external_domain=example.com` 后，在 Route 创建或编辑表单填写编码 `xx`，表单自动将 `xx.example.com` 填入现有域名控件；若域名已手工改为其他值，后续编码变化不覆盖该值。Route 仍保存完整域名。
- 将 Route 编码（现有 `name` 字段）的输入与服务端校验统一收紧为合法的小写 DNS 单标签，确保拼接后不会因编码格式产生非法域名。
- 让 Gateway 初始化、配置展示与编辑，以及相关 API、MCP、导入导出和配置入口使用一致的 `internal_domain` / `external_domain` 术语。
- 通过新增数据库迁移将现有 `base_domain` 数据转入 `internal_domain`，并为现有 Gateway 留空 `external_domain`；产品代码直接使用新字段。

## Non-goal

- 不管理、创建或校验 DNS 记录，也不保证生成的域名已指向 Gateway。
- 不把 `external_domain` 用于 `gateway` mode 的 Docker labels，不改变现有内网域名的部署行为。
- 不因修改 Gateway 的外网域名、打开 Route 编辑表单或读取 Route 而自动重写已有域名；只有用户在表单中修改编码时才可能更新域名控件。
- 不改变 HTTP Host/PathPrefix、TCP 端口路由、证书方式或 Traefik 全量同步机制。
- 不在业务逻辑中保留 `base_domain` 字段、旧配置键、别名、回退或运行时数据修复。
- 不处理或验收存量不符合新编码规则的 Route；所需数据更新由部署前的离线 SQL 执行。

## User scenarios

1. 已有 Gateway 升级后，其原 `base_domain` 成为 `internal_domain`；原有 `gateway` mode 服务继续使用相同的 Docker label 域名。
2. 用户在 Gateway 中填写 `internal_domain=internal.example.test` 和 `external_domain=example.com`，创建 HTTP Route 时在现有编码控件输入 `xx`，现有域名控件随之填入 `xx.example.com`，保存后的 Route 域名也是该完整值。
3. 用户从列表或详情打开已有 Route 的编辑表单，首先看到已保存的域名；若该域名等于修改前的 `编码.external_domain`，修改编码时域名控件随之更新。若域名是用户手填的其他值，修改编码不覆盖该值。
4. 用户创建或编辑 TCP Route 时获得同样的域名控件联动；TCP 的转发仍由独占监听端口决定，域名不参与 SNI 分流。
5. 未配置 `external_domain` 时，Route 创建表单的域名控件保持空白，用户手工填写完整 Route 域名；已保存 Route 的读取、编辑和发布不依赖该 Gateway 字段。
6. 用户修改或清空 Gateway 的 `external_domain`，已有 Route 的域名保持不变，除非用户随后在 Route 表单中主动修改相关字段并保存。

## Acceptance

- [ ] Gateway 配置、初始化入口和对外契约统一使用 `internal_domain` 与可选 `external_domain`；`internal_domain` 保持必填。
- [ ] Gateway 初始化与编辑表单中，内网域名和外网域名各占一整行；所有 Gateway 创建和编辑入口均传递新字段，包括 MCP 更新入口。
- [ ] 新迁移在受支持数据库中保留既有 `base_domain` 值并转为 `internal_domain`，为既有 Gateway 建立空的 `external_domain`；不改写已执行迁移。
- [ ] 业务代码、API/Proto、配置文件和前端不保留 `base_domain` 的读取、写入或兼容分支；部署所需配置直接更新为新字段。
- [ ] `gateway` mode 的 Docker labels 和 Gateway 自身派生地址仅使用 `internal_domain`，现有生成结果不变；无 `external_domain` 时 Route 创建表单的域名控件保持空白。
- [ ] 配置 `external_domain` 后，创建页、列表编辑弹窗和详情编辑弹窗中的 Route 编码变化可将 `编码.<external_domain>` 自动填入现有域名控件；提交到 Route 的是该控件中的完整域名。
- [ ] 联动仅更新空域名或与旧编码生成值一致的域名；打开已有 Route 的编辑表单不会覆盖已保存域名，手填的其他完整域名在之后修改编码时保持不变。
- [ ] Route 编码在创建、编辑和服务端写入校验中符合小写 DNS 单标签：以 `a-z` 开头，仅含 `a-z`、`0-9`、`-`，末尾为字母或数字，长度不超过 32；拼接后的完整域名也满足 DNS 名称长度限制。
- [ ] `external_domain` 支持多级子域名，各级标签符合 ASCII 主机名规则，末级标签不能纯数字；最长 220 字符。
- [ ] 修改 `external_domain` 不批量改变已有 Route，也不直接触发 Traefik 发布；Route 原有同步流程保持不变。
- [ ] 文档和验证覆盖 Gateway 存量迁移、内网 label 不变、外网域名辅助以及完整域名保存。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

- 采用 `internal_domain` / `external_domain` 作为持久化、代码与接口术语；原 `base_domain` 直接更名，不保留运行时兼容。
- 每个 Gateway 配置一个可选的 `external_domain` 后缀；它只驱动创建和编辑表单中“编码 -> 域名控件”的联动，不代表平台管理 DNS 或创建外网路由；Route 持久化完整域名。
- 域名控件为空或仍等于旧编码与当前外网域名拼接的值时才自动联动；其他值视为手工覆盖并保留。
- Route 编码直接采用最多 32 字符、以小写字母开头的 DNS 单标签规则；前后端保持一致，不接受纯数字、数字开头、点号、下划线或超长编码。
- `internal_domain` 的运行时行为保持不变，包括 `gateway` mode 标签；Route 创建表单不再用它预填域名。
- 既有 Gateway 数据由新增迁移转换；已有配置文件需同步采用新键，不在业务逻辑中兜底。

## Risks

- 字段更名跨数据库、项目初始化配置、Gateway/Route 接口、MCP、导入导出、前端和文档；遗漏任一入口会造成构建失败或运行配置不一致。
- 现有部署使用的 `project_initialization.gateway.base_domain` 配置键需要随版本更新；不兼容旧键意味着升级前须同步修改部署配置。
- 外网域名只是地址生成辅助，实际可访问性仍取决于用户维护的 DNS、Gateway 网络入口与证书配置。

## User review notes

- 用户确认 DNS 记录等外部配置不由 Orbit 管理。
- 用户要求标准模式开始任务，使用 `internal_domain` / `external_domain` 术语；必要时添加 SQL migration，业务逻辑不做旧字段兼容。
- 用户澄清：Route 创建和编辑都应将路由编码与外网域名联动，自动拼接结果要填入现有域名控件；内网域名行为保持不变。
- 用户确认：手填域名与生成格式不同即可识别，不应被后续编码变化覆盖；Route 编码规则应收紧为 DNS 合法格式，避免生成非法域名。
- 用户补充：Route 编码长度上限为 32 字符。
- 用户明确要求开始实现；存量不合规 Route 编码以后由离线 SQL 更新，本次不处理非法存量数据，也不在业务逻辑中兼容。
- 用户补充：Gateway 创建与编辑表单的两个域名字段各占一行，并核对包括 MCP 在内的全部创建和编辑入口。
- 用户进一步确认：自定义 Route 编码不能以数字开头；外网域名后缀按主机名标签规则校验并允许多级子域名。
- 用户更正：未配置外网域名时，Route 创建表单的域名控件应留空，不以内网域名预填。
