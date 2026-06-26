# CD 应用创建方式选择计划
最后修改时间: 2026-06-27 00:00:51

Review status: Accepted

## Requirement basis / 需求依据

依据 `docs/requirement/20260626-cd-application-create-methods.md`，本次处于标准模式 / standard 的 Plan / 计划阶段。Requirement 已按用户“进入 plan”要求标记为 `Accepted`。

本次实现范围限定为 P1 中的“从镜像创建”：

- 应用列表页主入口改为“创建应用”，点击后跳转独立创建页面 `/cd/applications/create`。
- 创建流程第一步展示 4 种方式：从镜像创建、从 docker-compose 创建、导入应用包、高级空白应用。
- 本次只有“从镜像创建”完整可提交。
- 环境变量落盘为 `.env`，并在 `docker-compose.yml` 里通过 `env_file: .env` 显式引用。
- 数据目录支持多条 volume 映射。
- 不新增 backend-go 聚合接口，由前端串联现有接口。
- 串联失败后，用户重新提交应继续补齐同一个已创建应用。
- 失败恢复操作提供“跳转详情页手动修复”。
- 生成规则需要遵从当前实现，尤其是 backend-go 对 compose service、ports、route 的解析方式。

## Current implementation findings / 当前实现结论

### 前端现状

- `frontend/src/views/cd/ApplicationPage.vue`
  - 当前主按钮原先打开 `AppDialog`，使用 `ApplicationFormFields` 创建空白应用。
  - 本次需要将主按钮改为跳转独立创建页面 `/cd/applications/create`。
  - 当前独立“导入”按钮保留 JSON 应用包导入能力。
  - 已有 API 能力包括：创建应用、创建/更新/删除配置文件、列出配置文件、创建/更新路由、列出路由。
- `frontend/src/components/ApplicationFormFields.vue`
  - 当前封装空白应用基础字段：name、code、image_pull_policy、route_managed。
- `frontend/src/api/cd/application.ts`
  - 已有 `create`、`createFile`、`listFiles`、`writeFile`、`deleteFile`、`createRoute`、`updateRoute`、`listRoutes` 等方法，足够前端串联创建从镜像应用。
- `frontend/src/i18n/locales/zh-CN.ts` 和 `frontend/src/i18n/locales/en-US.ts`
  - 需要新增创建向导文案和校验提示。
- `reka-ui` 版本为 `^2.9.7`，可使用 `StepperRoot`、`StepperItem`、`StepperTrigger`、`StepperIndicator`、`StepperTitle`、`StepperDescription`、`StepperSeparator`。

### backend-go 现状

- 本次计划不修改 backend-go。
- `backend-go/internal/service/cd/application_extra.go` 中 `extractDefaultPort` 当前从 compose service 的 `ports` 第一项解析端口，解析失败默认 80。
- 路由创建需要 `service_name`、`domain`、`port`，并且部署时 `injectApplicationRouteLabels` 要求 route 的 `service_name` 必须存在于 compose services 中。
- 因此从镜像生成的 compose 必须包含明确 service，并为了遵从当前 service 端口解析，计划写入 `ports`。

## Design decisions / 设计决策

### 1. 组件组织

新增独立创建页面 `frontend/src/views/cd/ApplicationCreatePage.vue`，并新增 `frontend/src/components/ApplicationCreateWizard.vue` 封装创建向导和 Reka Stepper primitives。

理由：用户明确要求应用创建使用独立路由和页面，不停留在应用列表上，不使用模态窗。项目规范要求页面层优先使用共享组件，因此 Stepper 和创建表单主体封装在 `ApplicationCreateWizard.vue`，独立页面负责标题、返回按钮和项目上下文。

### 2. 创建方式展示

向导第一步展示 4 个方式卡片：

1. 从镜像创建：可继续下一步并提交。
2. 从 docker-compose 创建：展示“后续支持”提示，不进入提交流程。
3. 导入应用包：展示提示，说明当前仍可使用列表页独立导入按钮。
4. 高级空白应用：在独立创建页面内进入原方式表单，只创建应用基础信息，不生成配置文件。

本次不删除现有独立“导入”按钮，避免破坏当前 JSON 导入能力。

### 3. Stepper 步骤

使用受控 `StepperRoot v-model="currentStep"`，保留线性流程。

建议步骤：

1. 选择创建方式。
2. 填写镜像应用配置。
3. 确认生成内容并提交。

第 2 步只在选择“从镜像创建”时可进入。其他创建方式展示提示后停留在第 1 步或提示不可继续。

### 4. 从镜像表单字段

表单状态建议包含：

```ts
interface ImageCreateForm {
  name: string;
  code: string;
  image: string;
  containerPort: string;
  imagePullPolicy: string;
  envVars: Array<{ key: string; value: string }>;
  volumes: Array<{ hostPath: string; containerPath: string }>;
  routeDomain: string;
}
```

校验规则：

- `name`：沿用当前非空校验。
- `code`：沿用当前 `^[a-z][a-z0-9-]*$`。
- `image`：trim 后非空。
- `containerPort`：整数，范围 1-65535。
- 环境变量：忽略 key/value 都为空的行；非空 key 必须匹配 `^[A-Za-z_][A-Za-z0-9_]*$`；value 不允许包含换行。
- volume：忽略 hostPath/containerPath 都为空的行；非空行要求两端都填写；containerPath 建议以 `/` 开头。
- routeDomain：可选；填写后 trim 非空即可提交，域名格式不做过度校验，后端仍会校验 route 必填和端口范围。

### 5. 生成 compose / env 的规则

新增前端工具文件：`frontend/src/utils/applicationCompose.ts`。

职责：

- `buildImageCompose(form)`：生成 `docker-compose.yml` 内容。
- `buildEnvFile(envVars)`：生成 `.env` 内容。
- `normalizeEnvRows` / `normalizeVolumeRows`：清理空行和 trim。

生成规则：

- service 名称使用 `app`。
  - 理由：单服务向导生成稳定 service 名称，路由创建也固定使用 `app`，避免从应用代码派生时在 Traefik router label、compose key 等位置引入额外命名差异。
- compose 结构：

```yaml
services:
  app:
    image: "<image>"
    ports:
      - "<containerPort>:<containerPort>"
    env_file:
      - .env
    volumes:
      - "<hostPath>:<containerPath>"
```

- `env_file` 只有存在有效环境变量时生成。
- `volumes` 只有存在有效 volume 映射时生成。
- `ports` 使用同端口主机映射，原因是当前 backend-go `extractDefaultPort` 从 `ports` 解析默认端口；这也让详情页 service 默认端口、路由端口和部署行为保持一致。
- YAML 使用简单字符串生成，不新增 YAML 依赖；需要对双引号和反斜杠做最小转义，确保镜像、端口、volume 路径作为双引号字符串输出。
- `.env` 每行输出为 `KEY=value`，末尾保留换行。

风险说明：同端口主机映射可能造成宿主端口冲突，但这是遵从当前 `ports` 解析的最小实现。后续若要区分 host port 与 container port，应另开需求。

### 6. 前端串联接口与失败恢复

提交“从镜像创建”时按以下顺序执行：

1. 如果当前向导状态没有 `createdApplication`：
   - 调用 `applicationApi.create`。
   - `route_managed` 根据 `routeDomain.trim() !== ''` 派生。
   - 成功后把返回的 application 保存在向导局部状态中。
2. 如果当前向导状态已有 `createdApplication`：
   - 不再重复创建应用，直接继续补齐配置。
3. 调用 `applicationApi.listFiles(createdApplication.id)`。
4. 对 `docker-compose.yml` 执行 upsert：
   - 已存在同路径文件时调用 `writeFile`。
   - 不存在时调用 `createFile`。
5. 对 `.env` 执行同步：
   - 有环境变量时 upsert `.env`。
   - 没有环境变量且 `.env` 是本次向导已创建应用的配置文件时，可以删除向导创建的 `.env`；实现中若无法可靠判断，宁可不删除已有 `.env`，避免误删用户手动修复内容。
6. 如果填写了路由域名：
   - 调用 `applicationApi.listRoutes(createdApplication.id)`。
   - 查找 `service_name === 'app'` 的 route。
   - 已存在则调用 `updateRoute`。
   - 不存在则调用 `createRoute`。
7. 成功后跳转回应用列表页，并清空独立页面中的向导状态。

失败处理：

- 如果第 1 步创建应用成功、后续任一步失败，保留 `createdApplication`，显示错误提示。
- 用户再次点击提交时继续补齐同一个 `createdApplication`，不再创建新应用。
- 错误提示区提供“跳转详情页手动修复”按钮，目标为 `/cd/applications/{createdApplication.id}`。
- 如果第 1 步创建应用失败且没有 `createdApplication`，直接展示错误，不尝试用同 code 搜索并接管已有应用，避免误修改用户已有应用。

### 7. 现有空白创建处理

为避免本次范围扩散，高级空白应用在独立创建页面中复用 `ApplicationFormFields` 和现有 `applicationApi.create` 创建逻辑。用户选择“高级空白应用”后进入原字段表单，只创建应用基础信息，不生成 `docker-compose.yml`、`.env` 或路由配置。

这保证高级空白模式仍然是当前原语义，只是入口被纳入独立创建页面。

## Files to change / 预计改动文件

### 新增

- `frontend/src/views/cd/ApplicationCreatePage.vue`
  - 独立应用创建页面，承载标题、返回按钮和创建向导。
- `frontend/src/components/ApplicationCreateWizard.vue`
  - 创建方式选择、Stepper、从镜像表单、高级空白表单、确认页、失败恢复 UI。
- `frontend/src/utils/applicationCompose.ts`
  - 生成 `docker-compose.yml` 和 `.env` 的纯函数。
- `frontend/src/utils/applicationCompose.test.ts`
  - 覆盖 compose/env 生成、env key 校验、多 volume 输出等纯函数逻辑。

### 修改

- `frontend/src/views/cd/ApplicationPage.vue`
  - 主按钮跳转 `/cd/applications/create`。
  - 保留现有导入流程。
- `frontend/src/i18n/locales/zh-CN.ts`
  - 新增创建向导中文文案、校验文案和失败恢复文案。
  - 将主入口文案调整为“创建应用”。
- `frontend/src/i18n/locales/en-US.ts`
  - 同步英文文案。
- `frontend/src/types/cd/application.ts`
  - 如需要，为创建向导增加本地可复用类型；若类型只在组件内部使用，则不修改此文件。

### 不修改

- `backend-go/**`
  - 本次不新增聚合接口，不改 backend-go。
- 数据库迁移文件
  - 不修改。
- 旧 Python backend
  - 不修改。

## Implementation steps / 实施步骤

1. 接受 requirement 并创建本 plan 文档。
2. 新增 `applicationCompose.ts`：
   - 实现环境变量、volume 归一化。
   - 实现 `.env` 与 `docker-compose.yml` 字符串生成。
   - 实现必要的校验辅助函数。
3. 新增 `applicationCompose.test.ts`：
   - 环境变量生成 `.env`。
   - compose 在有 env 时包含 `env_file: .env`。
   - 多条 volumes 正确输出。
   - 没有 env/volume 时不输出对应节点。
4. 新增 `ApplicationCreatePage.vue`：
   - 提供独立创建路由页面。
   - 展示页面标题、说明和返回应用列表按钮。
   - 承载 `ApplicationCreateWizard`。
5. 新增 `ApplicationCreateWizard.vue`：
   - 不使用 `AppDialog`。
   - 使用 Reka Stepper primitives 实现 3 步进度。
   - 第一阶段展示 4 种方式卡片。
   - “从镜像创建”进入表单和确认步骤。
   - “高级空白应用”进入原字段表单和确认步骤。
   - 其他方式展示后续支持/现有入口说明。
5. 在 `ApplicationCreateWizard.vue` 中实现从镜像表单：
   - 基础字段。
   - 环境变量多行增删。
   - volume 多行增删。
   - 可选 route domain。
   - 表单校验和错误展示。
6. 在 `ApplicationCreateWizard.vue` 中实现提交串联：
   - 创建应用。
   - upsert `docker-compose.yml`。
   - upsert 或保守处理 `.env`。
   - upsert route。
   - 失败后保留 `createdApplication`，允许重新提交继续补齐。
   - 提供跳转详情页手动修复。
7. 修改 `ApplicationPage.vue`：
   - 主按钮文案和点击行为改为跳转 `/cd/applications/create`。
   - 保留当前导入按钮与导入逻辑。
8. 更新 zh-CN/en-US 文案。
9. 运行前端测试和检查。
10. 若检查暴露 lint/type 错误，按项目规范修复。

## Verification plan / 验证计划

Implementation 完成后先停在实现阶段，不创建 verification 文档；用户进入 Verification 后再执行以下验证。

计划命令：

```bash
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn test -- applicationCompose
cd /d/SourceCodes/mywork/pomelo-orbit/frontend && yarn lint --fix && yarn typecheck
```

说明：

- `yarn lint --fix && yarn typecheck` 是前端变更后的强制检查。
- 因本计划不修改 backend-go，默认不运行 Go 测试；若 Implementation 实际触及 backend-go，再补充相关 `go test`。
- 不启动/停止/重启开发服务器。

建议人工验收点：

1. 打开应用列表页，主入口显示“创建应用”，点击后跳转 `/cd/applications/create`。
2. 独立创建页面展示 Stepper，第一步展示 4 种创建方式。
3. 选择“从镜像创建”，填写必填字段后可进入确认并提交。
4. 创建完成后，应用拥有 `docker-compose.yml`。
5. 填写环境变量时，应用拥有 `.env`，且 compose 包含 `env_file: .env`。
6. 填写多条数据目录时，compose 包含多条 `volumes`。
7. 填写域名时，应用启用路由托管并创建 service `app` 的路由。
8. 模拟配置文件或路由创建失败后，重新提交继续补齐同一个已创建应用，并可跳转详情页手动修复。

## Rollback / 回滚思路

- 前端回滚 `ApplicationCreateWizard.vue`、`applicationCompose.ts`、相关测试和 i18n 文案即可恢复旧创建入口。
- 因不修改 backend-go 和数据库迁移，不涉及数据结构回滚。
- 若用户已通过向导创建应用，产生的是现有模型中的 application/config files/routes，可继续用现有详情页编辑或删除。

## Risks / 风险

1. 前端串联接口不具备事务性。虽然重试会继续补齐同一个 `createdApplication`，但浏览器刷新或关闭后局部状态丢失，无法自动识别之前的半成品应用；此时用户需要从详情页手动修复。
2. `ports` 使用同端口主机映射，可能遇到宿主端口冲突；这是为了遵从当前 backend-go 默认端口解析的最小实现。
3. `.env` value 仅做基础限制，不实现完整 dotenv 语法编辑器；复杂值场景后续由 P2 `.env` 专用编辑体验处理。
4. 其他创建方式本次只展示，不完整落地，可能让用户期待后续支持；UI 文案必须明确“后续支持”或指向现有导入入口。
5. 直接在新组件中使用 Reka Stepper primitives，后续如果其他页面也需要 Stepper，应再抽取共享组件。

## User review notes / 用户审查记录

- 用户确认环境变量使用 `.env`。
- 用户确认数据目录支持多条映射。
- 用户确认本次只完整实现“从镜像创建”。
- 用户确认不新增 backend-go 聚合接口，采用前端串联现有接口。
- 用户确认串联失败后重新提交应继续补齐同一个应用。
- 用户确认失败恢复提供跳转详情页手动修复。
- 用户要求了解并遵从当前实现，本计划据此保留现有 backend-go 解析模型并生成 `ports`。
