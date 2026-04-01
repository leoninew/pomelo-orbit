"""Checkout Action 实现"""

import json
import logging
import tempfile
from pathlib import Path
from typing import Any

from pomelo_orbit.domain.ci.entities import Credential
from pomelo_orbit.domain.ci.value_objects import CredentialType, StepDefinition
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor

logger = logging.getLogger(__name__)


class CheckoutAction:
    """Checkout Action - 拉取代码"""

    def __init__(self, container_executor: ContainerExecutor):
        self.container_executor = container_executor

    async def execute(
        self,
        step: StepDefinition,
        repository_url: str,
        credential: Credential,
        trigger_ref: str,
        workspace_path: Path,
        artifacts_path: Path,
        run_id: str,
    ) -> dict[str, Any]:
        """
        执行 checkout

        Args:
            step: Step 定义
            repository_url: 仓库地址
            credential: 凭据
            trigger_ref: 分支/tag/commit
            workspace_path: workspace 路径
            artifacts_path: artifacts 路径
            run_id: Run ID

        Returns:
            outputs 字典
        """
        # 解析参数
        inputs = step.inputs or {}
        depth = inputs.get("depth", 1)
        ref = inputs.get("ref", trigger_ref)
        sparse_checkout = inputs.get("sparse_checkout")
        submodules = inputs.get("submodules", False)

        # 解密凭据
        credential_data = json.loads(credential.encrypted_data)

        # 准备环境变量和卷
        environment = {}
        volumes = []
        temp_key_path = None

        try:
            if credential.type == CredentialType.GIT_SSH:
                # SSH 私钥注入
                private_key = credential_data.get("private_key", "")
                with tempfile.NamedTemporaryFile(mode="w", delete=False, suffix=".key") as temp_key_file:
                    temp_key_file.write(private_key)
                    temp_key_path = temp_key_file.name

                # 设置权限
                Path(temp_key_path).chmod(0o600)

                # 挂载私钥到容器
                volumes.append(f"{temp_key_path}:/root/.ssh/id_rsa:ro")
                environment["GIT_SSH_COMMAND"] = "ssh -i /root/.ssh/id_rsa -o StrictHostKeyChecking=no"

            elif credential.type == CredentialType.GIT_TOKEN:
                # HTTPS token 注入
                token = credential_data.get("token", "")
                # 将 token 内嵌到 URL
                if repository_url.startswith("https://"):
                    repository_url = repository_url.replace("https://", f"https://{token}@")

            # 构造 git clone 命令
            commands = []

            # 基础 clone 命令
            clone_cmd = f"git clone --depth={depth}"
            if ref:
                clone_cmd += f" --branch={ref}"
            if sparse_checkout:
                clone_cmd += " --no-checkout"
            clone_cmd += f" {repository_url} /workspace"
            commands.append(clone_cmd)

            # Sparse checkout
            if sparse_checkout:
                commands.append("cd /workspace")
                commands.append("git sparse-checkout init --cone")
                commands.append(f"git sparse-checkout set {sparse_checkout}")
                commands.append("git checkout")

            # Submodules
            if submodules:
                commands.append("cd /workspace && git submodule update --init --recursive")

            # 提取 outputs
            commands.append("cd /workspace")
            commands.append("git log -1 --format='%H' > /tmp/commit_sha")
            commands.append("git log -1 --format='%s' > /tmp/commit_message")
            commands.append("git log -1 --format='%an' > /tmp/author")
            commands.append("git log -1 --format='%aI' > /tmp/committed_at")

            # 执行容器
            exit_code, logs = await self.container_executor.run(
                image="alpine/git",
                commands=commands,
                volumes=volumes,
                environment=environment,
                workspace_path=workspace_path,
                artifacts_path=artifacts_path,
            )

            if exit_code != 0:
                raise Exception(f"Checkout 失败: {logs}")

            # 读取 outputs
            outputs = {}
            try:
                outputs["commit_sha"] = (workspace_path.parent / "tmp" / "commit_sha").read_text().strip()
                outputs["commit_message"] = (workspace_path.parent / "tmp" / "commit_message").read_text().strip()
                outputs["author"] = (workspace_path.parent / "tmp" / "author").read_text().strip()
                outputs["committed_at"] = (workspace_path.parent / "tmp" / "committed_at").read_text().strip()
            except Exception as e:
                logger.warning(f"读取 checkout outputs 失败: {e}")
                outputs = {
                    "commit_sha": ref,
                    "commit_message": "",
                    "author": "",
                    "committed_at": "",
                }

            return outputs

        finally:
            # 清理临时私钥文件
            if temp_key_path:
                try:
                    Path(temp_key_path).unlink(missing_ok=True)
                except Exception as e:
                    logger.warning(f"清理临时私钥文件失败: {e}")
