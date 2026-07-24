# Web 1440 中等屏布局适配验证
最后修改时间: 2026-07-14 10:17:55

Review status: Accepted

## Requirement alignment

- 在 1440px 宽度下，针对已发现的应用卡片、流水线运行表格和构建阶段详情问题完成了修复并重新自动化验证。
- 验证覆盖页面级水平溢出、应用创建时间换行、流水线表格最小可读宽度与开始时间换行，以及构建阶段空制品配置文案。
- Traefik HTTP Routers 页面在超过后端 10 秒超时边界后不再停留在 loading 状态，显示现有的失败重试或空状态。
- 未改动 HTTP API、权限逻辑、数据模型或导航体系。

## Spec alignment

不适用：light 模式按 Requirement 核对。

## Plan alignment

不适用：light 模式按 Requirement 核对。

## Actual diff summary

- `web/src/views/Home.vue`：首页概览卡片和近期构建/部署区域延后到 `2xl` 才切换为更密集的多列布局，避免 1440px 下近期表格状态标签被挤压截断。
- `web/src/views/cd/ApplicationPage.vue`：应用卡片在 1440px 下保持三列布局；应用 code 允许在自身空间内截断并提供 `title`，创建时间固定为不可收缩、不换行项。
- `web/src/views/ci/PipelineRunPage.vue`：流水线运行表格保持局部横向滚动，将最小宽度调整为 `1200px`，重新分配 Template 与 Start Time 列宽，并确保开始时间不换行。
- `web/src/views/ci/BuildStageDetail.vue`：将空制品配置从不存在的 `buildStageDetail.noArtifactConfigResp` 改为已有的 `buildStageDetail.noArtifactConfig`。
- `web/src/i18n/locales/buildStageDetail.spec.ts`：新增中英文 locale 均定义空制品配置文案的回归测试。
- `web/src/views/Settings.vue`：设置表格改为固定列宽，长配置 key/value 以单元格内截断和 `title` 方式展示，避免长 JWT/config 值撑开表格。
- `configs/config.yaml`：为本地无交互浏览器复测关闭 Turnstile（`turnstile.enabled: false`）；运行中的服务已确认加载该配置。
- 新增或更新 `docs/requirement/20260713-web-1440-layout-adaptation.md` 与本 verification 文档。

## Expected versus actual changed files

| 预期文件 | 实际状态 | 说明 |
| --- | --- | --- |
| `web/src/views/Home.vue` | 已修改 | 优化 1440px 首页卡片与近期列表排布。 |
| `web/src/views/cd/ApplicationPage.vue` | 已修改 | 稳定应用 code 与创建时间的单行元数据布局。 |
| `web/src/views/ci/PipelineRunPage.vue` | 已修改 | 增加 9 列运行表格可读空间，避免开始时间换行。 |
| `web/src/views/ci/BuildStageDetail.vue` | 已修改 | 修复空制品配置的 i18n key。 |
| `web/src/i18n/locales/buildStageDetail.spec.ts` | 已新增 | 防止中英文 locale 缺失空制品配置文案。 |
| `web/src/views/Settings.vue` | 已修改 | 控制设置表格列宽和长值溢出。 |
| `configs/config.yaml` | 已修改 | 按本地自动化验证需要关闭 Turnstile。 |
| `docs/requirement/20260713-web-1440-layout-adaptation.md` | 已新增 | 轻量 Requirement。 |
| `docs/verification/20260713-web-1440-layout-adaptation.md` | 已更新 | 本轮验证记录。 |

## Acceptance checklist

- [x] 1440px 宽度下已复测页面没有页面级水平溢出。
- [x] 首页近期构建/部署表格在 1440px 下不再截断状态标签。
- [x] 应用卡片页在 1440px 下保持三列，长应用 code 不会挤压或折行创建时间。
- [x] 流水线运行表格在 1440px 下拥有至少 `1200px` 的局部可滚动表格宽度，开始时间不换行。
- [x] 构建阶段详情不再显示原始 `buildStageDetail.noArtifactConfigResp` key。
- [x] 设置页表格在 1440px 下不再因长配置值撑出容器。
- [x] Traefik HTTP Routers 在超过下游超时边界后已离开 loading 状态。
- [x] 前端测试、lint、类型检查和构建检查通过。

## Test results

本地开发服务运行于 `http://localhost:9020`。为使自动化登录不受 Cloudflare 验证阻塞，已将 `configs/config.yaml` 中的 `turnstile.enabled` 设为 `false`；`GET /api/auth/turnstile-config` 返回 `{"enabled":false,...}`，确认热更新已生效。

1440px Pomelo PW 自动化产物和流程均位于 `.pomelo-pw/`：

```text
.pomelo-pw/verify-1440-fixes.yaml
.pomelo-pw/inspect-1440-fixes.yaml
.pomelo-pw/verify-1440-fixes-output/
.pomelo-pw/inspect-1440-fixes-output/
```

检查命令：

```text
pomelo-pw validate .pomelo-pw/verify-1440-fixes.yaml
pomelo-pw run .pomelo-pw/verify-1440-fixes.yaml --headless -o .pomelo-pw/verify-1440-fixes-output
pomelo-pw validate .pomelo-pw/inspect-1440-fixes.yaml
pomelo-pw run .pomelo-pw/inspect-1440-fixes.yaml --headless -o .pomelo-pw/inspect-1440-fixes-output
yarn --cwd web test buildStageDetail.spec.ts
yarn --cwd web lint:fix
yarn --cwd web typecheck
yarn --cwd web build
```

结果：

- 两份 Pomelo PW flow 均通过。断言 flow 覆盖 `/cd/applications`、`/ci/run`、`/ci/build-stage/01STAGE000000000000000CLONE` 和 `/cd/traefik-http-routers`。
- 断言确认前三个页面均无页面级水平溢出；应用创建时间具有 `white-space: nowrap` 且未溢出；流水线运行表格宽度不小于 `1200px` 且开始时间不换行；构建阶段页面不包含原始 i18n key。
- Traefik 页面等待 12 秒后不再存在 loading spinner，并显示已有的 Retry 错误状态或 empty state。
- `yarn --cwd web test buildStageDetail.spec.ts`：通过，1 个测试文件、1 个测试。
- `yarn --cwd web lint:fix`：通过，0 errors；保留 5 条既有 `import-x/no-named-as-default` warnings，涉及 `AppTopBar.vue`、`request.ts`、`time.ts`。
- `yarn --cwd web typecheck`：通过。
- `yarn --cwd web build`：通过；仅有第三方 `reka-ui/@vueuse` 的 Rollup `#__PURE__` warning。

## Scope deviation

- 在需求原始范围之外，为执行用户要求的本地无交互复测，显式将开发配置中的 Turnstile 关闭。未对登录接口、前端登录逻辑或验证器实现添加绕过分支。
- Traefik 没有修改：受控等待证实其现有状态机能在下游超时后离开 loading 状态。

## Risks

- `configs/config.yaml` 当前关闭 Turnstile，适合本地自动化；部署到任何需要防护的环境前必须通过其环境配置重新启用并提供有效的 key。
- 设置页及受限表格列中的极长内容仍采用截断、局部横向滚动或 `title` 展示完整值，符合中等桌面宽度的可读性与不溢出目标。

## Incomplete items

无。

## Conclusion

本轮实现满足轻量 Requirement：已确认的 1440px 页面缺陷已修复并由浏览器断言复测；Traefik 不存在“超时后永久 loading”问题；前端静态检查通过。Turnstile 已按本地自动化需要关闭，未执行任何 Git 提交或推送。
