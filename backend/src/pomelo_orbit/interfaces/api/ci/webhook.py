"""Webhook 接收 API（公开入口）和项目 Webhook 管理 API"""

import fnmatch
import json
import logging
from typing import Annotated

from dishka import AsyncContainer
from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, BackgroundTasks, Depends, Header, HTTPException, Query, Request

from pomelo_orbit.application.ci.di import get_pipeline_run_service, get_webhook_service
from pomelo_orbit.application.ci.pipeline_run_service import PipelineRunService
from pomelo_orbit.application.ci.webhook_service import WebhookService
from pomelo_orbit.domain.ci.value_objects import PipelineRunTrigger
from pomelo_orbit.interfaces.api.ci.dto.webhook import (
    ProjectWebhookCreateReq,
    ProjectWebhookResp,
    ProjectWebhookUpdateReq,
)
from pomelo_orbit.interfaces.api.utils import run_in_new_scope

logger = logging.getLogger(__name__)

router = APIRouter(tags=["webhooks"])


@router.get("/repository/{repository_id}/webhook", response_model=list[ProjectWebhookResp])
def list_webhooks(
    repository_id: str,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    project_id: Annotated[str, Query()],
) -> list[ProjectWebhookResp]:
    return [ProjectWebhookResp.model_validate(wh) for wh in webhook_service.list_webhooks(project_id, repository_id)]


@router.post("/repository/{repository_id}/webhook", response_model=ProjectWebhookResp, status_code=201)
def create_webhook(
    repository_id: str,
    data: ProjectWebhookCreateReq,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    project_id: Annotated[str, Query()],
) -> ProjectWebhookResp:
    wh = webhook_service.create_webhook(
        project_id=project_id,
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
    project_id: Annotated[str, Query()],
) -> ProjectWebhookResp:
    wh = webhook_service.update_webhook(
        project_id=project_id,
        repository_id=repository_id,
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
    project_id: Annotated[str, Query()],
) -> None:
    webhook_service.delete_webhook(project_id, repository_id, webhook_id)


@router.post("/webhook/{webhook_id}")
@inject
async def receive_webhook(
    webhook_id: str,
    request: Request,
    background_tasks: BackgroundTasks,
    webhook_service: Annotated[WebhookService, Depends(get_webhook_service)],
    pipeline_run_service: Annotated[PipelineRunService, Depends(get_pipeline_run_service)],
    container: FromDishka[AsyncContainer],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
    x_gitlab_token: Annotated[str, Header(alias="X-Gitlab-Token")] = "",
) -> dict:
    wh = webhook_service.get_webhook(webhook_id)

    if not wh.enabled:
        return {"status": "ignored", "reason": "webhook disabled"}

    payload_bytes = await request.body()
    try:
        payload = json.loads(payload_bytes)
    except json.JSONDecodeError:
        logger.warning("Webhook payload is not valid JSON")
        raise HTTPException(status_code=400, detail="Invalid JSON payload")

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

    if not wh.branch_filter:
        logger.info(f"Webhook branch filtered: no branch_filter configured, webhook={wh.id}")
        return {"status": "ignored", "reason": "branch filtered"}

    if wh.branch_filter != "*" and not fnmatch.fnmatch(branch, wh.branch_filter):
        logger.info(f"Webhook branch filtered: branch={branch}, filter={wh.branch_filter}")
        return {"status": "ignored", "reason": "branch filtered"}

    repository = webhook_service.get_repository_for_webhook(wh)
    result = pipeline_run_service.create_run(
        project_id=repository.project_id,
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

    async def _run() -> None:
        await run_in_new_scope(
            container,
            PipelineRunService,
            lambda svc: svc.execute_run(result.run, result.repository, result.merged_variables, result.snapshot),
        )

    background_tasks.add_task(_run)
    logger.info(f"Webhook triggered: source={source}, run={result.run.id}, ref={branch}")
    return {"status": "triggered", "run_id": result.run.id}
