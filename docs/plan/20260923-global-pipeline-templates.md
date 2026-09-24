# 全局流水线模板与构建阶段实施计划
最后修改时间: 2026-09-24 08:49:42

Review status: Accepted

Mode: standard

## Implementation steps

1. 就地调整三种数据库的 pipeline schema 定义：模板 Pipeline 和模板阶段清除 Project 归属，阶段表允许模板行使用 NULL；开发库同步执行数据归一化。
2. 调整 Pipeline SQL 查询、模板阶段 SQL 查询和 SQLC 生成代码，区分全局模板与项目内 Application Pipeline/Stage。
3. 调整 Pipeline UseCase 的创建、读取、更新、删除、实例化和模板阶段更新边界；保留当前 Project 成员授权。
4. 增加 Repository/UseCase 测试，覆盖跨 Project 可见、模板名全局冲突和 Application Pipeline 项目隔离。
5. 更新产品与 CI Pipeline 活文档，运行 SQLC、Go 测试、前端类型和 lint 检查。

## Expected files

- `sql/migration/{postgres,mysql,sqlite}/000020_pipeline.up.sql`
- `sql/query/pipeline/pipeline.sql`
- `internal/model/pipeline.go`
- `internal/repository/{pipeline.go,impl/sqlc/pipeline/*}`
- `internal/application/pipeline/usecase/{pipeline.go,stage_template.go}`
- `internal/repository/impl/sqlc/pipeline/repository_test.go`
- `docs/product/overview.md`、`docs/guides/ci-pipeline-design.md`

## Verification

- SQLC 生成无差异。
- Pipeline Repository 测试和 Pipeline UseCase 测试通过。
- `DBTALK_DSN_APP` 中现有 Template Pipeline 与模板阶段的 `project_id` 均为 `NULL`，Application Pipeline 与 Application Stage 仍保留所属 Project。
- `go test ./cmd/... ./internal/...` 通过；前端只做文档和后端契约未改变时不扩大前端测试范围。
