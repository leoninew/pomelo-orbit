# 权限系统指引

## 当前范围

Pomelo Orbit 当前采用基于角色的权限控制（RBAC）：

- 用户通过角色获得权限。
- 角色通过权限码获得能力。
- 后端接口负责最终鉴权，前端只做导航和按钮显隐。
- 当前已接入权限控制的业务范围是用户管理和角色管理。

CI、CD、项目、登录历史、系统设置等页面目前只要求登录，不做细粒度权限控制。

## 数据模型

权限相关表：

| 表 | 作用 |
|---|---|
| `role` | 角色定义，包含 `code`、`name`、`description` |
| `permission` | 权限定义，包含稳定的权限码 `code` |
| `user_role` | 用户与角色的多对多关系 |
| `role_permission` | 角色与权限的多对多关系 |

用户账号状态使用 `user.status`，当前只允许：

- `enabled`
- `disabled`

不要在用户或角色链路重新引入 `is_active`。`project.is_active` 是项目归档状态，属于独立概念。

## 现有权限码

| 权限码 | 含义 | 主要用途 |
|---|---|---|
| `user:read` | 查看用户 | 用户列表、用户详情 |
| `user:write` | 管理用户 | 创建、编辑、启用、禁用、删除用户 |
| `role:read` | 查看角色 | 角色列表、角色详情、读取权限列表 |
| `role:write` | 管理角色 | 创建、编辑、删除角色、分配权限 |

初始化数据会创建 `admin` 角色，并授予上述全部权限。

## 后端鉴权

后端使用 `require_permission()` 保护接口：

```python
@router.get("", response_model=PaginatedResp[UserListResp])
def list_users(
    user_service: Annotated[UserService, Depends(get_user_service)],
    _current_user: Annotated[User, Depends(require_permission("user:read"))],
):
    ...
```

规则：

1. 需要登录但不需要权限码的接口使用 `get_current_user`。
2. 需要权限码的接口使用 `require_permission("resource:action")`。
3. 接口层不捕获鉴权异常，交给全局异常处理。
4. 前端显隐不能替代后端鉴权。

当前用户管理还有额外约束：

- 不能禁用或删除当前登录用户。
- 通过更新用户接口把当前用户 `status` 改为 `disabled` 也会被拒绝。
- 只有具备 `role:write` 的用户才能给用户分配角色。
- 仅具备 `user:write` 的用户不能重置拥有 `role:write` 权限用户的密码。

## 前端权限使用

登录后，`/api/auth/me` 返回当前用户的 `roles` 和 `permissions`。前端通过 `authStore.hasPermission()` 判断权限。

路由级控制：

```ts
{
  path: '/users',
  name: 'Users',
  component: () => import('@/views/UserPage.vue'),
  meta: { title: '用户管理', menuKey: 'users', permission: 'user:read' },
}
```

导航显隐：

```ts
{
  key: 'users',
  label: '用户管理',
  path: '/users',
  permission: 'user:read',
}
```

页面操作显隐：

```ts
const canWriteUsers = computed(() => authStore.hasPermission('user:write'));
```

使用原则：

1. 列表和详情入口使用 `*:read`。
2. 创建、编辑、删除、启用、禁用等操作使用 `*:write`。
3. 角色分配同时要求 `role:read` 和 `role:write`，因为页面需要读取角色列表并提交角色关系。
4. 前端没有权限时隐藏入口或按钮；后端仍必须拒绝未授权请求。

## 新增权限的流程

1. 在迁移数据中新增 `permission` 记录，权限码使用 `resource:action` 格式。
2. 给需要默认拥有该权限的角色补充 `role_permission` 记录。
3. 后端接口添加 `require_permission("resource:action")`。
4. 前端路由、导航和操作按钮使用同一个权限码。
5. 补充接口测试，覆盖有权限和无权限两种路径。

权限码是持久化数据，不要随意改名。确实要改名时，需要新增迁移同步更新 `permission.code`、`role_permission` 关系和前后端引用。
