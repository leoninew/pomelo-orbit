"""Docker 容器路径解析器

在容器内运行时，需要将容器内路径转换为宿主机物理路径，
以便启动的子容器能正确挂载宿主机目录。
"""

import json
import logging
import subprocess
from pathlib import Path

logger = logging.getLogger(__name__)


def detect_container_id() -> str | None:
    """检测当前进程是否在容器内运行，返回容器 ID

    Returns:
        容器 ID（64位十六进制字符串）或 None（本地运行）
    """
    # cgroup v1: /proc/self/cgroup 包含 docker/<container-id>
    try:
        cgroup = Path("/proc/self/cgroup")
        if cgroup.is_file():
            for line in cgroup.read_text().splitlines():
                if "docker" in line or "kubepods" in line:
                    for part in reversed(line.split("/")):
                        if len(part) == 64 and all(c in "0123456789abcdef" for c in part):
                            return part
    except Exception:
        pass

    # cgroup v2: /proc/self/cgroup 只有 "0::/"，改从 /proc/self/mountinfo 提取
    # 其中 /etc/hostname 挂载行包含 /data/docker/containers/<64-hex-id>/hostname
    try:
        mountinfo = Path("/proc/self/mountinfo")
        if mountinfo.is_file():
            for line in mountinfo.read_text().splitlines():
                if "/etc/hostname" in line:
                    for part in line.split("/"):
                        if len(part) == 64 and all(c in "0123456789abcdef" for c in part):
                            return part
    except Exception:
        pass

    return None


def get_container_mount_source(container_id: str, destination: str) -> str:
    """获取容器指定挂载点的宿主机源路径

    Args:
        container_id: 容器 ID
        destination: 容器内的挂载目标路径（如 /app/data）

    Returns:
        宿主机的源路径

    Raises:
        RuntimeError: 未找到指定的挂载点
    """
    result = subprocess.run(
        ["docker", "inspect", "--format", "{{json .Mounts}}", container_id],
        capture_output=True,
        text=True,
        timeout=5,
        check=True,
    )
    mounts = json.loads(result.stdout)
    for mount in mounts:
        if mount.get("Type") == "bind" and mount.get("Destination") == destination:
            source = mount.get("Source")
            if source:
                return str(source)
    raise RuntimeError(f"Mount point {destination} not found in container {container_id}")
