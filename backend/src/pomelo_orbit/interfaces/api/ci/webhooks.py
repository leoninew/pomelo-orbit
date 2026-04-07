"""Webhook 接收 API（公开入口）和项目 Webhook 管理 API"""

import fnmatch
import json
import logging
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Header, Request

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.infrastructure.ci.webhook_verifier import verify_github_signature, verify_gitlab_signature
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.webhook import (
    ProjectWebhookCreateReq,
    ProjectWebhookResp,
    ProjectWebhookUpdateReq,
)

logger = logging.getLogger(__name__)

# 最大 payload 大小：10MB
MAX_PAYLOAD_SIZE = 10 * 1024 * 1024

router = APIRouter(tags=["webhooks"])


# ── 项目 Webhook 管理（需认证）────────────────────────────────────────────────


@router.get("/projects/{project_id}/webhooks", response_model=list[ProjectWebhookResp])
def list_webhooks(
    project_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> list[ProjectWebhookResp]:
    return [ProjectWebhookResp.model_validate(wh) for wh in pipeline_service.list_webhooks(project_id)]


@router.post("/projects/{project_id}/webhooks", response_model=ProjectWebhookResp, status_code=201)
def create_webhook(
    project_id: str,
    data: ProjectWebhookCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectWebhookResp:
    wh = pipeline_service.create_webhook(
        project_id=project_id,
        name=data.name,
        template_id=data.template_id,
        plain_secret=data.secret,
        branch_filter=data.branch_filter,
    )
    return ProjectWebhookResp.model_validate(wh)


@router.put("/projects/{project_id}/webhooks/{webhook_id}", response_model=ProjectWebhookResp)
def update_webhook(
    project_id: str,
    webhook_id: str,
    data: ProjectWebhookUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> ProjectWebhookResp:
    wh = pipeline_service.update_webhook(
        webhook_id=webhook_id,
        name=data.name,
        template_id=data.template_id,
        branch_filter=data.branch_filter,
        plain_secret=data.secret,
        enabled=data.enabled,
    )
    return ProjectWebhookResp.model_validate(wh)


@router.delete("/projects/{project_id}/webhooks/{webhook_id}", status_code=204)
def delete_webhook(
    project_id: str,
    webhook_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_webhook(webhook_id)


# ── Git 平台推送入口（公开，无需认证）────────────────────────────────────────


@router.post("/webhooks/{webhook_id}")
async def receive_webhook(
    webhook_id: str,
    request: Request,
    background_tasks: BackgroundTasks,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
    x_gitlab_token: Annotated[str, Header(alias="X-Gitlab-Token")] = "",
) -> dict:
    # 查找 webhook 配置（不存在时 service 层抛 BusinessError 404）
    wh = pipeline_service.get_webhook(webhook_id)

    if not wh.enabled:
        return {"status": "ignored", "reason": "webhook disabled"}

    # 验证 payload 大小
    payload_bytes = await request.body()
    if len(payload_bytes) > MAX_PAYLOAD_SIZE:
        logger.warning("Webhook payload too large")
        return {"status": "ignored", "reason": "payload too large"}

    # 解析 JSON
    try:
        payload = json.loads(payload_bytes)
    except json.JSONDecodeError:
        logger.warning("Webhook payload is not valid JSON")
        return {"status": "ignored", "reason": "invalid json"}

    # 解密 secret 并验证签名
    decrypted_secret = pipeline_service.decrypt_webhook_secret(wh)

    if x_hub_signature_256:
        if not verify_github_signature(payload_bytes, x_hub_signature_256, decrypted_secret):
            logger.warning("Webhook signature verification failed")
            return {"status": "ignored", "reason": "signature verification failed"}
        source = "github"
        branch = payload.get("ref", "").removeprefix("refs/heads/")
        commit_sha = payload.get("after", "")
        author = payload.get("pusher", {}).get("name", "")
    elif x_gitlab_token:
        if not verify_gitlab_signature(x_gitlab_token, decrypted_secret):
            logger.warning("Webhook token verification failed")
            return {"status": "ignored", "reason": "signature verification failed"}
        source = "gitlab"
        branch = payload.get("ref", "").removeprefix("refs/heads/")
        commit_sha = payload.get("checkout_sha", "")
        author = payload.get("user_name", "")
    else:
        return {"status": "ignored", "reason": "missing signature header"}

    # 分支过滤：None/空字符串表示拒绝所有分支，"*" 表示接受所有分支，其他值用 glob 匹配
    if not wh.branch_filter:
        logger.info(f"Webhook branch filtered: no branch_filter configured, webhook={wh.id}")
        return {"status": "ignored", "reason": "branch filtered"}

    if wh.branch_filter != "*" and not fnmatch.fnmatch(branch, wh.branch_filter):
        logger.info(f"Webhook branch filtered: branch={branch}, filter={wh.branch_filter}")
        return {"status": "ignored", "reason": "branch filtered"}

    # 触发流水线（branch 为空时 fallback 到 commit_sha，create_run 内再 fallback 到 default_branch）
    run, proj, merged_vars, snapshot = pipeline_service.create_run(
        project_id=wh.project_id,
        template_id=wh.template_id,
        trigger=PipelineRunTrigger.WEBHOOK,
        trigger_ref=branch or commit_sha,
        runtime_variables={
            "commit_sha": commit_sha,
            "author": author,
            "event_type": "push",
        },
    )
    background_tasks.add_task(pipeline_service.execute_run, run, proj, merged_vars, snapshot)
    logger.info(f"Webhook triggered: source={source}, run={run.id}, ref={branch}")
    return {"status": "triggered", "run_id": run.id}
