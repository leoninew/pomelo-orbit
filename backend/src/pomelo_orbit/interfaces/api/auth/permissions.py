from collections.abc import Callable
from typing import Annotated

from fastapi import Depends

from pomelo_orbit.domain import BusinessError
from pomelo_orbit.domain.auth.entities import User
from pomelo_orbit.domain.cd.repositories import UserRepository
from pomelo_orbit.infrastructure.cd.repositories.di import get_user_repo
from pomelo_orbit.interfaces.api.auth.router import get_current_user


def require_permission(permission_code: str) -> Callable[..., User]:
    def dependency(
        current_user: Annotated[User, Depends(get_current_user)],
        user_repo: Annotated[UserRepository, Depends(get_user_repo)],
    ) -> User:
        permission_codes = {permission.code for permission in user_repo.find_permissions(current_user.id)}
        if permission_code not in permission_codes:
            raise BusinessError("Permission denied", status_code=403)
        return current_user

    return dependency
