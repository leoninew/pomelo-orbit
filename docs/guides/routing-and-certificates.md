# 路由与证书
最后修改时间: 2026-10-02 14:03:00

Doc role: living guide

Orbit 通过 Gateway 的 Traefik File provider 管理自定义 Route，每条 Route 使用路由编码命名的独立 YAML，例如 `route-mineru.yaml`，router/service 名分别为 `route-mineru-route`、`route-mineru-service`。Route 创建、编辑、删除先写入业务数据；列表页和详情页的启停先保留在前端草稿中。详情页的证书弹窗只保存 HTTP/HTTPS、证书方式及验证方式，不触发发布。同步预览显示发布、撤销、跳过清单；确认前核对选定业务配置、受管文件和 Gateway/Environment 依赖修订，发布后查询 Traefik API 核对加载结果。

详情页仅同步当前 Route，列表页显式选择 Project 范围。Project 预览取业务 Route 与已知发布记录的并集，确认按冻结 ID 顺序逐条处理：启用的 Route 发布或覆盖自己的文件；禁用及已删除的已知 Route 撤销自己的文件；禁用且从未发布的 Route 直接跳过。范围外路由、未知文件、Docker label 与其他 provider 资源保留。预览不比较 Traefik 增删改，也不提交运行时 hash。

Web 在同步模态窗的原预览表中保留名称、操作、协议与规则，追加状态列；协议列展示 HTTP/HTTPS 及证书方式（例如 `Let's Encrypt · DNS-01`），规则列只展示匹配条件与目标。API 返回独立的 `rule.protocol`、`rule.match` 和 `rule.target`，协议由入口配置决定，不从上游 URL 推断。确认、逐条反馈、失败重试与完成结果都留在同一模态窗和表格中。前端按预览顺序逐条确认，每次只提交该 Route 的预览修订和启停草稿。每条响应立即更新对应记录，单条失败后继续处理后续项；状态依次为待处理、处理中、完成或失败，失败原因通过状态提示查看。每条响应同时更新页面已保存的草稿，已保存的业务修改不会因外部发布失败回滚；未保存的启停草稿仍保留。重新预览只重试未完成项，并保留成功记录。同步执行期间禁止关闭模态窗，完成后由用户关闭；本轮不增加关闭页面或弹窗后的任务续跑能力。

API/MCP 保留显式批量确认能力。确认响应的汇总编码为 `route_sync_completed` 或 `route_sync_incomplete`，逐项报告业务保存、文件提交、配置匹配、证书验证、恢复和清理状态；Web 只展示一个状态字段，不展开内部阶段。预览同时返回整批及每项独立的 `business_hash`、`publication_hash`，单条确认使用对应项的修订，期间业务或共享依赖变化须重新预览，不能自动发布未确认的新内容。

Traefik 直接重启从持久化文件恢复，Orbit 可以离线。Gateway 部署/重启只检查已发布文件及其加载结果，不重新读取可编辑 Route 发布。业务 Service 部署会保护已发布的上游引用；需要移除或改名相关 Component/endpoint 时先显式同步撤销或调整对应 Route。Component `endpoint.mode=gateway` 仍是 Docker label 路由，默认 entrypoint/TLS 由 GatewayConfig 控制。

前端 `/routes` 以自定义 Route 为主列表，提供创建、编辑、批量启停草稿和同步入口；Route 详情页提供单条启停草稿、独立证书配置和同步。已启用 Route 修改证书后先保存，页面提示待同步；预览同时展示协议与证书方式。未启用 Route 的证书配置同样直接保存，不提示待同步，启用并同步后才发布。未启用 Route 的创建、编辑和删除也不单独提示待同步。`/route/traefik` 单独展示 Traefik 当前路由，可从主列表进入并返回。

Route 编码为最多 32 字符的小写 DNS 单标签，以字母开头，只允许 `a-z`、`0-9`、`-`，末尾不能是连字符。`external_domain` 可为 `sub.example.com` 等多级子域名，每级标签符合 ASCII 主机名规则，末级标签不能纯数字，最长 220 字符。Gateway 配置 `external_domain` 后，创建和编辑 Route 时修改编码会在现有域名控件中生成 `编码.external_domain`；手工填写的其他域名不会被后续编码修改覆盖。未配置外网域名时，创建表单域名控件留空。保存的是域名控件中的完整域名；外网域名配置本身不创建 DNS 记录或对外路由，也不批量修改已保存 Route。

Gateway 详情页在独立的「Traefik API」卡片中展示和编辑容器 API URL、宿主机 API URL 及就绪超时；基本信息单独编辑名称和域名。移除的是 REST provider 发布入口，管理 API 继续用于就绪检查、运行时列表和发布/恢复后的加载核验。本地容器中的 Orbit 使用容器地址；本机 Orbit 和 SSH 目标使用宿主机地址，SSH 请求在远端执行。

Gateway Service 工作目录的结构如下：

```text
<Environment.workspace_root>/deployment/<gateway-service-code>/
  gateway/
    dynamic/route-<route-code>.yaml
    certs/route-<route-id>/<sha256-revision>/
      cert.pem
      key.pem
    acme/acme.json
  .orbit/route-publication/<route-id>/
    state.json
    previous.yaml
```

证书目录和 `.orbit` 归属记录使用稳定 ID。修改路由编码后显式同步会写入新编码文件、撤下旧文件，并核对旧 router/service 消失；失败恢复旧路径。撤销使用已发布编码，未同步的编码修改不会使其他路由文件被删除。编码仍被其他已发布或 pending 路由占用，或目标文件未登记归属时，发布报告冲突。

删除 `gateway/dynamic` 后，重新部署 Gateway 可以重新准备空目录，但部署后仍会报告已发布文件缺失。重新预览并确认 Route 同步可重建选中 YAML 及其父目录；只有用户确认的内容被发布。外部修改的现存文件仍报告冲突，不自动覆盖；预览后删除文件会使该预览过期。

Gateway 部署统一应用 `providers.file.directory=/etc/traefik/dynamic`、`watch=true`，移除 REST provider，动态配置与手工证书目录只读挂载，ACME 目录读写挂载；静态文件覆盖后自动重建 Gateway 容器。发布记录与旧 YAML 位于未挂载、未监视的 `.orbit` 下。路由文件、PEM 与记录使用现有 `WorkspaceFile/SyncFiles`，文件 Mode 为 `0600`；完整临时文件写好后替换活动文件。清理只处理该 Route 的旧证书版本，保留当前和必要的上一版本。

SSH 文件由远端 runtime 写入，沿用远端身份和权限。Orbit 容器内运行 local target 时，文件写 logical path，Docker 挂载使用既有 resolver 得到的 daemon 可见 physical path。正常发布检查运行容器的 provider、entrypoint/resolver 和实际挂载；权限失败不会被当作成功。Windows 替换窗口与 ACL 平台差异仍为已知限制。现存受管 REST Gateway 通过线下告知重新部署后显式同步，不转换旧 REST 快照或新增迁移 UI。

Windows 工作目录 bind mount 到 Docker Desktop Linux 容器时，可能出现“容器可读新文件，但 File provider watcher 不刷新”。Orbit 在文件提交后向当前 Gateway 容器发送 SIGHUP，让 File provider 重新读取配置，再核对加载结果；撤销、失败恢复和中断恢复使用相同机制，不重启 Gateway。SSH 按远端平台判断，local 按实际 daemon mount source 判断，因此容器中的 Orbit 也能识别 Windows bind mount 的 DooD 场景。Linux 原生目录与 SSH Linux 继续使用 watcher，暂不主动发送 HUP；真实 SSH Linux 热更新留待后续测试。

重载失败使用逐项编码 `route_sync_reload_failed`，不会被当作同步成功。文件恢复后仍须重载并匹配才报告已恢复；失败则保留恢复材料供下次预览和重试。SIGHUP 重读现存文件集合，保留其他 Route 和未知文件，不读取未同步的业务编辑。

文件权限失败使用逐项编码 `route_sync_publish_permission_denied`，其他发布失败使用对应原因编码或 `route_sync_publish_failed`；弹窗的失败状态提示展示本地化原因，不显示内部阶段和请求/操作 ID。配置匹配与实际 TLS 证书验证在 API 中分开报告：router/service API 匹配不能证明新 PEM 已生效，当前正常同步的 HTTPS 证书结果为 `unverified`。界面“完成”表示配置同步及必要操作完成，不表示证书已生效。实际 TLS 验证使用正确 SNI 和预期证书 fingerprint；未验证证书不自动回滚已匹配的配置。

HTTP Route 可使用手工 PEM、mkcert 或 Let's Encrypt。HTTP-01 需要 Gateway `http` 或 `http-dns` profile；DNS-01 需要 `dns` 或 `http-dns` profile 和 Gateway 中保存的 Cloudflare token。DNS-01 仍要求可注册的真实域名。

配置 DNS-01：

1. 在 Cloudflare 创建仅包含目标 zone `Zone:Read`、`DNS:Edit` 的 API token。
2. 在 Gateway 编辑页选择 `dns` 或 `http-dns`，填写 email 与 token。
3. 保存并重新部署 Gateway。
4. 在 Route Let's Encrypt 对话框选择 DNS-01。

TCP Route 先要求选中 Gateway Version 已声明对应 TCP endpoint 和 host port，再创建 Route。GatewayConfig 不提供 listener 或端口编辑。
