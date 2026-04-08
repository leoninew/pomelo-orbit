
- 应用详情里凭据获取请求  /api/ci/credentials?per_page=100 应该推迟到编辑弹窗打开时加载
- 项目详情里保存各信息实质调用的同一个接口  PUT /api/ci/templates/01KNN8G1R25HT6R3QG5ZWY8XFM
- 流水线详情 PipelineRun 里没有各 Stage 的状态
- 流水线异步任务状态管理复杂 backend\src\pomelo_orbit\infrastructure\ci\executor_impl.py
- 持续部署里 port 应该用变量占位