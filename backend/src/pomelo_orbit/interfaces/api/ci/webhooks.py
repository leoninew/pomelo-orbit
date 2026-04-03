"""Webhook 接收 API"""

import json
import logging
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Depends, Header, Request

from pomelo_orbit.application.ci.di import get_ci_webhook_service
from pomelo_orbit.application.ci.webhook_service import CIWebhookService

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/webhooks", tags=["webhooks"])


@router.post("/git")
async def receive_git_webhook(
    request: Request,
    background_tasks: BackgroundTasks,
    ci_webhook_service: Annotated[CIWebhookService, Depends(get_ci_webhook_service)],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
    x_gitlab_token: Annotated[str, Header(alias="X-Gitlab-Token")] = "",
) -> dict:
    payload_bytes = await request.body()
    payload = json.loads(payload_bytes)

    if x_hub_signature_256:
        result = ci_webhook_service.handle_github_webhook(
            payload_bytes=payload_bytes, payload=payload, signature=x_hub_signature_256
        )
    elif x_gitlab_token:
        result = ci_webhook_service.handle_gitlab_webhook(payload=payload, token=x_gitlab_token)
    else:
        result = ci_webhook_service.handle_github_webhook(payload_bytes=payload_bytes, payload=payload, signature="")

    pipeline_service = ci_webhook_service.pipeline_service
    for item in result.get("triggered", []):
        background_tasks.add_task(pipeline_service.execute_run, item["run"], item["project"], item["merged_vars"])

    return {
        "status": result["status"],
        "triggered": [{"project_id": t["project_id"], "run_id": t["run_id"]} for t in result.get("triggered", [])],
        "errors": result.get("errors", []),
        "reason": result.get("reason"),
    }
