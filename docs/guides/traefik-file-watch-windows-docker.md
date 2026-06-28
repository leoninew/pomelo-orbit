# Traefik 文件监听在 Windows + Docker 环境下的问题

## 问题现象

在 Windows + Docker Desktop 环境下，通过目录挂载方式使用 Traefik 的文件提供者（File Provider）时，动态添加或修改路由配置文件后，Traefik 无法自动检测到变更，必须重启容器才能生效。

## 环境信息

- 操作系统：Windows 10/11
- Docker：Docker Desktop
- Traefik：3.x
- 挂载方式：`./data/dynamic:/etc/traefik/dynamic`
- 配置：`watch: true`

## 技术原因

### Traefik 文件监听机制

通过阅读 Traefik 源码（`pkg/provider/file/file.go`），了解到：

1. **使用 fsnotify 库**：跨平台的文件系统事件监听库
2. **监听策略**：
   - Directory 模式：监听目录本身 + 目录下所有一级文件
   - 不监听子目录（忽略子目录变更）
3. **事件处理**：
   - 任何文件事件（Create/Write/Remove）都触发配置重新加载
   - 目录本身的变更事件也会触发重新加载
4. **没有定时轮询机制**：
   - Traefik 文件提供者不支持定时轮询
   - 只有两种触发方式：fsnotify 事件监听 或 SIGHUP 信号
   - 启动时加载一次配置，之后完全依赖事件驱动

### 问题根源

**跨文件系统边界的事件传播失败**

在 Windows + Docker Desktop 环境下：
- 主机文件系统：NTFS/ReFS
- Docker 容器内：Linux 文件系统（通过 WSL2 或 Hyper-V）
- 文件系统事件（inotify）无法可靠地从主机传播到容器内

当主机上的 Python 程序写入 `backend/data/applications/traefik/data/dynamic/*.yml` 时：
1. 主机文件系统产生变更事件
2. Docker Desktop 需要将事件传播到容器内的 Linux 文件系统
3. **事件传播失败或延迟**，导致容器内的 fsnotify 无法接收到事件
4. Traefik 认为配置未变更，不触发重新加载

## 解决方案

### 方案对比

| 方案 | 测试结果 | 适用场景 | 优缺点 |
|------|---------|---------|--------|
| os.utime() 触发目录时间戳 | ❌ 失效 | - | 事件无法跨文件系统边界传播 |
| SIGHUP 信号 | ✅ 有效 | Windows + Docker | 可靠但需要 Docker 权限 |
| 手动重启容器 | ✅ 有效 | 所有环境 | 简单但不够优雅 |
| 轮询模式 | 未测试 | 所有环境 | 有延迟，增加负载 |

### 最终方案：SIGHUP 信号（仅 Windows）

**实现策略**：
- Linux 环境：依赖 fsnotify（正常工作）
- Windows 环境：写入配置后发送 SIGHUP 信号

**核心实现**：

```python
def _reload_traefik(self) -> None:
    """发送 SIGHUP 信号给 Traefik 容器，强制重新加载配置（仅 Windows 需要）"""
    import platform

    if platform.system() != "Windows":
        return

    try:
        # 检查容器是否运行
        result = subprocess.run(
            ["docker", "ps", "-q", "-f", f"name={self.traefik_container}"],
            capture_output=True,
            text=True,
            check=True,
        )
        if not result.stdout.strip():
            return

        # 发送 SIGHUP 信号
        subprocess.run(
            ["docker", "kill", "--signal=HUP", self.traefik_container],
            capture_output=True,
            text=True,
            check=True,
        )
        logger.info(f"Sent SIGHUP to Traefik container: {self.traefik_container}")
    except subprocess.CalledProcessError as e:
        logger.warning(f"Failed to reload Traefik: {e.stderr.strip()}")
    except FileNotFoundError:
        logger.warning("Docker command not found")
```

**配置项**：

```yaml
# config.defaults.yaml
traefik:
  dynamic_config_dir: "data/applications/traefik/data/dynamic"
  container_name: "traefik"  # 可通过环境变量覆盖
```

**实现特点**：

1. **平台判断**：只在 Windows 下执行，Linux 依赖 fsnotify
2. **先检查后发送**：避免无效的 Docker 命令调用
3. **容错处理**：
   - 容器未运行：静默跳过
   - 发送失败：warning 日志
   - Docker 不可用：warning 日志

**调用时机**：
- 创建路由：`write_route_file()` 后
- 更新路由：`write_route_file()` 后
- 删除路由：`delete_route_file()` 后

## Traefik API 调查

Traefik 的 API（8080 端口）只提供**只读查询接口**，所有路由都是 `GET` 方法：
- `/api/rawdata` - 获取运行时配置
- `/api/http/routers` - 查询路由
- `/api/http/services` - 查询服务

**没有任何 POST/PUT 接口来触发配置重载**，因此无法通过 API 实现配置热更新。

- Traefik 源码：`/d/Works/opensource/traefik/pkg/provider/file/file.go`
- fsnotify 库：https://github.com/fsnotify/fsnotify
- Docker Desktop 文件系统：https://docs.docker.com/desktop/windows/wsl/

## 测试验证

验证方案是否生效：

1. 添加或修改路由配置
2. 观察 Traefik 日志：`docker logs -f traefik`
3. 检查 Traefik Dashboard：http://localhost:8080
4. 测试路由是否立即生效

预期日志输出：
```
time="..." level=info msg="Configuration reloaded" providerName=file
```

## 结论

### 问题根源

- Traefik 文件提供者完全依赖 fsnotify 事件驱动，没有轮询机制
- Windows + Docker 环境下，跨文件系统边界的事件传播不可靠
- Traefik API 不提供配置重载接口

### 解决方案总结

| 环境 | 方案 | 状态 |
|------|------|------|
| Linux + Docker | fsnotify 自动监听 | ✅ 正常工作 |
| Windows + Docker | SIGHUP 信号触发 | ✅ 已实现 |

### 实现状态

**已实现**：
- ✅ 平台判断：Windows 下自动发送 SIGHUP，Linux 下跳过
- ✅ 容器检查：先检查容器是否运行再发送信号
- ✅ 异常处理：失败时记录 warning 日志，不影响主流程
- ✅ 配置灵活：通过 `traefik.container_name` 配置容器名

**测试验证**：
```bash
# 1. 添加路由
curl -X POST http://localhost:9020/api/route -d '{"domain":"test.local","target_url":"http://localhost:8000"}'

# 2. 观察 Traefik 日志
docker logs -f traefik

# 3. 检查 Dashboard
open http://localhost:8080

# 预期日志
# time="..." level=info msg="Configuration reloaded" providerName=file
```

### 相关文件

- 实现：`backend/src/pomelo_orbit/application/route_service.py`
- 配置：`backend/config.defaults.yaml`
- 调用：`backend/src/pomelo_orbit/interfaces/api/route.py`
- 调用：`backend/src/pomelo_orbit/interfaces/api/application.py`
