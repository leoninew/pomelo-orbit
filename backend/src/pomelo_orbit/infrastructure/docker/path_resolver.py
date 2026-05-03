"""Docker 容器路径解析器

在容器内运行时，需要将容器内路径转换为宿主机物理路径，
以便启动的子容器能正确挂载宿主机目录。
"""

import json
import logging
import subprocess
from pathlib import Path

from pomelo_orbit.infrastructure.config import get_project_root

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


class PhysicalPathResolver:
    """物理路径解析器：将容器内路径转换为宿主机物理路径

    用于在容器内运行时，将容器内的路径（如 /app/data/ci/xxx）转换为
    宿主机的物理路径（如 /opt/pomelo-orbit/data/ci/xxx），以便启动
    子容器时能正确挂载宿主机目录。
    """

    def __init__(self, container_mount_point: str = "/app/data"):
        """
        Args:
            container_mount_point: 容器内的数据目录挂载点（默认 /app/data）
        """
        self.container_mount_point = container_mount_point
        self._physical_data_dir: str | None = None

    def get_physical_data_dir(self) -> str:
        """获取宿主机的 data 目录物理路径

        - 容器内运行：通过 docker inspect 获取挂载点的宿主机路径
        - 本地运行：返回项目根目录下的 data 路径

        Returns:
            宿主机的 data 目录绝对路径
        """
        if self._physical_data_dir is not None:
            return self._physical_data_dir

        container_id = detect_container_id()
        if container_id:
            try:
                source = get_container_mount_source(container_id, self.container_mount_point)
                logger.info(f"Detected physical data dir: {source} (container mode)")
                self._physical_data_dir = source
                return source
            except Exception as e:
                logger.warning(f"Failed to detect physical data dir: {e}, falling back to local path")

        # 本地开发模式
        local_path = str(get_project_root() / "data")
        logger.info(f"Using local data dir: {local_path}")
        self._physical_data_dir = local_path
        return local_path

    def convert_to_physical_path(self, container_path: Path) -> str:
        """将容器内路径转换为宿主机物理路径

        Args:
            container_path: 容器内的路径（如 /app/data/ci/xxx/workspace）

        Returns:
            宿主机的物理路径（如 /opt/pomelo-orbit/data/ci/xxx/workspace）

        Examples:
            >>> resolver = PhysicalPathResolver()
            >>> # 容器模式：/app/data/ci/xxx -> /opt/pomelo-orbit/data/ci/xxx
            >>> resolver.convert_to_physical_path(Path("/app/data/ci/xxx"))
            '/opt/pomelo-orbit/data/ci/xxx'
            >>> # 本地模式：/project/data/ci/xxx -> /project/data/ci/xxx
            >>> resolver.convert_to_physical_path(Path("/project/data/ci/xxx"))
            '/project/data/ci/xxx'
        """
        container_data_dir = get_project_root() / "data"

        # 检查路径是否在 data 目录下
        try:
            relative_path = container_path.relative_to(container_data_dir)
        except ValueError:
            # 路径不在 data 目录下，直接返回
            return str(container_path)

        # 转换为宿主机路径
        physical_data_dir = self.get_physical_data_dir()
        physical_path = Path(physical_data_dir) / relative_path
        return str(physical_path)
