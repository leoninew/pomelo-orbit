# 服务容器版本关联计划
最后修改时间: 2026-07-28 15:23:46

Review status: Accepted

## Implementation steps

1. 就地更新 SQLite 与 MySQL 服务建表迁移，删除最近成功版本列。
2. 删除 Service 模型、SQL 查询、仓储、DTO 与 Proto 中的对应字段。
3. 渲染 Compose 时注入容器版本标签；运行时状态查询用受限 `docker inspect` 格式读取标签。
4. 扩展容器状态 API 与服务详情容器表，展示可跳转的版本。
5. 将 `data/db/pomelo-orbit.db` 就地更新为与服务建表脚本一致的结构。

## Files to change

`sql/migration/`、`sql/query/service/`、服务/部署 usecase 与仓储、Proto 生成输入、服务详情 Vue 页面及相关测试。

## Verification plan

- 重新生成 sqlc 与 Proto 代码。
- 运行 Go 格式化、vet、测试，以及前端 lint/typecheck。
- 查询运行中的 SQLite，确认列已删除且迁移版本为 30。

## Assumptions and risks

历史容器不会补写标签，版本列会在下一次部署后出现。运行时 `docker inspect` 仅格式化输出容器 ID 和 labels，不读取环境变量等敏感配置。

## Rollback

回退需在服务建表脚本与运行中 SQLite 一并恢复该列；容器标签本身为附加元数据，可保留。
