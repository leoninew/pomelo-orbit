"""凭据管理 API"""

import logging
import math
from typing import Annotated

from fastapi import APIRouter, Depends, Query

from pomelo_orbit.application.ci.credential_service import CredentialService
from pomelo_orbit.application.ci.di import get_credential_service
from pomelo_orbit.domain.project.entities import Project
from pomelo_orbit.interfaces.api.ci.dto.credential import (
    CredentialCreateReq,
    CredentialExportResp,
    CredentialImportReq,
    CredentialResp,
    CredentialUpdateReq,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.project.dependencies import get_current_project

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/credential", tags=["credential"])


@router.get("", response_model=PaginatedResp[CredentialResp])
def list_credentials(
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    current_project: Annotated[Project, Depends(get_current_project)],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 20,
) -> PaginatedResp[CredentialResp]:
    creds, total = credential_service.list_credentials(current_project.id, page=page, per_page=per_page)
    return PaginatedResp(
        items=[CredentialResp.model_validate(c) for c in creds],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=CredentialResp, status_code=201)
def create_credential(
    data: CredentialCreateReq,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    current_project: Annotated[Project, Depends(get_current_project)],
) -> CredentialResp:
    cred = credential_service.create_credential(
        project_id=current_project.id,
        name=data.name,
        credential_type=data.type,
        data=data.data,
    )
    return CredentialResp.model_validate(cred)


@router.get("/{credential_id}", response_model=CredentialResp)
def get_credential(
    credential_id: str,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
) -> CredentialResp:
    cred = credential_service.get_credential(credential_id)
    return CredentialResp.model_validate(cred)


@router.put("/{credential_id}", response_model=CredentialResp)
def update_credential(
    credential_id: str,
    data: CredentialUpdateReq,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
) -> CredentialResp:
    cred = credential_service.update_credential(credential_id, name=data.name, data=data.data)
    return CredentialResp.model_validate(cred)


@router.delete("/{credential_id}", status_code=204)
def delete_credential(
    credential_id: str,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
) -> None:
    credential_service.delete_credential(credential_id)


@router.get("/{credential_id}/export", response_model=CredentialExportResp)
def export_credential(
    credential_id: str,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
) -> CredentialExportResp:
    data = credential_service.export_credential(credential_id)
    return CredentialExportResp(**data)


@router.post("/import", response_model=CredentialResp, status_code=201)
def import_credential(
    data: CredentialImportReq,
    credential_service: Annotated[CredentialService, Depends(get_credential_service)],
    current_project: Annotated[Project, Depends(get_current_project)],
) -> CredentialResp:
    cred = credential_service.import_credential(
        project_id=current_project.id,
        name=data.name,
        credential_type=data.type,
        data=data.data,
    )
    return CredentialResp.model_validate(cred)
