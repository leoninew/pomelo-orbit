"""
凭据管理 API
"""

import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query, status

from pomelo_orbit.application.credential_service import CredentialService
from pomelo_orbit.application.di import get_credential_service
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.schemas import (
    CredentialCreateReq,
    CredentialDetailResp,
    CredentialResp,
    CredentialUpdateReq,
    PaginatedResp,
)

router = APIRouter(prefix="/credential", tags=["credential"])


@router.get("", response_model=PaginatedResp[CredentialResp])
def list_credentials(
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[CredentialResp]:
    """列出所有凭据（不返回值）"""
    credentials, total = credential_service.list_credentials(page, per_page, search)
    return PaginatedResp(
        items=[CredentialResp.model_validate(c) for c in credentials],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.get("/{credential_id}", response_model=CredentialDetailResp)
def get_credential(
    credential_id: str,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    _current_user=Depends(get_current_user),
) -> CredentialDetailResp:
    """获取凭据详情"""
    credential = credential_service.get_credential(credential_id)
    return CredentialDetailResp(
        id=credential.id,
        application_id=credential.application_id,
        name=credential.name,
        type=credential.type,
        value=credential_service.decrypt_credential_value(credential),
        extra_data=credential.extra_data,
        created_at=credential.created_at,
    )


@router.post("", response_model=CredentialResp, status_code=status.HTTP_201_CREATED)
def create_credential(
    data: CredentialCreateReq,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    """创建凭据"""
    credential = credential_service.create_credential(
        application_id=data.application_id,
        name=data.name,
        credential_type=data.type,
        value=data.value,
        extra_data=data.extra_data,
    )
    return CredentialResp.model_validate(credential)


@router.put("/{credential_id}", response_model=CredentialResp)
def update_credential(
    credential_id: str,
    data: CredentialUpdateReq,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    _current_user=Depends(get_current_user),
) -> CredentialResp:
    """更新凭据"""
    credential = credential_service.update_credential(
        credential_id=credential_id,
        name=data.name,
        value=data.value,
        extra_data=data.extra_data,
    )
    return CredentialResp.model_validate(credential)


@router.delete("/{credential_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_credential(
    credential_id: str,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    _current_user=Depends(get_current_user),
):
    """删除凭据"""
    credential_service.delete_credential(credential_id)
