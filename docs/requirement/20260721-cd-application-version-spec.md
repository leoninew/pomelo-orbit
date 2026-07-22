# CD 应用版本化：版本规格最小 schema
最后修改时间: 2026-07-22 08:35:55

Review status: Accepted

Flow mode: standard

Parent cadence: `docs/requirement/20260721-cd-application-version-cadence.md`（P1）

Depends on: P0 领域语义（`20260721-cd-application-version-domain.md`）

**后续路线**：P1 原文「域名/路由不进 Version 最小 schema」仍成立。  
ExposeSpec 进 Version、环境侧接入分别由：

| 修订 | 文档 |
|------|------|
| R1 | `20260721-cd-environment-expose-service.md`（Expose 表 + Binding 用户路径，已交付） |
| R2 | `20260722-cd-environment-ingress-policy.md`（Draft；策略取代 Binding） |

## Terminology / 规范用词

| 中文 | 英文 | 含义 |
|------|------|------|
| 版本 | **Version** | 业务数据；结构化规格 SoT |
| 组件 | **Component** | Version 内逻辑单元；定名不用 Workload |
| 服务 | Service | 仅运行绑定；不指 spec 内单元 |
| 容器 | Container | Component 物化实例 |

Version ≠ compose 文件；compose = P3 渲染结果。

## Background / 背景

P0 已钉死 Version 为业务数据。P1 钉死 **spec 最小面**：必须表达什么、禁止什么、与现状如何过渡。  
不定 JSON Schema 正则与编辑器 UI。

## Goal / 目标

1. 锁定 Version.spec 最小结构：Component、env、mounts、networks、depends_on。
2. 锁定生命周期与可变性（对齐 P0）。
3. 锁定路径、多 Component、覆盖表、expose 边界。
4. **P1 无遗留 Open question**。

## Non-goal / 非目标

1. 不定每个字段类型校验正则与完整 JSON Schema 文件。
2. 不实现编辑器 UI、密钥保险库。
3. 不实现 Renderer（P3）、deploy 任务（P2）。
4. **不把 expose/route/TCP 纳入最小 schema 必达**。
5. 不以 compose 全文作为 Version 本体。

## User scenarios / 用户场景

1. 未发布 Version 上改 Component 镜像/挂载后保存并可部署。
2. 发布后只读；fork 新 Version 再改。
3. 多 Component 应用在同一 Version 内表达依赖与网络。
4. 对比两 Version 的 Component/env/mounts 差异（粒度：按上述类别）。

## Spec surface / 规格面

> 全部属于 **Version.spec_payload**（及嵌入 Component）。  
> Service 只持 version_id 与状态；compose 全文是产物。

### 必须（P1 Accepted）

| 类别 | 最小内容 | 说明 |
|------|----------|------|
| **components[]** | 见下表 | 至少 1 个；**多 Component 一等支持** |
| **env** | 全局 map 与/或 per-component map | 敏感值本期可明文占位；密钥引用另开 |
| **mounts / volumes** | 源、目标、只读标志 | 源为**逻辑路径或命名卷名**，见路径规则 |
| **networks** | Component 加入的网络名列表 | 至少能表达加入平台网络（如 `traefik`） |
| **depends_on** | Component → Component 名列表 | compose 语义优先；无环由实现校验（Plan） |

### Component 最小字段（P1 Accepted）

| 字段 | 必选 | 说明 |
|------|------|------|
| name | 是 | Version 内唯一；渲染为 compose service 名 |
| image | 是 | 镜像引用 |
| command / args | 否 | 覆盖入口 |
| env | 否 | 与全局 env 合并规则：Component 覆盖同名键 |
| ports | 否 | **容器端口/协议意图**；宿主映射不进规格（由 runtime/暴露策略处理） |
| mounts | 否 | 见上 |
| networks | 否 | 默认空=仅项目默认网络；显式列出加入的网络 |
| depends_on | 否 | 依赖的 Component.name |
| healthcheck | 否 | 有则 P0 就绪口径走 runtime health；无则走 running |
| resources | 否 | CPU/内存等，可后置实现 |
| pull_policy | 否 | 覆盖 Application 默认 |

### 明确排除（不进 P1 最小必达）

| 类别 | 归属 |
|------|------|
| 域名/端口/路由等**对外访问**配置 | **本期不做**（现状 `application_route` 若仍在，不并入 Version 最小 schema；不是 Version 必达） |
| 用户编辑的 raw compose yaml | **不存在**该产品概念；compose 只由渲染生成 |
| force recreate 等 | **部署操作选项**（P2 / Deployment），不进 Version |
| Container 状态/日志 | Service / Container / Deployment |
| 物理绝对路径写死 | 禁止；见路径规则 |
| 项目权限 | Application / Project |

### 路径规则（Accepted）

1. Version 内挂载源使用 **逻辑路径**（相对应用数据根）或 **命名卷名**，**禁止**把本机绝对路径写进已发布规格作为唯一真相。
2. 部署时由 **本机平台配置**（如现有 `physical_app_dir` 方向）解析为物理路径。
3. Windows/Linux 差异在 **解析层**，不在 Version 内复制两套绝对路径。

### 与现状过渡（概念，Accepted）

| 现状 | 过渡 |
|------|------|
| `application_config_file` / 旧 compose | **迁移导入为 Version.spec** 后废弃编辑入口；SoT=Version |
| `application_service` / service_config | **迁入 Component** 后 **删除覆盖语义** |
| `application_route` | 对外访问；**不进** Version 最小 schema，本期不改造 |
| 磁盘 compose | **仅渲染产物**；功能完成后无「业务 raw compose」 |

## Version 生命周期（对齐 P0）

| 状态 | 编辑 | 部署 | 变更 |
|------|------|------|------|
| unpublished | 是 | 是 | 直接改 spec |
| published | **否** | 是 | **仅新增 Version**；不可回退 unpublished |

发布动作：`unpublished → published`（不可逆）。  
label：`(application_id, label)` 唯一。  
fork：`created_from_version_id` 指向源 Version，复制 spec 后可改（新 Version 为 unpublished）。

## Component 与 Container（对齐 P0）

```text
Service 绑 Version → 读 components[] 渲染 → apply → Container.component_name = Component.name
```

多副本：P1 不强制 replicas 字段；默认一 Component 一实例意图，runtime 多副本时 Container 0..N 仍挂同一 component_name。

## Acceptance / 验收标准

1. 必选/排除类别与 Component 最小字段表完整，与 P0 一致。
2. 多 Component 一等支持；路径规则与覆盖表废止写死。
3. expose/raw 明确不在 P1 必达。
4. 未发布/已发布与不可回退写死。
5. 过渡映射表可供 Plan 使用。
6. **Open questions 为空**。

## Open questions / 待确认

**无。** P1 范围已闭合。下列**不在 P1**：

| 项 | 归属 |
|----|------|
| deploy/stop、操作选项、就绪轮询 | **P2** |
| Compose 渲染实现（无 raw 路径） | **P3** |
| 域名/路由对外访问 | **本期不做** |
| 密钥、完整 JSON Schema 文件 | **Plan** |

## Decisions / 决策（P1 闭合）

1. Version = **结构化业务规格**；compose 非本体。
2. **多 Component 一等支持**（至少 1 个 Component）。
3. Component 最小字段表 Accepted；name+image 必选。
4. env 全局 + per-component；同名 Component 覆盖。
5. mounts/networks/depends_on 进最小面。
6. ports 只表达容器侧意图；宿主映射不进 Version 最小 schema。
7. healthcheck 可选；有则走 runtime health，无则 running（P0）。
8. **路径：逻辑/命名卷 + 本机解析**；禁止规格内写死绝对路径为 SoT。
9. **域名/路由对外访问不进 P1**；本期不做。
10. **无 raw compose 产品概念**。
11. force recreate 等 **不进 Version**。
12. **service_config 覆盖语义删除**（无兼容保留）。
13. 定名 **Component**；published 不可改不可回退。
14. **放弃历史兼容**；旧 config 文件/覆盖表不作为并行业务路径。

## Risk / 风险

1. 结构化与旧 compose 文件双轨若迁移拖延会分裂真相。
2. 无 expose 时 HTTP 入口仍依赖旧 route 表——接受为过渡。
3. depends_on 环与非法 name 需实现期校验。

## Follow-on / 路线索引

见 cadence Sub-requirements：P0–P3 → R1 → **R2** `20260722-cd-environment-ingress-policy.md`。

## Assumptions / 假设

1. Component.name 可安全映射为 compose service 名（字符集 Plan 约束）。
2. 平台始终能提供本机数据根以解析逻辑路径。
3. 一期主 runtime 为 Docker Compose。

## User review notes

- 2026-07-21：对齐 P0 术语与 Version 状态。
- 2026-07-21：**P1 闭合**；并澄清无 raw、无对外访问必达、操作选项不进 Version。
- 2026-07-21：用户 **接受 P1**；**放弃历史兼容**。
