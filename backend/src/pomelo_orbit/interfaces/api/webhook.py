"""
Webhook API 路由
"""

import json
import logging
from typing import Annotated

from dynaconf import Dynaconf
from fastapi import APIRouter, BackgroundTasks, Depends, Header, HTTPException, Request, status
from sqlalchemy.orm import Session
from ulid import ULID

from pomelo_orbit.application.application_service import ApplicationService
from pomelo_orbit.application.di import get_application_service
from pomelo_orbit.application.webhook_parser import parse_github_payload
from pomelo_orbit.domain.value_objects import DeployStatus
from pomelo_orbit.infrastructure import verify_github_signature
from pomelo_orbit.infrastructure.config import get_settings
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.mappers import ApplicationMapper, DeploymentMapper
from pomelo_orbit.infrastructure.persistence.models import (
    ApplicationModel,
    DeploymentModel,
    GitSourceModel,
    WebhookEventModel,
)
from pomelo_orbit.infrastructure.time_utils import utc_now

router = APIRouter(prefix="/hooks", tags=["webhooks"])

logger = logging.getLogger(__name__)


@router.post("/github")
async def github_webhook(
    request: Request,
    background_tasks: BackgroundTasks,
    x_github_event: Annotated[str, Header(alias="X-Github-Event")],
    db: Annotated[Session, Depends(get_db)],
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    settings: Annotated[Dynaconf, Depends(get_settings)],
    x_hub_signature_256: Annotated[str, Header(alias="X-Hub-Signature-256")] = "",
):
    """接收 Github Webhook"""
    payload_bytes = await request.body()
    payload = json.loads(payload_bytes)

    parsed = parse_github_payload(x_github_event, payload)
    if not parsed:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Unsupported event type: {x_github_event}",
        )

    # 提取镜像信息
    deployment_info = payload.get("deployment", {})
    image_name = deployment_info.get("image") if deployment_info else None

    # 查找匹配的应用(通过git_source关联)
    application = (
        db.query(ApplicationModel)
        .join(GitSourceModel)
        .filter(
            GitSourceModel.repository_url.contains(parsed.repository_name),
            ApplicationModel.enabled.is_(True),
        )
        .first()
    )

    # 基于系统配置验证签名
    webhook_secret: str = settings.webhook.secret
    assert webhook_secret, "webhook.secret must be configured for webhook endpoint"
    signature_valid = verify_github_signature(payload_bytes, x_hub_signature_256, webhook_secret)

    # 记录事件
    event = WebhookEventModel(
        source="github",
        event_type=parsed.event_type,
        repository_name=parsed.repository_name,
        repository_url=parsed.repository_url,
        branch=parsed.branch,
        sender=parsed.sender,
        image_name=image_name,
        payload=json.dumps(payload),
        signature_valid=signature_valid,
        status="received",
    )
    db.add(event)
    db.flush()  # 获取 ID，但不提交事务
    db.refresh(event)

    logger.info(f"Webhook received: source=github, event={x_github_event}, repo={parsed.repository_name}")

    if x_github_event == "ping":
        event.status = "processed"
        event.processed_at = utc_now()
        return {"message": "pong"}

    if not application:
        event.status = "ignored"
        event.error_message = "No matching application found"
        event.processed_at = utc_now()
        return {"message": "No matching application", "event_id": event.id}

    event.matched_application_id = application.id

    if signature_valid is False:
        event.status = "error"
        event.error_message = "Invalid signature"
        event.processed_at = utc_now()
        raise HTTPException(status_code=status.HTTP_401_UNAUTHORIZED, detail="Invalid signature")

    git_source = application.git_source
    if not git_source or not git_source.auto_deploy:
        event.status = "ignored"
        event.error_message = "Auto deploy disabled"
        event.processed_at = utc_now()
        return {"message": "Auto deploy disabled", "event_id": event.id}

    allowed_branches = [b.strip() for b in git_source.deploy_branches.split(",")]
    if parsed.branch not in allowed_branches:
        event.status = "ignored"
        event.error_message = f"Branch {parsed.branch} not in allowed list"
        event.processed_at = utc_now()
        return {"message": "Branch not allowed", "event_id": event.id}

    # 创建部署记录
    deployment = DeploymentModel(
        id=str(ULID()),
        application_id=application.id,
        application_name=application.name,
        trigger_type="webhook",
        trigger_ref=parsed.branch,
        webhook_event_id=event.id,
        image_name=image_name,
        status=DeployStatus.QUEUED.value,
        started_at=utc_now(),
    )
    db.add(deployment)
    db.flush()
    db.refresh(deployment)

    # 更新事件记录
    event.triggered_deployment_id = deployment.id
    event.status = "matched"
    event.processed_at = utc_now()

    # 提交事务，确保后台任务能读取到数据
    db.commit()

    # 添加后台任务执行部署
    app_entity = ApplicationMapper.to_domain(application)
    deployment_entity = DeploymentMapper.to_domain(deployment)
    background_tasks.add_task(
        app_service.deploy,
        app_entity,
        deployment_entity,
    )

    logger.info(
        f"Deployment queued via webhook: app={application.name}, deployment={deployment.id}, branch={parsed.branch}"
    )

    return {
        "message": "Deploy triggered",
        "event_id": event.id,
        "deployment_id": deployment.id,
        "application_id": application.id,
    }
