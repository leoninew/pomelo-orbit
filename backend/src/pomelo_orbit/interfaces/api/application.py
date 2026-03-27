"""
应用管理 API
"""

import math
from typing import Annotated

from fastapi import APIRouter, BackgroundTasks, Body, Depends, Query, status

from pomelo_orbit.application.application_service import ApplicationService
from pomelo_orbit.application.di import get_application_service
from pomelo_orbit.domain.entities import TriggerType
from pomelo_orbit.domain.value_objects import OperationType
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto import (
    ApplicationCreateReq,
    ApplicationExportResp,
    ApplicationImportReq,
    ApplicationResp,
    ApplicationUpdateReq,
    ConfigFileReq,
    ConfigFileResp,
    PaginatedResp,
)

router = APIRouter(prefix="/application", tags=["application"])


@router.get("", response_model=PaginatedResp[ApplicationResp])
def list_applications(
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[ApplicationResp]:
    """列出所有应用"""
    apps, total = app_service.list_applications(page, per_page, search)
    return PaginatedResp(
        items=[ApplicationResp.model_validate(a) for a in apps],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=ApplicationResp, status_code=status.HTTP_201_CREATED)
def create_application(
    data: ApplicationCreateReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ApplicationResp:
    """创建应用"""
    git_source_data = data.git_source.model_dump() if data.git_source else None
    image_source_data = data.image_source.model_dump() if data.image_source else None

    app = app_service.create_application(
        name=data.name,
        code=data.code,
        enabled=data.enabled,
        image_pull_policy=data.image_pull_policy,
        git_source_data=git_source_data,
        image_source_data=image_source_data,
    )

    return ApplicationResp.model_validate(app)


@router.post("/import", response_model=ApplicationResp, status_code=status.HTTP_201_CREATED)
def import_application(
    data: ApplicationImportReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ApplicationResp:
    """导入应用"""
    app = app_service.import_application(data.model_dump())
    return ApplicationResp.model_validate(app)


@router.get("/{app_id}", response_model=ApplicationResp)
def get_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ApplicationResp:
    """获取应用详情"""
    app = app_service.get_application(app_id)
    return ApplicationResp.model_validate(app)


@router.get("/{app_id}/export")
def export_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ApplicationExportResp:
    """导出应用"""
    data = app_service.export_application(app_id)
    return ApplicationExportResp(**data)


@router.put("/{app_id}", response_model=ApplicationResp)
def update_application(
    app_id: str,
    data: ApplicationUpdateReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ApplicationResp:
    """更新应用基本信息"""
    update_data = data.model_dump(exclude_unset=True, exclude={"git_source", "image_source"})
    git_source_data = data.git_source.model_dump() if data.git_source is not None else None
    image_source_data = data.image_source.model_dump() if data.image_source is not None else None

    app = app_service.update_application(
        application_id=app_id,
        update_data=update_data,
        git_source_data=git_source_data,
        image_source_data=image_source_data,
    )

    return ApplicationResp.model_validate(app)


@router.delete("/{app_id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    remove_dir: Annotated[bool, Body(embed=True)] = False,
):
    """删除应用"""
    await app_service.delete_application(app_id, remove_dir)


@router.get("/{app_id}/files")
def list_application_files(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> list[ConfigFileResp]:
    """获取应用配置文件列表"""
    files = app_service.get_config_files(app_id)
    return [ConfigFileResp.model_validate(f) for f in files]


@router.post("/{app_id}/file")
def create_application_file(
    app_id: str,
    data: ConfigFileReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ConfigFileResp:
    """创建应用配置文件"""
    config_file = app_service.create_config_file(app_id, data.path, data.content)
    return ConfigFileResp.model_validate(config_file)


@router.get("/{app_id}/file/{file_id}")
def read_application_file(
    app_id: str,
    file_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> dict:
    """读取应用配置文件"""
    config_file = app_service.get_config_file(app_id, file_id)
    return {"content": config_file.content, "path": config_file.path}


@router.put("/{app_id}/file/{file_id}")
async def write_application_file(
    app_id: str,
    file_id: str,
    data: ConfigFileReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> ConfigFileResp:
    """写入应用配置文件"""
    config_file = await app_service.update_config_file(app_id, file_id, data.content, path=data.path)
    return ConfigFileResp.model_validate(config_file)


@router.delete("/{app_id}/file/{file_id}", status_code=status.HTTP_204_NO_CONTENT)
def delete_application_file(
    app_id: str,
    file_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
):
    """删除应用配置文件"""
    app_service.delete_config_file(app_id, file_id)


@router.post("/{app_id}/deploy")
async def deploy_application(
    app_id: str,
    background_tasks: BackgroundTasks,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    branch: Annotated[str | None, Body(embed=True)] = None,
    env: Annotated[str | None, Body(embed=True)] = None,
) -> dict:
    """手动触发部署"""
    app = app_service.get_application(app_id)

    deployment = app_service.create_deployment(
        application_id=app.id,
        operation_type=OperationType.DEPLOY,
        trigger_type=TriggerType.MANUAL,
        trigger_ref=branch,
        env_file=env,
        is_rollback=False,
    )

    background_tasks.add_task(app_service.deploy, app, deployment)

    return {"deployment_id": deployment.id}


@router.post("/{app_id}/stop")
async def stop_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    remove_volumes: Annotated[bool, Body(embed=True)] = False,
    env: Annotated[str | None, Body(embed=True)] = None,
) -> dict:
    """停止应用"""
    deployment = await app_service.stop_application(app_id, remove_volumes, env)
    return {"deployment_id": deployment.id}


@router.post("/{app_id}/restart")
async def restart_application(
    app_id: str,
    background_tasks: BackgroundTasks,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    env: Annotated[str | None, Body(embed=True)] = None,
) -> dict:
    """重启应用"""
    app = app_service.get_application(app_id)

    deployment = await app_service.restart_application(app_id, env)

    background_tasks.add_task(app_service.execute_restart, app, deployment)

    return {"deployment_id": deployment.id}


@router.get("/{app_id}/status")
async def get_application_status(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
) -> dict:
    """获取应用运行状态"""
    app = app_service.get_application(app_id)
    status_info = await app_service.get_application_status(app.code)
    return {"status": status_info}


@router.get("/{app_id}/logs")
async def get_application_logs(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    _current_user=Depends(get_current_user),
    tail: Annotated[int, Query(ge=1, le=1000)] = 100,
) -> dict:
    """获取应用日志"""
    app = app_service.get_application(app_id)
    logs = await app_service.get_application_logs(app.code, tail)
    return {"logs": logs}
