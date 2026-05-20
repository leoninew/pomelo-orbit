"""
应用管理 API
"""

import math
from typing import Annotated

from dishka import AsyncContainer
from dishka.integrations.fastapi import FromDishka, inject
from fastapi import APIRouter, BackgroundTasks, Body, Depends, Query

from pomelo_orbit.application.cd.application_service import ApplicationService
from pomelo_orbit.application.cd.di import get_application_service
from pomelo_orbit.application.project.di import get_project_service
from pomelo_orbit.application.project.project_service import ProjectService
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.entities import Application, TriggerType
from pomelo_orbit.domain.cd.value_objects import OperationType
from pomelo_orbit.interfaces.api.auth.dependencies import get_current_user
from pomelo_orbit.interfaces.api.cd.dto.application import (
    ApplicationCreateReq,
    ApplicationExportResp,
    ApplicationImportReq,
    ApplicationResp,
    ApplicationRouteReq,
    ApplicationRouteResp,
    ApplicationServiceConfigResp,
    ApplicationServiceConfigUpdateReq,
    ApplicationUpdateReq,
    ComposePreviewResp,
    ComposeServiceResp,
    ConfigFileReq,
    ConfigFileResp,
)
from pomelo_orbit.interfaces.api.common import PaginatedResp
from pomelo_orbit.interfaces.api.utils import run_in_new_scope

router = APIRouter(prefix="/application", tags=["application"])


def _authorize_application_project(
    app_service: ApplicationService,
    project_service: ProjectService,
    current_user: User,
    app_id: str,
) -> Application:
    app = app_service.get_application(app_id)
    project_service.get_project(current_user.id, app.project_id)
    return app


@router.get("", response_model=PaginatedResp[ApplicationResp])
def list_applications(
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
    page: Annotated[int, Query(ge=1)] = 1,
    per_page: Annotated[int, Query(ge=1, le=100)] = 10,
    search: Annotated[str | None, Query()] = None,
) -> PaginatedResp[ApplicationResp]:
    """列出所有应用"""
    project_service.get_project(current_user.id, project_id)
    apps, total = app_service.list_applications(project_id, page, per_page, search)
    return PaginatedResp(
        items=[ApplicationResp.model_validate(a) for a in apps],
        total=total,
        page=page,
        per_page=per_page,
        pages=math.ceil(total / per_page) if total > 0 else 1,
    )


@router.post("", response_model=ApplicationResp, status_code=201)
def create_application(
    data: ApplicationCreateReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
) -> ApplicationResp:
    """创建应用"""
    project_service.get_project(current_user.id, project_id)
    app = app_service.create_application(
        project_id=project_id,
        name=data.name,
        code=data.code,
        image_pull_policy=data.image_pull_policy,
    )

    return ApplicationResp.model_validate(app)


@router.post("/import", response_model=ApplicationResp, status_code=201)
def import_application(
    data: ApplicationImportReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    project_id: Annotated[str, Query()],
) -> ApplicationResp:
    """导入应用"""
    project_service.get_project(current_user.id, project_id)
    app = app_service.import_application(project_id, data.model_dump())
    return ApplicationResp.model_validate(app)


@router.get("/{app_id}", response_model=ApplicationResp)
def get_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationResp:
    """获取应用详情"""
    app = _authorize_application_project(app_service, project_service, current_user, app_id)
    return ApplicationResp.model_validate(app)


@router.get("/{app_id}/export")
def export_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationExportResp:
    """导出应用"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    data = app_service.export_application(app_id)
    return ApplicationExportResp(**data)


@router.put("/{app_id}", response_model=ApplicationResp)
def update_application(
    app_id: str,
    data: ApplicationUpdateReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationResp:
    """更新应用"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    update_data = data.model_dump(exclude_unset=True)

    app = app_service.update_application(
        application_id=app_id,
        update_data=update_data,
    )

    return ApplicationResp.model_validate(app)


@router.delete("/{app_id}", status_code=204)
async def delete_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    remove_dir: Annotated[bool, Query()] = False,
):
    """删除应用"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    await app_service.delete_application(app_id, remove_dir)


@router.post("/{app_id}/compose-preview", response_model=ComposePreviewResp)
def preview_compose_yaml(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ComposePreviewResp:
    """预览部署时生成的 docker-compose.yml"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    yaml_text = app_service.preview_compose_yaml(app_id)
    return ComposePreviewResp(compose_yaml=yaml_text)


@router.get("/{app_id}/files")
def list_application_files(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ConfigFileResp]:
    """获取应用配置文件列表"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    files = app_service.get_config_files(app_id)
    return [ConfigFileResp.model_validate(f) for f in files]


@router.post("/{app_id}/file")
def create_application_file(
    app_id: str,
    data: ConfigFileReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ConfigFileResp:
    """创建应用配置文件"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    config_file = app_service.create_config_file(app_id, data.path, data.content)
    return ConfigFileResp.model_validate(config_file)


@router.get("/{app_id}/file/{file_id}")
def read_application_file(
    app_id: str,
    file_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> dict:
    """读取应用配置文件"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    config_file = app_service.get_config_file(app_id, file_id)
    return {"content": config_file.content, "path": config_file.path}


@router.put("/{app_id}/file/{file_id}")
async def write_application_file(
    app_id: str,
    file_id: str,
    data: ConfigFileReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ConfigFileResp:
    """写入应用配置文件"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    config_file = await app_service.update_config_file(app_id, file_id, data.content, path=data.path)
    return ConfigFileResp.model_validate(config_file)


@router.delete("/{app_id}/file/{file_id}", status_code=204)
def delete_application_file(
    app_id: str,
    file_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
):
    """删除应用配置文件"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    app_service.delete_config_file(app_id, file_id)


@router.post("/{app_id}/deploy")
@inject
async def deploy_application(
    app_id: str,
    background_tasks: BackgroundTasks,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    container: FromDishka[AsyncContainer],
) -> dict:
    """手动触发部署"""
    app = _authorize_application_project(app_service, project_service, current_user, app_id)

    # 验证 docker-compose 文件存在（在创建部署记录之前）
    app_service.get_compose_file(app_id)
    deployment = app_service.create_deployment(
        project_id=app.project_id,
        application_id=app.id,
        operation_type=OperationType.DEPLOY,
        trigger_type=TriggerType.MANUAL,
        is_rollback=False,
    )

    async def _run_deploy() -> None:
        await run_in_new_scope(container, ApplicationService, lambda svc: svc.deploy(app, deployment))

    background_tasks.add_task(_run_deploy)

    return {"deployment_id": deployment.id}


@router.post("/{app_id}/stop")
async def stop_application(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    remove_volumes: Annotated[bool, Body(embed=True)] = False,
) -> dict:
    """停止应用"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    deployment = await app_service.stop_application(app_id, remove_volumes)
    return {"deployment_id": deployment.id}


@router.post("/{app_id}/restart")
@inject
async def restart_application(
    app_id: str,
    background_tasks: BackgroundTasks,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    container: FromDishka[AsyncContainer],
) -> dict:
    """重启应用"""
    app = _authorize_application_project(app_service, project_service, current_user, app_id)

    # 验证 docker-compose 文件存在（在创建部署记录之前）
    app_service.get_compose_file(app_id)

    deployment = await app_service.restart_application(app_id)

    async def _run_restart() -> None:
        await run_in_new_scope(container, ApplicationService, lambda svc: svc.execute_restart(app, deployment))

    background_tasks.add_task(_run_restart)

    return {"deployment_id": deployment.id}


@router.get("/{app_id}/status")
async def get_application_status(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> dict:
    """获取应用运行状态"""
    app = _authorize_application_project(app_service, project_service, current_user, app_id)
    status_info = await app_service.get_application_status(app.code)
    return {"status": status_info}


@router.get("/{app_id}/logs")
async def get_application_logs(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
    tail: Annotated[int, Query(ge=1, le=1000)] = 100,
) -> dict:
    """获取应用日志"""
    app = _authorize_application_project(app_service, project_service, current_user, app_id)
    logs = await app_service.get_application_logs(app.code, tail)
    return {"logs": logs}


@router.get("/{app_id}/route")
def list_app_routes(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ApplicationRouteResp]:
    """获取应用路由配置列表"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    routes = app_service.list_app_routes(app_id)
    return [ApplicationRouteResp.model_validate(r) for r in routes]


@router.post("/{app_id}/route", status_code=201)
def create_app_route(
    app_id: str,
    data: ApplicationRouteReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationRouteResp:
    """创建应用路由配置"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    route = app_service.create_app_route(app_id, data.service_name, data.domain, data.port)
    return ApplicationRouteResp.model_validate(route)


@router.put("/{app_id}/route/{route_id}")
def update_app_route(
    app_id: str,
    route_id: str,
    data: ApplicationRouteReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationRouteResp:
    """更新应用路由配置"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    route = app_service.update_app_route(app_id, route_id, data.service_name, data.domain, data.port)
    return ApplicationRouteResp.model_validate(route)


@router.delete("/{app_id}/route/{route_id}", status_code=204)
def delete_app_route(
    app_id: str,
    route_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> None:
    """删除应用路由配置"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    app_service.delete_app_route(app_id, route_id)


@router.get("/{app_id}/compose-service")
def list_compose_services(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ComposeServiceResp]:
    """解析 docker-compose 模板，返回 service 信息列表"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    return [ComposeServiceResp(**s) for s in app_service.parse_compose_services(app_id)]


@router.get("/{app_id}/service-config")
def list_service_configs(
    app_id: str,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> list[ApplicationServiceConfigResp]:
    """获取应用 service 级配置列表"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    return [ApplicationServiceConfigResp(**s) for s in app_service.list_service_configs(app_id)]


@router.put("/{app_id}/service-config/{service_name}")
def update_service_config(
    app_id: str,
    service_name: str,
    data: ApplicationServiceConfigUpdateReq,
    app_service: Annotated[ApplicationService, Depends(get_application_service)],
    project_service: Annotated[ProjectService, Depends(get_project_service)],
    current_user: Annotated[User, Depends(get_current_user)],
) -> ApplicationServiceConfigResp:
    """更新应用某个 service 的配置覆盖"""
    _authorize_application_project(app_service, project_service, current_user, app_id)
    service_config = app_service.update_service_config(app_id, service_name, data.image)
    return ApplicationServiceConfigResp(**service_config)
