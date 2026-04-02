"""
系统配置 API 路由
"""

from typing import Annotated

from fastapi import APIRouter, Depends

from pomelo_orbit.application.cd.di import get_setting_service
from pomelo_orbit.application.setting_service import SettingService
from pomelo_orbit.domain.shared.entities import User
from pomelo_orbit.interfaces.api.auth import get_current_user
from pomelo_orbit.interfaces.api.dto import SystemConfigResetReq, SystemConfigResp, SystemConfigUpdateReq

router = APIRouter(prefix="/settings", tags=["settings"])


@router.get("/config", response_model=SystemConfigResp)
def get_config(
    setting_service: Annotated[SettingService, Depends(get_setting_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> SystemConfigResp:
    """获取运行时配置列表（需要登录）"""
    return SystemConfigResp.model_validate(setting_service.get_config())


@router.put("/config", response_model=SystemConfigResp)
def update_config(
    req: SystemConfigUpdateReq,
    setting_service: Annotated[SettingService, Depends(get_setting_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> SystemConfigResp:
    """新增或更新单个配置项，写入 .env（需要重启服务生效）"""
    result = setting_service.update_config(req.key, req.value)
    return SystemConfigResp.model_validate(result)


@router.delete("/config", response_model=SystemConfigResp)
def reset_config(
    req: SystemConfigResetReq,
    setting_service: Annotated[SettingService, Depends(get_setting_service)],
    _current_user: Annotated[User, Depends(get_current_user)],
) -> SystemConfigResp:
    """重置指定配置项为默认值（从 .env 删除对应行）"""
    result = setting_service.reset_config(req.keys)
    return SystemConfigResp.model_validate(result)
