# Project 级 CD 接管数据传输验证
最后修改时间: 2026-09-17 18:06:46

Review status: Accepted

Mode: standard

## Intent alignment

- 接管以 Project 的 CD 配置闭包为单位，提供 Orbit 内部的下载、上传与 Web 操作入口；没有调用或扩展 Python `database-transfer`、dbtalk 或 JSONL。
- 包格式为 `pomelo-orbit/project-handover` v1 单一 JSON 文档。编码移除根 Project ID 及所有直接归属 `project_id`；解码拒绝这些字段和未知字段。
- 模式 A 通过 Project 领域创建 Project 并写入当前操作者成员关系；模式 B 校验成员关系、保护 CI 引用、清除既有 CD 配置后以目标 Project ID 恢复配置，保留原成员。
- Environment、SSH credential、Application / Version / Component、Gateway、Service 与 Route 均经所属领域的完整定义读取、创建和删除。接管协调层不访问 SQL，不以表顺序实现事务脚本。
- SSH 私钥在接管包中使用 `encrypted_private_key`；导入使用提交的解密 Key，缺省时回退到实例 system key，解密后再由目标 Environment 持久化。
- 导入保留 Service 的包内状态，Route 继续禁用；不调用 Docker、SSH、workspace、Traefik、证书或 Probe。
- 既有跨领域删除已调整为由拥有数据的领域处理，跨领域引用由应用服务/仓储读取并拒绝或显式协调，不再依赖隐藏级联。

## Actual diff

- 新增 `internal/application/project_handover/{dto,usecase}`，并接入 Project HTTP handler、route 和 bootstrap。
- 扩展 Project、Environment、Application、Gateway、Service、Route、Pipeline、PipelineRun、Deployment 等领域的定义读取、恢复、引用保护或删除边界。
- 新增跨领域限制的前向 migration `000045` 与 Role/User assignment 限制的前向 migration `000046`；没有修改已执行 migration。
- Project Web 详情页提供一致样式的导出与导入按钮；项目列表提供新建/覆盖模式、文件选择和目标 Project 选择。
- 清理原有两个 Vue 测试文件的多组件 lint warning，保留原断言并将同步弹窗断言归属到组件测试文件。

## Acceptance

- [x] 接管 API 与 UI 为 Orbit 应用能力，不依赖 Python transfer/dbtalk/JSONL。
- [x] API 支持下载 JSON attachment 和 multipart 上传；UI 支持新建 Project 与覆盖已有 Project。
- [x] 包不携带源 Project identity；内部源 ID 只用于恢复期间的关系映射。
- [x] Environment 与 SSH credential 作为正常业务数据恢复；SSH 私钥在包中加密，导入使用自定义或 system key 解密，目标持久化时使用当前实例密钥加密。
- [x] 导入后的 Service status 保持包内原值，Route 仍为禁用，且用例不触发运行时副作用。
- [x] 覆盖模式在同一 HTTP request UoW 内处理 Project 更新、CD 数据替换、Environment 保存和 Deployment history 清理；失败由事务切面回滚。
- [x] 模式 B 在 CD 配置清理前由 Pipeline / PipelineRun 拒绝会留下失效引用的操作。
- [x] 跨领域删除边界已由领域能力与前向迁移收敛。

## Test results

| Command | Result |
| --- | --- |
| `yarn --cwd web lint:fix` | Passed |
| `yarn --cwd web typecheck` | Passed |
| `yarn --cwd web vitest run src/views/route/routeSync.test.ts src/components/RouteSyncDialog.test.ts src/views/service/components/ServiceEnvironmentCard.test.ts` | 4/4 passed |
| `task check` | Passed: Web typecheck/lint/format and Go format/static analysis, 0 issues |
| `task test` | Passed: Web 26 files / 117 tests; Go `./cmd/... ./internal/... ./sql` passed |
| `go test ./internal/repository/impl/sqlc/role -count=1 -v` | Passed: 3/3 after one non-reproducible full-suite retry failure |

## Scope and residual risk

- 本轮按确认的主线覆盖原则，没有额外增加 HTTP multipart 或真实跨实例端到端矩阵。接管 DTO 的严格编解码、Application 协调顺序、Environment/Version/Route/Service 领域集成和全仓标准测试已覆盖关键路径。
- 导入不会接管目标服务器上已有容器、workspace 或 Traefik provider 状态；首次后续显式部署/路由同步才会根据导入配置收敛，这属于既定产品边界。

## Conclusion

实现与已接受的 Intent/Plan 一致，标准质量检查与测试均通过，可以验收。
