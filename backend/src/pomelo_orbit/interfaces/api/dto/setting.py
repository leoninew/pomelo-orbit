"""
系统配置相关 DTO
"""

from pydantic import BaseModel, Field


class ConfigItemResp(BaseModel):
    key: str
    value: object
    default: object
    is_overridden: bool  # .env 中是否存在该 key


class SystemConfigResp(BaseModel):
    items: list[ConfigItemResp]


class SystemConfigUpdateReq(BaseModel):
    key: str = Field(..., description="Dynaconf __ 约定的字段名, 如 traefik__domain_suffix")
    value: str | bool | int | float = Field(..., description="新值")


class SystemConfigResetReq(BaseModel):
    keys: list[str] = Field(..., description="要重置为默认值的字段名列表")
