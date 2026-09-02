package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// UserStore persists user accounts and role assignments.
type UserStore interface {
	UserById(ctx context.Context, id string) (model.User, error)
	UserByUsername(ctx context.Context, username string) (model.User, error)
	UserByEmail(ctx context.Context, email string) (model.User, error)
	CreateUser(ctx context.Context, user model.User) error
	UpdateUser(ctx context.Context, user model.User) error
	SetUserStatus(ctx context.Context, userId string, status string) error
	DeleteUser(ctx context.Context, userId string) error
	DeleteUserRoles(ctx context.Context, userId string) error
	MarkUserLoggedIn(ctx context.Context, id string) error
	UserRoles(ctx context.Context, userId string) ([]string, error)
	UserPermissions(ctx context.Context, userId string) ([]string, error)
	ListUsers(ctx context.Context, page int, perPage int, search string) (Page[model.User], error)
	UserRolesByUserIds(ctx context.Context, userIds []string) (map[string][]model.Role, error)
	UserRoleDetails(ctx context.Context, userId string) ([]model.Role, error)
	SetUserRoles(ctx context.Context, userId string, roleIds []string) error
}
