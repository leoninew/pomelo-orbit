package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// RoleStore persists roles and their permission assignments.
type RoleStore interface {
	RoleById(ctx context.Context, id string) (model.Role, error)
	RoleByCode(ctx context.Context, code string) (model.Role, error)
	RoleByName(ctx context.Context, name string) (model.Role, error)
	PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error)
	CreateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	UpdateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	DeleteRole(ctx context.Context, roleId string) error
}
