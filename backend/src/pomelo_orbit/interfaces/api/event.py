"""
回调事件 API
"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query
from sqlalchemy.orm import Session

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.infrastructure.persistence.di import get_db
from pomelo_orbit.infrastructure.persistence.models import WebhookEventModel
from pomelo_orbit.infrastructure.time_utils import from_iso8601
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.schemas import (
    PaginatedResp,
    WebhookEventDetailResp,
    WebhookEventResp,
)

router = APIRouter(prefix="/webhook-event", tags=["webhook-event"])


@router.get("", response_model=PaginatedResp[WebhookEventResp])
def list_events(
    db: Annotated[Session, Depends(get_db)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
    source: str | None = None,
    status_filter: Annotated[str | None, Query(alias="status")] = None,
    date_from: Annotated[str | None, Query()] = None,
    date_to: Annotated[str | None, Query()] = None,
) -> PaginatedResp[WebhookEventResp]:
    """列出 回调事件"""
    query = db.query(WebhookEventModel)

    if search:
        query = query.filter(
            (WebhookEventModel.repository_name.contains(search)) | (WebhookEventModel.sender.contains(search))
        )
    if source:
        query = query.filter(WebhookEventModel.source == source)
    if status_filter:
        query = query.filter(WebhookEventModel.status == status_filter)
    if date_from:
        query = query.filter(WebhookEventModel.received_at >= from_iso8601(date_from))
    if date_to:
        query = query.filter(WebhookEventModel.received_at < from_iso8601(date_to))

    total = query.count()
    offset = (page - 1) * per_page
    events = query.order_by(WebhookEventModel.received_at.desc()).offset(offset).limit(per_page).all()

    return PaginatedResp(
        items=[WebhookEventResp.model_validate(e) for e in events],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.get("/{event_id}", response_model=WebhookEventDetailResp)
def get_event(
    event_id: str,
    db: Annotated[Session, Depends(get_db)],
    _current_user=Depends(get_current_user),
) -> WebhookEventDetailResp:
    """获取 回调事件详情"""
    event = db.query(WebhookEventModel).filter(WebhookEventModel.id == event_id).first()
    if not event:
        raise BusinessError("Event not found", status_code=404)
    return WebhookEventDetailResp.model_validate(event)
