# CD Environment IngressPolicy 实现计划
最后修改时间: 2026-07-22 09:05:11

Review status: Accepted

Flow mode: strict

## Requirement basis

| 文档 | 状态 |
|------|------|
| `docs/requirement/20260722-cd-environment-ingress-policy.md` | **Accepted** |
| `docs/spec/20260722-cd-environment-ingress-policy.md` | **Accepted** |
| `docs/requirement/20260721-cd-application-version-cadence.md` | Accepted；本 Plan = **修订 R2** |
| `docs/plan/20260721-cd-environment-expose-service.md` | R1 已交付；本 Plan **废止** Binding 用户 SoT |

**硬约束（不得偏离 Spec）**：

1. DROP `environment_binding`；策略列内嵌 `environment`。
2. 默认 Host = `{app_code}.{base_domain}`；可选 `domain_template`。
3. Environment **禁止** component 配置维度。
4. 无双轨 / 无历史兼容；migration 仅新增 `000010_*`。
5. Render = `Version.Expose × Environment.IngressPolicy × Application`。

## Plan decisions

| # | 决策 | 说明 |
|---|------|------|
| P1 | 列名 | `base_domain`, `domain_template`, `default_entrypoint`, `tcp_entrypoint`, `tls_mode` |
| P2 | seed 默认 | 全部 env 回填 `base_domain=local.test`, `default_entrypoint=web`, `tls_mode=none`；`domain_template`/`tcp_entrypoint` 空 |
| P3 | 新建 Project / CreateEnvironment | 同步默认策略字段 |
| P4 | 域名模板 | 仅 `{app_code}` `{env_code}` `{base_domain}`；未知 token → 校验失败 |
| P5 | path 冲突 | 规范化后空 path 视为 `/`；同 host+path 重复失败 |
| P6 | HTTP labels | 单 Host；`path_prefix` 非空且非 `/` 时加 PathPrefix |
| P7 | TCP labels | entrypoint = `tcp_entrypoint` 或 `default_entrypoint`；`tls_mode=none` 时 `HostSNI(\`*\`)`，**不**强制 base_domain；`letsencrypt`/`tls` 用推导 host 并要求 base_domain |
| P8 | API | 删除 `PUT .../binding`；CRUD 请求/响应带策略字段 |
| P9 | UI | Environment 表单编策略；删除 Bindings 模态与列表「绑定数」列，展示 base_domain |
| P10 | 测试 | migrate version = **10**；compose 测 Host 推导与缺策略失败 |

## Implementation steps

1. **Migration 000010**（sqlite + mysql）：ALTER environment 加列 + 回填 + DROP environment_binding。
2. **Model / Repo**：`model.Environment` 扩字段；删除 `EnvironmentBinding` 与 Bindings API；Create/Update SQL 含策略列；Project seed 写默认策略。
3. **Render**：去掉 Bindings；`deriveHost` + `injectExposeLabels` 按策略写 Traefik labels；path 冲突校验。
4. **Usecase**：Environment CRUD 归一化策略；删除 `ReplaceEnvironmentBindings`；Deploy/Preview 不再读 binding。
5. **Proto / handler / routes**：环境消息增策略字段；删 Binding 消息与路由；`task proto`。
6. **Frontend**：EnvironmentPage + api + i18n。
7. **Tests / checks**：migration 断言 10；compose/deploy tests；`go fmt/vet/test`；`yarn lint:fix` + `typecheck`。

## Rollback

- Down：不恢复 binding 数据；可重建空 `environment_binding` 表结构并去掉策略列（sqlite 表重建）。
- 产品无兼容层。

## Out of scope

- 应用级域名覆盖、K8s、创建向导重做、manual 证书上传。
