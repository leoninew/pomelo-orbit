"""Checkout Action 实现"""

import logging
from pathlib import Path
from typing import Any

from pomelo_orbit.domain.ci.entities import Credential
from pomelo_orbit.domain.ci.value_objects import CredentialType, StepDefinition
from pomelo_orbit.infrastructure.ci.container import ContainerExecutor
from pomelo_orbit.infrastructure.ci.workspace import get_secrets_path

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

        # 准备环境变量和卷
        environment: dict[str, str] = {}
        extra_binds: dict[str, dict[str, str]] = {}

        if credential.type == CredentialType.GIT_SSH:
            # 写入 run 专用的 secrets 目录，挂载到容器 /run/secrets，不污染 workspace
            secrets_path = get_secrets_path(run_id)
            secrets_path.mkdir(parents=True, exist_ok=True)
            key_path = secrets_path / "id_rsa"
            key_path.write_text(credential.get_private_key())
            key_path.chmod(0o600)
            extra_binds[str(secrets_path)] = {"bind": "/run/secrets", "mode": "rw"}
            environment["GIT_SSH_COMMAND"] = "ssh -i /run/secrets/id_rsa -o StrictHostKeyChecking=no"

        elif credential.type == CredentialType.GIT_TOKEN:
            token = credential.get_token()
            # 如果是 SSH 格式（git@host:user/repo.git），转换为 HTTPS 格式
            if repository_url.startswith("git@"):
                # git@github.com:user/repo.git -> https://token@github.com/user/repo.git
                without_prefix = repository_url[len("git@") :]
                host, path = without_prefix.split(":", 1)
                repository_url = f"https://{token}@{host}/{path}"
            elif repository_url.startswith("https://"):
                repository_url = repository_url.replace("https://", f"https://{token}@")

        # 构造 git clone 命令
        commands = []

        if credential.type == CredentialType.GIT_SSH:
            # Windows 上 chmod 不生效，在容器内修正私钥权限
            commands.append("chmod 600 /run/secrets/id_rsa")

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

        # 提取 outputs（写到 /workspace/.git_outputs，随 workspace 挂载同步到宿主机）
        commands.append("cd /workspace")
        commands.append("git log -1 --format='%H' > /workspace/.git_outputs_commit_sha")
        commands.append("git log -1 --format='%s' > /workspace/.git_outputs_commit_message")
        commands.append("git log -1 --format='%an' > /workspace/.git_outputs_author")
        commands.append("git log -1 --format='%aI' > /workspace/.git_outputs_committed_at")

        # 执行容器
        logger.info(f"Checkout: credential_type={credential.type}, ref={ref}")
        exit_code, logs = await self.container_executor.run(
            image="alpine/git",
            commands=commands,
            volumes=[],
            environment=environment,
            workspace_path=workspace_path,
            artifacts_path=artifacts_path,
            extra_binds=extra_binds,
        )

        if exit_code != 0:
            raise Exception(f"Checkout 失败: {logs}")

        # 读取 outputs
        outputs = {}
        try:
            outputs["commit_sha"] = (workspace_path / ".git_outputs_commit_sha").read_text().strip()
            outputs["commit_message"] = (workspace_path / ".git_outputs_commit_message").read_text().strip()
            outputs["author"] = (workspace_path / ".git_outputs_author").read_text().strip()
            outputs["committed_at"] = (workspace_path / ".git_outputs_committed_at").read_text().strip()
        except Exception as e:
            logger.warning(f"读取 checkout outputs 失败: {e}")
            outputs = {
                "commit_sha": ref,
                "commit_message": "",
                "author": "",
                "committed_at": "",
            }

        return outputs
