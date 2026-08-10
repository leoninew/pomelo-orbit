# 服务组件有效端点响应验证
最后修改时间: 2026-08-10

Review status: Accepted

## Requirement Alignment

实现让服务详情组件的 `effective_endpoints` 直接反映 `BuildEffectiveServicePlan` 的组件端点合并结果；原有稀疏覆盖字段保持其既有语义。

## Spec / Plan Alignment

不适用。本任务采用 light 流程，依据 Requirement 实现。

## Actual Diff Summary

- `ServiceView` 保存成功构建的 EffectiveServicePlan components；计划构建失败时继续通过现有 `effective_error` 表达，未合成任何替代端点。
- `ServiceComponentResp` 新增只读 `effective_endpoints`，并由 HTTP mapper 按 `service_component_id` 投影对应的有效组件端点。
- 原 `endpoints` 保持 Service 端点稀疏覆盖的读写语义；没有改变其字段或要求前端合并。
- Go 与 TypeScript proto DTO 由 `task proto` 重新生成。
- 服务详情组件表新增端口列，以模式标签和“协议/容器端口”文本呈现端点，并将 `internal` 端点排除在已发布端口之外。

## Acceptance Checklist

- [x] Service view 保存有效计划中的组件端点。
- [x] HTTP/proto 响应按 Service component 映射输出 `effective_endpoints`。
- [x] 覆盖后的 mode 与监听端口在响应中可见，协议与容器端口保留 Version 契约。
- [x] 服务详情组件表显示生效的已发布端口，无发布端口时端口单元格留空。
- [x] 生成 DTO、Go 检查、前端 lint 和 typecheck 完成。

## Test Results

- `task proto` 通过。
- 定向测试 `go test ./internal/application/service/usecase ./internal/api/http/handler/service` 通过。
- `go fmt ./cmd/... ./internal/...`、`go vet ./cmd/... ./internal/...`、`go test ./cmd/... ./internal/...` 全部通过。
- `yarn --cwd web test` 通过：11 个测试文件、64 个测试（包含端口展示单元测试）。
- `yarn --cwd web lint:fix` 与 `yarn --cwd web typecheck` 通过。

## Risks And Incomplete Items

- 本次不提供可点击的 Gateway URL 或运行时端口探测；端口列只展示有效计划声明的发布方式与端口。
- 未启动开发服务器，符合仓库约束。

## Conclusion

服务详情现可在单次 Service view 响应中获得每个组件的有效端点，且端点值与部署计划使用的合并结果一致。
