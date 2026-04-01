"""CI Webhook 应用服务 - 处理 Git webhook 触发 pipeline"""

import logging

from pomelo_orbit.application.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.repositories import ProjectRepository
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.infrastructure.ci.webhook_payload_parser import (
    CIWebhookPayload,
    WebhookPayloadParseError,
    parse_github_webhook,
    parse_gitlab_webhook,
)
from pomelo_orbit.infrastructure.ci.webhook_verifier import (
    verify_github_signature,
    verify_gitlab_signature,
)

logger = logging.getLogger(__name__)


class CIWebhookService:
    """CI Webhook 服务"""

    def __init__(
        self,
        project_repo: ProjectRepository,
        pipeline_service: PipelineService,
    ):
        self.project_repo = project_repo
        self.pipeline_service = pipeline_service

    def handle_github_webhook(
        self,
        payload_bytes: bytes,
        payload: dict,
        signature: str,
    ) -> dict:
        """
        处理 GitHub webhook

        Args:
            payload_bytes: 原始请求体（用于签名验证）
            payload: 解析后的 JSON payload
            signature: X-Hub-Signature-256 header 值

        Returns:
            处理结果 dict
        """
        # 解析 payload
        try:
            ci_payload = parse_github_webhook(payload)
        except WebhookPayloadParseError as e:
            logger.warning(f"GitHub webhook payload parse failed: {e}")
            return {"status": "ignored", "reason": str(e)}

        return self._handle_webhook(
            source="github",
            ci_payload=ci_payload,
            payload_bytes=payload_bytes,
            signature=signature,
        )

    def handle_gitlab_webhook(
        self,
        payload: dict,
        token: str,
    ) -> dict:
        """
        处理 GitLab webhook

        Args:
            payload: 解析后的 JSON payload
            token: X-Gitlab-Token header 值

        Returns:
            处理结果 dict
        """
        try:
            ci_payload = parse_gitlab_webhook(payload)
        except WebhookPayloadParseError as e:
            logger.warning(f"GitLab webhook payload parse failed: {e}")
            return {"status": "ignored", "reason": str(e)}

        return self._handle_webhook(
            source="gitlab",
            ci_payload=ci_payload,
            payload_bytes=None,
            signature=token,
        )

    def _handle_webhook(
        self,
        source: str,
        ci_payload: CIWebhookPayload,
        payload_bytes: bytes | None,
        signature: str,
    ) -> dict:
        """
        核心处理逻辑

        1. 根据 repository_url 查找匹配的 Project
        2. 验证签名
        3. 检查 branch_filter
        4. 触发 PipelineRun
        """
        repository_url = ci_payload.repository_url
        projects = self.project_repo.find_by_repository_url(repository_url)

        if not projects:
            logger.info(
                f"No matching project for webhook: source={source}, "
                f"repo={repository_url}"
            )
            return {"status": "ignored", "reason": "no matching project"}

        triggered_runs = []
        errors = []

        for project in projects:
            try:
                # 验证签名
                if project.webhook_secret:
                    if source == "github" and payload_bytes is not None:
                        valid = verify_github_signature(payload_bytes, signature, project.webhook_secret)
                    elif source == "gitlab":
                        valid = verify_gitlab_signature(signature, project.webhook_secret)
                    else:
                        valid = False

                    if not valid:
                        logger.warning(
                            f"Webhook signature verification failed: "
                            f"source={source}, project={project.id}"
                        )
                        continue

                # 检查 branch_filter
                if project.branch_filter and ci_payload.branch:
                    allowed = [b.strip() for b in project.branch_filter.split(",")]
                    if ci_payload.branch not in allowed:
                        logger.info(
                            f"Branch filtered: project={project.id}, "
                            f"branch={ci_payload.branch}, allowed={allowed}"
                        )
                        continue

                # 触发 pipeline
                run, triggered_project, merged_vars = self.pipeline_service.create_run(
                    project_id=project.id,
                    trigger=PipelineRunTrigger.WEBHOOK,
                    trigger_ref=ci_payload.branch or ci_payload.commit_sha,
                    runtime_variables={
                        "commit_sha": ci_payload.commit_sha,
                        "author": ci_payload.author,
                        "event_type": ci_payload.event_type,
                    },
                )
                triggered_runs.append({
                    "project_id": project.id,
                    "run_id": run.id,
                    "run": run,
                    "project": triggered_project,
                    "merged_vars": merged_vars,
                })
                logger.info(
                    f"Pipeline triggered via webhook: source={source}, "
                    f"project={project.id}, run={run.id}, ref={ci_payload.branch}"
                )

            except Exception as e:
                logger.error(
                    f"Failed to trigger pipeline: project={project.id}, error={e}",
                    exc_info=True,
                )
                errors.append({"project_id": project.id, "error": str(e)})

        if not triggered_runs and not errors:
            return {"status": "ignored", "reason": "branch filtered"}

        return {
            "status": "triggered",
            "triggered": triggered_runs,
            "errors": errors,
        }
