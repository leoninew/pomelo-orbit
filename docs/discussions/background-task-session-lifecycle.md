# 后台任务与 Session 生命周期

## 问题

`PipelineService.execute_run` 是一个后台任务（由 FastAPI `BackgroundTasks` 调度），它在 HTTP 响应发送后才执行。此时请求 session 已关闭，因此必须在内部自己创建独立 session：

```python
# 当前实现 — pipeline_service.py
async def execute_run(self, run, project, variables):
    with self._session_factory() as session:   # 业务层直接管理 session 生命周期
        run_repo = PipelineRunRepositoryImpl(session)
        executor = self._executor_factory(session)
        ...
```

这导致 `PipelineService`（应用层）直接依赖 `session_factory`（基础设施概念），违反了分层原则。

## 业界主流方案

### 方案 A：Unit of Work 模式

service 持有 `UnitOfWork` 而非 repo，每次操作显式开启工作单元：

```python
async with self.uow:
    run = await self.uow.runs.find_by_id(run_id)
    run.start()
    await self.uow.commit()
```

后台任务和请求路径使用同一套接口，session 生命周期对业务完全透明。改动面较大，需引入 UoW 抽象层。

### 方案 B：任务队列（Celery / ARQ）

后台任务作为独立 worker 运行，每个任务有完整的独立 DI 生命周期，根本不存在 session 泄漏问题。生产环境主流选择，但引入运维复杂度。

### 方案 C：run_executor 工厂（最小改动）

把 session 生命周期封装进注入的 `run_executor` callable，`PipelineService` 只知道"调用它来执行"：

```python
# ci_di.py — session 管理集中在这里
def make_run_executor(session_factory, executor_factory):
    async def run_executor(run, context, definition):
        with session_factory() as session:
            run_repo = PipelineRunRepositoryImpl(session)
            executor = executor_factory(session)
            ...
    return run_executor

# pipeline_service.py — 业务层不再感知 session
async def execute_run(self, run, project, variables):
    ...
    await self._run_executor(run, context, definition)
```

## 现状

当前采用折中方案：`session_factory` 和 `executor_factory` 作为构造参数注入，业务层通过工厂调用而非直接 `new` 具体类。虽然 session 管理仍在业务层，但依赖是可替换的，测试可以 mock。

待后续评审决定是否推进方案 C 或引入 UoW。
