# TODO / 待设计

## 待实现

- 持续部署里 port 应该用变量占位（CD 模块）
- 制品（Artifact）领域：制品存储、版本管理、跨 Run 引用
- 触发弹窗重构：去掉独立的 trigger_ref 输入框，改为完整变量列表（`repository_ref` 可编辑）
- 流水线模板详情页"运行"按钮：有未保存变更时已拦截，后续考虑支持"保存并运行"快捷操作

## 已知技术债

- `backend/src/pomelo_orbit/infrastructure/ci/executor_impl.py`：流水线异步任务状态管理复杂，cancel/retry 逻辑与 asyncio task 生命周期耦合较深，需要梳理
