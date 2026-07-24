# Verification：网关详情部署与停止入口
最后修改时间: 2026-07-23 22:34:55

Review status: Accepted

Flow mode: light

Requirement: `docs/requirement/20260723-cd-gateway-detail-deploy-stop.md`（Accepted）

Spec: 不适用（light）

Plan: 不适用（light；按 requirement 实现）

## Requirement alignment

| Goal / Acceptance | 结果 |
|-------------------|------|
| 详情顶栏部署 / 停止 | 通过 — `GatewayDetail` 顶栏 Deploy / Stop |
| 复用 application deploy/stop API | 通过 — `applicationApi.deploy` / `stop` |
| 成功跳转 deployment 或刷新 | 通过 — 有 `deployment_id` 则 `router.push`，否则 `loadRuntimeContext` |
| 保留「版本与部署」 | 通过 — `goWorkload` → `/cd/application/:id` |
| 部署表单字段 | 通过 — version / environment / instance_key / force_recreate |
| 无 published 时回退最新版本 | 通过 — `selectDeployableVersions` + newest-first |
| 停止仅 running/faulted | 通过 — `stoppableServices` / `canStop` |
| 多实例可选 service + remove_volumes | 通过 |
| 中英文 i18n | 通过 — `gateway.actions` / `deploy` / `stop` / toast |

## Spec alignment

不适用（light）。

## Plan alignment

不适用（light；按 requirement 核对）。

## Actual diff summary

1. `GatewayDetail.vue`：部署/停止按钮与弹窗；加载 services / versions / environments；版本选择优先 published，否则全部 newest-first。
2. `zh-CN.ts` / `en-US.ts`：网关部署/停止文案。
3. `docs/requirement/20260723-cd-gateway-detail-deploy-stop.md`：需求与后续「无 published 用最新」决策。

## Expected vs actual changed files

| 预期（requirement） | 实际 | 备注 |
|---------------------|------|------|
| `web/src/views/cd/GatewayDetail.vue` | 是 | 核心实现 |
| `web/src/i18n/locales/zh-CN.ts` | 是 | |
| `web/src/i18n/locales/en-US.ts` | 是 | |
| requirement 文档 | 是 | light 过程文档 |
| verification 文档 | 是 | 本文件 |
| `ApplicationPage.vue` | **额外** | 应用列表增加「版本」链接，**不在本需求范围** |
| `VersionDetail.vue` | **额外** | 镜像列表头 i18n key 修正，**不在本需求范围** |

## Acceptance checklist

- [x] 顶栏可见部署、停止
- [x] 部署弹窗字段完整
- [x] 无已发布版本时回退最新版本
- [x] 停止按钮按 service 状态禁用/启用
- [x] 调用既有 deploy/stop API
- [x] i18n 覆盖
- [x] 用户手工验收：无问题

## Command / test results

| 命令 | 结果 |
|------|------|
| `yarn vue-tsc --noEmit`（`web/`） | 通过 |
| `yarn eslint src/views/cd/GatewayDetail.vue src/i18n/locales/{zh-CN,en-US}.ts` | 通过 |
| 浏览器手工验收（用户） | 通过（「验证无问题」） |
| 后端/集成测试 | 未跑（本期仅前端编排，无 API 变更） |

## Missed or expanded scope

1. **范围外已暂存改动**：`ApplicationPage.vue`（版本入口）、`VersionDetail.vue`（镜像字段 label）。与本需求无关，提交时应拆分或单独说明。
2. requirement Risk §2「无 published 应失败」已被 Decision §4 / 用户指示覆盖（回退最新版本）；行为以 Decision §4 为准。

## Risks

1. 提交若混入 ApplicationPage / VersionDetail，会扩大 commit 边界。
2. 未对 `selectDeployableVersions` 做单元测试（逻辑简单，依赖手工验收）。

## Incomplete items

无（相对本需求验收项）。

## Conclusion

**通过。** 网关详情部署/停止入口与 requirement 对齐；类型检查与目标文件 lint 通过；用户手工验收无问题。建议将本 feature 与无关 UI 改动分 commit（或至少在 message 中隔离说明）。
