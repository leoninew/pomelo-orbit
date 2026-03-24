"""
通用映射器基类

利用 dataclass 特性自动转换，子类只需指定字段映射
"""

from dataclasses import fields
from typing import ClassVar


class BaseMapper[TDomain, TORM]:
    """通用映射器基类"""

    domain_class: type
    orm_class: type

    # 字段名映射：{domain_field: orm_field}，相同可省略
    field_mapping: ClassVar[dict[str, str]] = {}

    @classmethod
    def to_domain(cls, orm: TORM) -> TDomain:
        """ORM 转领域实体"""
        data = {}
        for f in fields(cls.domain_class):
            orm_field = cls.field_mapping.get(f.name, f.name)
            value = getattr(orm, orm_field, None)
            data[f.name] = value

        return cls.domain_class(**data)  # type: ignore[no-any-return]

    @classmethod
    def to_orm(cls, entity: TDomain) -> TORM:
        """领域实体转 ORM"""
        data = {}
        reverse_mapping = {v: k for k, v in cls.field_mapping.items()}

        for f in fields(cls.domain_class):
            value = getattr(entity, f.name)
            orm_field = reverse_mapping.get(f.name, f.name)
            data[orm_field] = value

        return cls.orm_class(**data)  # type: ignore[no-any-return]
