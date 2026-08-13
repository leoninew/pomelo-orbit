package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RoleStore persists roles and their permission assignments.
type RoleStore interface {
	RoleById(ctx context.Context, id string) (model.Role, error)
	RoleByCode(ctx context.Context, code string) (model.Role, error)
	RoleByName(ctx context.Context, name string) (model.Role, error)
	ListRoles(ctx context.Context, page int, perPage int, search string) (Page[model.Role], error)
	ListPermissions(ctx context.Context) ([]model.Permission, error)
	PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error)
	RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error)
	CreateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	UpdateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	DeleteRole(ctx context.Context, roleId string) error
}
