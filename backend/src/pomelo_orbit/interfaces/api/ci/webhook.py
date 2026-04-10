"""Webhook 接收 API（公开入口）和项目 Webhook 管理 API"""

import fnmatch
import json
import logging
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Header, HTTPException, Request

from pomelo_orbit.application.ci.di import get_pipeline_run_service, get_webhook_service
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.application.ci.webhook_service import WebhookService
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.webhook import (
    ProjectWebhookCreateReq,
    ProjectWebhookResp,
    ProjectWebhookUpdateReq,
)

logger = logging.getLogger(__name__)

router = APIRouter(tags=["webhooks"])


# ── 项目 Webhook 管理（需认证）────────────────────────────────────────────────


@router.get("/repository/{repository_id}/webhook", response_model=list[ProjectWebhookResp])
def list_webhooks(
    repository_id: str,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    _current_user=Depends(get_current_user),
) -> list[ProjectWebhookResp]:
    return [ProjectWebhookResp.model_validate(wh) for wh in webhook_service.list_webhooks(repository_id)]


@router.post("/repository/{repository_id}/webhook", response_model=ProjectWebhookResp, status_code=201)
def create_webhook(
    repository_id: str,
    data: ProjectWebhookCreateReq,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    _current_user=Depends(get_current_user),
) -> ProjectWebhookResp:
    wh = webhook_service.create_webhook(
        repository_id=repository_id,
        name=data.name,
        template_id=data.template_id,
        plain_secret=data.secret,
        branch_filter=data.branch_filter,
    )
    return ProjectWebhookResp.model_validate(wh)


@router.put("/repository/{repository_id}/webhook/{webhook_id}", response_model=ProjectWebhookResp)
def update_webhook(
    repository_id: str,
    webhook_id: str,
    data: ProjectWebhookUpdateReq,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    _current_user=Depends(get_current_user),
) -> ProjectWebhookResp:
    wh = webhook_service.update_webhook(
        webhook_id=webhook_id,
        name=data.name,
        template_id=data.template_id,
        branch_filter=data.branch_filter,
        plain_secret=data.secret,
        enabled=data.enabled,
    )
    return ProjectWebhookResp.model_validate(wh)


@router.delete("/repository/{repository_id}/webhook/{webhook_id}", status_code=204)
def delete_webhook(
    repository_id: str,
    webhook_id: str,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    _current_user=Depends(get_current_user),
) -> None:
    webhook_service.delete_webhook(webhook_id)


# ── Git 平台推送入口（公开，无需认证）────────────────────────────────────────


@router.post("/webhook/{webhook_id}")
async def receive_webhook(
    webhook_id: str,
    request: Request,
    background_tasks: BackgroundTasks,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
    x_gitlab_token: Annotated[str, Header(alias="X-Gitlab-Token")] = "",
) -> dict:
    # 查找 webhook 配置（不存在时 service 层抛 BusinessError 404）
    wh = webhook_service.get_webhook(webhook_id)

    if not wh.enabled:
        return {"status": "ignored", "reason": "webhook disabled"}

    # 解析 JSON
    payload_bytes = await request.body()
    try:
        payload = json.loads(payload_bytes)
    except json.JSONDecodeError:
        logger.warning("Webhook payload is not valid JSON")
        raise HTTPException(status_code=400, detail="Invalid JSON payload")

    # 解密 secret 并验证签名
    decrypted_secret = webhook_service.decrypt_webhook_secret(wh)

    if x_hub_signature_256:
        if not webhook_service.verify_webhook_signature("github", payload_bytes, x_hub_signature_256, decrypted_secret):
            logger.warning("Webhook signature verification failed")
            raise HTTPException(status_code=401, detail="Invalid signature")
        source = "github"
        branch = payload.get("ref", "").removeprefix("refs/heads/")
        commit_sha = payload.get("after", "")
        author = payload.get("pusher", {}).get("name", "")
    elif x_gitlab_token:
        if not webhook_service.verify_webhook_signature("gitlab", b"", x_gitlab_token, decrypted_secret):
            logger.warning("Webhook token verification failed")
            raise HTTPException(status_code=401, detail="Invalid token")
        source = "gitlab"
        branch = payload.get("ref", "").removeprefix("refs/heads/")
        commit_sha = payload.get("checkout_sha", "")
        author = payload.get("user_name", "")
    else:
        raise HTTPException(status_code=401, detail="Missing signature header")

    # 分支过滤：None/空字符串表示拒绝所有分支，"*" 表示接受所有分支，其他值用 glob 匹配
    if not wh.branch_filter:
        logger.info(f"Webhook branch filtered: no branch_filter configured, webhook={wh.id}")
        return {"status": "ignored", "reason": "branch filtered"}

    if wh.branch_filter != "*" and not fnmatch.fnmatch(branch, wh.branch_filter):
        logger.info(f"Webhook branch filtered: branch={branch}, filter={wh.branch_filter}")
        return {"status": "ignored", "reason": "branch filtered"}

    result = pipeline_run_service.create_run(
        repository_id=wh.repository_id,
        template_id=wh.template_id,
        trigger=PipelineRunTrigger.WEBHOOK,
        trigger_ref=branch or commit_sha,
        runtime_variables={
            "commit_sha": commit_sha,
            "author": author,
            "event_type": "push",
        },
    )
    background_tasks.add_task(
        pipeline_run_service.execute_run, result.run, result.repository, result.merged_variables, result.snapshot
    )
    logger.info(f"Webhook triggered: source={source}, run={result.run.id}, ref={branch}")
    return {"status": "triggered", "run_id": result.run.id}
