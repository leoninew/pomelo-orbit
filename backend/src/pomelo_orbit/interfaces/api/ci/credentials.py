"""凭据管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.di import get_pipeline_service
from pomelo_orbit.application.ci.pipeline_service import PipelineService
from pomelo_orbit.infrastructure.di import get_security_service
from pomelo_orbit.infrastructure.security import SecurityService
from pomelo_orbit.interfaces.api.auth.router import get_current_user
from pomelo_orbit.interfaces.api.ci.dto.credential import CredentialCreateReq, CredentialResp, CredentialUpdateReq
from pomelo_orbit.interfaces.api.common import PaginatedResp

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/credentials", tags=["credentials"])


@router.get("", response_model=PaginatedResp[CredentialResp])
def list_credentials(
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[CredentialResp]:
    creds, total = pipeline_service.list_credentials(page=page, per_page=per_page)
    return PaginatedResp(
        items=[CredentialResp.model_validate(c.__dict__) for c in creds],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=CredentialResp, status_code=201)
def create_credential(
    data: CredentialCreateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    encrypted = security_service.encrypt_value(data.data)
    cred = pipeline_service.create_credential(name=data.name, credential_type=data.type, encrypted_data=encrypted)
    return CredentialResp.model_validate(cred.__dict__)


@router.put("/{credential_id}", response_model=CredentialResp)
def update_credential(
    credential_id: str,
    data: CredentialUpdateReq,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    security_service: Annotated[SecurityService, Depends(get_security_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    encrypted = security_service.encrypt_value(data.data) if data.data else None
    cred = pipeline_service.update_credential(credential_id, name=data.name, encrypted_data=encrypted)
    return CredentialResp.model_validate(cred.__dict__)


@router.delete("/{credential_id}", status_code=204)
def delete_credential(
    credential_id: str,
    pipeline_service: Annotated[PipelineService, Depends(get_pipeline_service)],
    _current_user=Depends(get_current_user),
) -> None:
    pipeline_service.delete_credential(credential_id)
