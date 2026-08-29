# RAGFlow 部署工作流能力改进验证
最后修改时间: 2026-08-03 12:38:42

Review status: Draft

流程模式: 标准 / standard

## Verification scope

本次只验证已实现的 deployment contract authority：split 与 integrated 拓扑的机器可读契约、Compose/primitive 生成、两份 skill 的漂移检查入口，以及运行时变量分配校验。按用户要求，不运行集成测试，不执行 Docker Compose、MCP、Orbit 或服务生命周期操作。

## Requirement alignment

- split 与 integrated 的 Compose、primitive 和 skill 由 topology-specific 契约及共享渲染器约束，满足本轮配置权威来源目标。
- `MYSQL_ROOT_HOST=%` 已同时进入两种拓扑的契约和生成 Compose。
- `TEI_BASE_URL` 仅作为生成 primitive 中的手工 Provider 地址，不是 `ragflow-cpu` 容器环境变量。
- 运行时变量只以键名和适用 Service 进入契约，未写入实际值。

其余 Requirement 中的 MCP 网络可用性、Gateway 供应、Deployment 查询和 TEI registry 缓存改进不在本轮实现和验证范围内。

## Spec alignment

- `scripts/ragflow-split/deployment-contract.json` 与 `scripts/ragflow-integrated/deployment-contract.json` 分别描述各自的拓扑配置和数据边界。
- `scripts/render_ragflow_deployment_contract.py --write|--check` 统一生成或比较两种拓扑的 Compose 与 primitive。
- `deploy-ragflow-split-orbit` 和 `deploy-ragflow-integrated-orbit` 都先运行通用 `--check`，再从各自生成的 primitive 获取配置。

## Actual diff summary

- 新增 integrated 部署契约、通用渲染入口及其说明文档。
- 由契约重生成 bundled 与 split Compose、两份 deployment primitive。
- 将两份 skill 收敛为流程编排，移除重复的镜像、网络别名、环境变量和 endpoint 配置。
- 新增或扩展渲染器单元测试，覆盖 split/integrated 生成物一致性、运行时键分配漂移和通用检查入口。

## Expected And Actual Files

| 预期范围 | 实际文件 |
| --- | --- |
| topology 契约与渲染器 | `scripts/ragflow-split/deployment-contract.json`、`scripts/ragflow-integrated/deployment-contract.json`、`scripts/render_ragflow_deployment_contract.py`、兼容入口 |
| 派生 Compose 与 primitive | `scripts/ragflow-split/docker-compose.*.yml`、`scripts/ragflow-bundled/docker-compose*.yml`、两份 skill reference |
| 流程与测试 | 两份部署 skill、渲染器测试与脚本文档 |

## Acceptance Checklist

- [x] 两种拓扑均有机器可读配置权威来源。
- [x] 生成物漂移会使 `--check` 非零退出。
- [x] skill 不再重复维护组件运行配置。
- [x] `MYSQL_ROOT_HOST` 与手工 Provider 地址处于正确的契约边界。
- [x] 基本单元与静态检查通过。
- [ ] MCP、Gateway、Deployment discovery 和 TEI preflight 的独立实现与验收不在本轮范围内。
- [ ] 集成测试按用户要求未运行。

## Test Results

| 命令 | 结果 |
| --- | --- |
| `python scripts/render_ragflow_deployment_contract.py --check` | 通过 |
| `python -m unittest scripts/test_render_ragflow_split_contract.py` | 通过，5 tests |
| `python -m ruff check scripts/render_ragflow_split_contract.py scripts/render_ragflow_deployment_contract.py scripts/test_render_ragflow_split_contract.py` | 通过 |
| `git diff --check` | 通过 |

## Risks And Incomplete Items

- 此记录不证明两种 Compose 可在目标主机完成启动，也不验证模型、镜像、网络或外部依赖；这些属于未运行的集成验证。
- 现有工作区包含用户或先前流程已暂存的变更；本次未执行暂存、提交、回退或任何运行时写操作。

## Conclusion

配置契约子范围的基本验证通过。由于用户排除了集成测试，且更广泛的 MCP/Gateway/TEI 工作尚未在本轮实现，Verification 保持 `Draft`。
