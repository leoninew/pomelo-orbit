# Service Code Deployment Directory Verification
最后修改时间: 2026-08-09 23:07:32

Review status: Draft

## Requirement Alignment

需求文档状态为 `Accepted`。实现将部署工作目录、Compose 工作目录和部署日志目录统一改为 Service code 根目录；普通 directory 挂载在渲染前物化；RAGFlow 集成式和拆分式合同均改为组件目录 bind source。

## Spec Alignment

不适用。本任务采用标准模式，但未单独创建 Spec。

## Plan Alignment

不适用。本任务采用标准模式，但未单独创建 Plan。

## Actual Diff Summary

- 部署工作区 API 改为接收 `serviceCode`，调用方、运行态查询、预览、部署日志和删除 Application 语义同步调整。
- directory mount 现在会在渲染前创建；RAGFlow 合同、Compose 参考、TEI 准备脚本和说明改为服务根下的组件目录。
- 前端部署目录显示改为 Service code。
- 文档更新了 CD 运行时、领域模型和 RAGFlow 运维说明。
- 本地已停止服务的数据迁移为 `cli-proxy-api-default` 与 `ragflow-integrated-default`；后者的 `es01`、`minio`、`mysql`、`redis` 和 `ragflow` 目录已扁平化。孤立备份 `data/deployment/ragflow-integrated/default.rar` 已按用户明确指示删除。

## Expected Vs Actual Changed Files

预期范围为部署工作区、部署 use case、MCP/HTTP 接口、前端目录显示和 RAGFlow 合同/引用。实际 Git 变更位于这些区域及其测试、活文档和 Requirement 文档，未发现无关产品模块变更；`git diff --check HEAD` 通过。

本地数据处置是用户在实现后明确扩展的范围，已回写 Requirement 的 User Review Notes；未对运行服务或没有 Service 映射的目录作推断性移动。

## Acceptance Checklist

- [x] 服务工作目录、Compose 工作目录和部署日志目录以 Service code 为根。
- [x] 相对 directory mount 以服务根解析，且在部署前物化。
- [x] 集成式与拆分式 RAGFlow Compose 共 8 份引用中 `./data/` 匹配数为 0；不再使用 RAGFlow named volume。
- [x] 集成式 MySQL 容器使用 `MYSQL_DATABASE=rag_flow` 初始化数据库；RAGFlow 容器使用 `MYSQL_DBNAME=rag_flow` 连接该数据库，两者职责与值一致。
- [x] 数据库中 `data/...` 或 `./data/...` 挂载记录为 0；集成式 CPU/GPU Version 的 12 条挂载均以对应组件目录起始。
- [x] 已停止服务的数据目录迁移后字节数保持：`cli-proxy-api-default` 为 3,419,648，`ragflow-integrated-default` 为 375,999,566。
- [x] 旧 `default` 路径、RAGFlow 内嵌 `data` 路径、`ragflow-cpu` 路径和 `default.rar` 均不存在；`tei/cache` 与 `ragflow/logs` 存在。

## Test Results

- `go fmt ./cmd/... ./internal/...` passed.
- `go vet ./cmd/... ./internal/...` passed.
- `go test ./cmd/... ./internal/...` passed.
- `yarn --cwd web lint:fix` passed.
- `yarn --cwd web typecheck` passed.
- `python skills/deploy-ragflow-integrated-orbit/test_render_contract.py` passed (5 tests).
- `python skills/deploy-ragflow-split-orbit/test_render_contract.py` passed (5 tests).
- `git diff --check HEAD` passed.

## Risks And Incomplete Items

- `mineru-api/default` 与 `traefik/default` 对应运行服务，未移动；Traefik 的空证书目录 `traefik/data/certs` 也保持原状。它们必须在服务停止且迁移计划明确后再处理。
- `nginx/default` 与 `postgres/default` 没有数据库 Service 记录，无法安全推导目标 Service code，保持原状。
- 本次未执行部署生命周期操作或运行态部署验证。

## Conclusion

代码、RAGFlow 合同、数据库挂载记录和已授权的停止服务数据均符合 Service code 目录规则。运行中的服务和未映射遗留目录已明确保留，等待单独的数据处置决策。
