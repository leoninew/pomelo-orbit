# 移除应用导入导出
最后修改时间: 2026-08-15 13:04:30

Review status: Accepted

## Background

当前应用的导入与导出不是可往返的数据契约：导出包含应用、所有版本和服务实例，导入却只接受应用与一个初始版本。该业务场景不再需要，继续暴露入口会造成错误预期和维护负担。

## Goal

彻底移除 Application 的导入、导出能力，包括前端入口、HTTP API、应用用例、传输 DTO 与 Proto 定义。

## Non-goal

- 不改变普通应用创建、版本创建和部署能力。
- 不删除凭据等其他领域的导入或导出能力。
- 不保留兼容路由、旧 DTO 或旧 Proto 契约。

## User scenarios

1. 用户在应用列表只能创建、编辑和删除应用，不能选择 JSON 文件导入。
2. 用户在应用详情不能再下载应用 JSON。
3. 客户端和服务端均不存在应用导入、导出 API。

## Acceptance

1. `/applications` 不再显示应用导入控件或导入对话框。
2. 应用详情不再显示导出操作。
3. `POST /api/application/import` 与 `GET /api/application/:app_id/export` 不再注册。
4. 不再生成或引用 `application_bundle.proto`、`ApplicationImportReq`、`ApplicationExportResp` 及对应应用用例。
5. 前端与后端检查通过。

## Open questions

暂无需要用户确认的未决事项。

## Decisions

1. 此变更删除整个应用导入导出业务能力，而非仅隐藏 UI。
2. 删除不对称的数据契约，不保留兼容层或替代格式。

## Risk

已有自动化或外部客户端若调用已删除端点会收到 404；这是本次明确的破坏性 API 变更。

## User review notes

- 用户先要求审视应用详情导出逻辑，并确认准备删除该业务场景。
- 用户随后明确要求删除导出和导入能力，需求阶段视为 Accepted 并直接进入 Implementation。
