# 编码字段按用途采用 RFC 优先规则
最后修改时间: 2026-09-24 22:59:45

Review status: Draft

Mode: light

## Background

Project 编码现以小写 RFC 3986 Scheme 字符集为基础，产品限制为 1–32 位；`orbit.preflite.cn` 可用于创建 Project 和导入接管包。其他业务编码分别进入 DNS 名称、Docker Compose 项目名、工作目录、证书文件名或权限标识，现有字符与长度限制并不一致。Project code 同时是 Environment code；Application code 会派生域名及带 `-preview`、`-default` 后缀的 Service code；Service 和 Component code 还会成为 DNS 标签。Repository code 用作流水线工作目录的一段。

RFC 3986 §3.1 的 Scheme 语法本身不规定 32 位上限，也不要求小写。DNS 标签的字符和长度约束与 Scheme 不同；角色、权限和环境变量键没有可直接套用的 Scheme 语义。长度、小写和业务范围应明确标为 Orbit 产品约束，不能称作 RFC 原文要求。

## Goal

1. 为每类编码或机器标识指定与用途相符的规则；有对应 RFC 语义时优先引用 RFC，没有时明确采用产品规则，不强行共用 Project 编码正则。
2. 统一同一字段在前端、后端、导入/接管及运行时的校验和提示，使创建时被接受的值在实际使用场景中仍有效。
3. 将长度、允许字符、首尾限制、唯一性作用域和派生标识约束写清楚，并在修改规则前审计现存数据。
4. 选择已有 Project、Repository、Application、Service 等编码关联对象时使用项目共享的可搜索控件，能按对象名称或编码定位；新建编码仍使用文本输入。

拟采用的目标规则如下。长度和字符集除明确注明 RFC 的部分外，均为待确认的产品约束：

| 字段 | 规则依据与目标字符限制 | 目标长度 |
| --- | --- | --- |
| Project code、Environment code | RFC 3986 §3.1 Scheme 的小写产品子集：`^[a-z][a-z0-9+.-]*$`；Environment 继承 Project code | 1–32 |
| Repository code | Scheme 字符集的路径安全子集；禁止末尾 `.` 和 Windows 保留文件名，保留可移植的单目录段 | 1–100（待确认） |
| Application code、Gateway Application code | RFC 1123 DNS 标签的小写形式：首尾为字母或数字，中间可用 `-`；为 `-preview`、`-default` 预留 8 位 | 1–55（待确认） |
| Service code | RFC 1123 DNS 标签的小写形式：首尾为字母或数字，中间可用 `-` | 1–63 |
| Version Component name | 同一 DNS 标签规则，创建和发布时一致 | 1–63 |
| Route name | RFC 3986 unreserved 的产品子集：`^[a-z][a-z0-9._-]*$`；兼顾 Traefik 标识和证书文件名 | 1–100（待确认） |
| Role code | 无直接对应的 RFC 语义；暂保留 `^[A-Za-z0-9_-]+$` | 1–50 |
| Permission code | 产品定义的 `领域:操作` 格式，例如 `user:read`；仅系统预置，不新增用户创建入口 | 1–100（待确认） |
| 环境变量键 | 保留变量名规则 `^[A-Za-z_][A-Za-z0-9_]*$`，不套用 URI 或 DNS 规则 | 1–255（待确认） |

## Non-goal

- 不将所有编码强制收敛到 Project 的 Scheme 语法；尤其不允许 `+`、`.` 进入单个 DNS 标签。
- 不改变 Version label 等供人阅读的文本字段，也不为所有字段增加统一的 32 位上限。
- 不新增编码别名、运行时 slug、新旧规则双轨或静默改写已有标识。
- 本 Intent 只记录需求；本轮不实施代码、数据库迁移或活产品文档更新。

## User scenarios

1. 用户创建或从接管包导入 Project 时可使用 `orbit.preflite.cn`；对应 Environment code 与 Project code 一致。
2. 用户创建 Application、Service、Component 时，界面和 API 对 DNS 标签限制给出一致反馈；值在派生域名、Compose 项目名和运行时容器名时仍合法。
3. 用户创建 Repository 时，代码可作为本地与 SSH 流水线工作目录的可移植单段名称；不因尾随点或系统保留名称产生路径歧义。
4. 角色和权限维持各自业务标识语义；修改字符或长度规则前能识别与处理现存值，而非在后续读取或授权时才失败。
5. 用户在关联对象选择器中输入名称或编码片段，均能找到同一对象，不必浏览完整选项列表。

## Acceptance

- [ ] 对上表每个字段明确最终字符规则、长度和唯一性作用域；RFC 来源与 Orbit 附加限制分别说明。
- [ ] Project/Environment 保持现行 1–32 位 Scheme 子集，`orbit.preflite.cn` 在创建和接管包导入流程均有效。
- [ ] Application/Gateway、Service、Component 的创建、编辑、导入和运行时约束一致；派生 `-preview`、`-default` 后缀后的 Service code 不超过 63 位。
- [ ] Repository code 在受支持的本地和 SSH 工作目录中是安全单段名称；Route name 用作 Traefik 标识与证书文件名时有效。
- [ ] 前端字段错误、后端校验和对用户显示的规则说明一致；同一字段的新建和更新不出现相互矛盾的字符条件。
- [ ] 涉及上述编码对象的已有对象选择入口使用共享的可搜索控件；名称和编码都可搜索，选项能区分同名对象，键盘选择及无结果状态可用。
- [ ] 规则收紧前审计现存标识和跨领域引用，明确冲突处理；不引入兼容别名或静默转换。
- [ ] 对主要正向流程和真实的高风险边界补充验证，并同步更新相关活文档。

## Open questions

1. Repository 与 Route 的 100 位上限、Permission 的 100 位上限及环境变量键的 255 位上限是否符合产品预期？这些不是 RFC 限制。
2. Repository code 是否允许内部 `.` 和 `+`？若允许，须确认 Windows 保留名和末尾 `.` 的拒绝方式，并审计现存编码。
3. Role code 是否维持大小写和下划线；是否需要把权限编码的 `领域:操作` 格式及长度固化为显式校验？
4. Application/Gateway code 收敛为最多 55 位、允许数字开头后，现有项目数据和各派生路径是否存在冲突？
5. 与进行中的全局 Repository/Repository Credential 需求是否存在编码唯一性作用域调整；两项实现需要对齐最终作用域。

## Decisions

- 已确定按用途选择规则，RFC 优先但不强行将 Scheme 语法用于 DNS、路径或权限字段。
- 现行 Project code 规则是本需求的基线；其余表格内容为待审查提议，不能视为已经获准实施。
- 本轮采用 SpecFlow 轻量模式，仅形成 Intent 草稿，不进入 Implementation 或 Verification。

## Risk

- 收紧 Application、Component、Repository 等既有规则可能拒绝已保存的值；上线前需要数据审计与明确处置。
- MySQL 列长度、SQLite/PostgreSQL 无同等列长度限制与应用层校验可能不一致，不能仅凭单一数据库验收。
- DNS 主机名、Compose 项目名、Windows/SSH 路径和 Traefik 标识的约束不同；只验证正则匹配不足以证明实际运行安全。
- 现有 `ComboboxSelect` 以选项 `label` 作为搜索文本，编码若仅放在 `description`，按编码检索可能失效；实施时需核对共享控件的搜索契约与调用处。
