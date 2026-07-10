package usersvc

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo  repository.UserStore
	roles repository.RoleStore
}

func New(repo repository.UserStore, roles repository.RoleStore) Service {
	return Service{repo: repo, roles: roles}
}

func (s Service) List(ctx context.Context, page int, perPage int, search string) (repository.Page[userdto.ListItem], error) {
	users, err := s.repo.ListUsers(ctx, page, perPage, search)
	if err != nil {
		return repository.Page[userdto.ListItem]{}, err
	}
	userIds := make([]string, 0, len(users.Items))
	for _, user := range users.Items {
		userIds = append(userIds, user.Id)
	}
	rolesByUserId, err := s.repo.UserRolesByUserIds(ctx, userIds)
	if err != nil {
		return repository.Page[userdto.ListItem]{}, err
	}
	items := make([]userdto.ListItem, 0, len(users.Items))
	for _, user := range users.Items {
		items = append(items, userdto.ListItem{User: user, Roles: rolesByUserId[user.Id]})
	}
	return repository.Page[userdto.ListItem]{Items: items, Total: users.Total, Page: users.Page, PerPage: users.PerPage}, nil
}

func (s Service) Detail(ctx context.Context, userId string) (userdto.Detail, error) {
	user, err := s.find(ctx, userId)
	if err != nil {
		return userdto.Detail{}, err
	}
	roles, err := s.repo.UserRoleDetails(ctx, user.Id)
	if err != nil {
		return userdto.Detail{}, err
	}
	permissions, err := s.repo.UserPermissions(ctx, user.Id)
	if err != nil {
		return userdto.Detail{}, err
	}
	roleIds := make([]string, 0, len(roles))
	for _, role := range roles {
		roleIds = append(roleIds, role.Id)
	}
	permissionsByRoleId, err := s.roles.RolePermissionCodesByRoleIds(ctx, roleIds)
	if err != nil {
		return userdto.Detail{}, err
	}
	return userdto.Detail{User: user, Roles: emptyRoles(roles), Permissions: emptyStrings(permissions), PermissionsByRoleId: permissionsByRoleId}, nil
}

func (s Service) Create(ctx context.Context, input userdto.CreateInput) (model.User, error) {
	username := strings.TrimSpace(input.Username)
	email := normalizeEmail(input.Email)
	if username == "" || len(username) > 50 || len(input.Password) < 6 || len(input.Password) > 255 || (email != nil && len(*email) > 255) {
		return model.User{}, ErrInvalidUserFields
	}
	if _, err := s.repo.UserByUsername(ctx, username); err == nil {
		return model.User{}, apperror.New(apperror.KindConflict, "Username "+username+" already exists")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return model.User{}, err
	}
	if email != nil {
		if _, err := s.repo.UserByEmail(ctx, *email); err == nil {
			return model.User{}, apperror.New(apperror.KindConflict, "Email "+*email+" already exists")
		} else if !errors.Is(err, sql.ErrNoRows) {
			return model.User{}, err
		}
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, err
	}
	now := time.Now().UTC()
	user := model.User{Id: idutil.NewId(), Username: username, PasswordHash: string(passwordHash), Status: "enabled", OAuthProvider: "", OAuthProviderId: "", Email: email, AuthSource: "password", CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return model.User{}, err
	}
	return user, nil
}

func (s Service) UpdateByActor(ctx context.Context, actor userdto.Actor, userId string, input userdto.UpdateInput) (userdto.Detail, error) {
	if err := requirePermission(actor, "user:write"); err != nil {
		return userdto.Detail{}, err
	}
	user, err := s.find(ctx, strings.TrimSpace(userId))
	if err != nil {
		return userdto.Detail{}, err
	}
	if input.Password != nil && *input.Password != "" {
		if err := s.canResetPassword(ctx, actor, user); err != nil {
			return userdto.Detail{}, err
		}
	}
	if input.Status != nil && actor.UserId == user.Id && strings.TrimSpace(*input.Status) == "disabled" {
		return userdto.Detail{}, ErrCannotDisableCurrentUser
	}
	updated, err := s.update(ctx, user, input)
	if err != nil {
		return userdto.Detail{}, err
	}
	if err := s.repo.UpdateUser(ctx, updated); err != nil {
		return userdto.Detail{}, err
	}
	return s.Detail(ctx, user.Id)
}

func (s Service) SetStatusByActor(ctx context.Context, actor userdto.Actor, userId string, status string) error {
	if err := requirePermission(actor, "user:write"); err != nil {
		return err
	}
	user, err := s.find(ctx, strings.TrimSpace(userId))
	if err != nil {
		return err
	}
	status = strings.TrimSpace(status)
	if status != "enabled" && status != "disabled" {
		return ErrInvalidUserStatus
	}
	if status == "disabled" && actor.UserId == user.Id {
		return ErrCannotDisableCurrentUser
	}
	if user.Status == status {
		return nil
	}
	return s.repo.SetUserStatus(ctx, user.Id, status)
}

func (s Service) SetRoles(ctx context.Context, actor userdto.Actor, userId string, roleIds []string) (userdto.Detail, error) {
	if err := requirePermission(actor, "user:write"); err != nil {
		return userdto.Detail{}, err
	}
	if err := requirePermission(actor, "role:write"); err != nil {
		return userdto.Detail{}, err
	}
	user, err := s.find(ctx, strings.TrimSpace(userId))
	if err != nil {
		return userdto.Detail{}, err
	}
	normalizedRoleIds, err := normalizeRoleIds(roleIds)
	if err != nil {
		return userdto.Detail{}, err
	}
	for _, roleId := range normalizedRoleIds {
		if _, err := s.roles.RoleById(ctx, roleId); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return userdto.Detail{}, apperror.New(apperror.KindNotFound, "Role "+roleId+" not found")
			}
			return userdto.Detail{}, err
		}
	}
	if err := s.repo.SetUserRoles(ctx, user.Id, normalizedRoleIds); err != nil {
		return userdto.Detail{}, err
	}
	return s.Detail(ctx, user.Id)
}

func (s Service) DeleteByActor(ctx context.Context, actor userdto.Actor, userId string) error {
	if err := requirePermission(actor, "user:write"); err != nil {
		return err
	}
	user, err := s.find(ctx, strings.TrimSpace(userId))
	if err != nil {
		return err
	}
	if actor.UserId == user.Id {
		return ErrCannotDeleteCurrentUser
	}
	return s.repo.DeleteUser(ctx, user.Id)
}

func (s Service) update(ctx context.Context, user model.User, input userdto.UpdateInput) (model.User, error) {
	if input.Username != nil {
		username := strings.TrimSpace(*input.Username)
		if username == "" || len(username) > 50 {
			return model.User{}, ErrInvalidUserFields
		}
		if username != user.Username {
			existing, err := s.repo.UserByUsername(ctx, username)
			if err == nil && existing.Id != user.Id {
				return model.User{}, apperror.New(apperror.KindConflict, "Username "+username+" already exists")
			}
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return model.User{}, err
			}
		}
		user.Username = username
	}
	if input.Password != nil && *input.Password != "" {
		password := *input.Password
		if len(password) < 6 || len(password) > 255 {
			return model.User{}, ErrInvalidUserFields
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return model.User{}, err
		}
		user.PasswordHash = string(passwordHash)
	}
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		if status != "enabled" && status != "disabled" {
			return model.User{}, ErrInvalidUserStatus
		}
		user.Status = status
	}
	return user, nil
}

func (s Service) find(ctx context.Context, userId string) (model.User, error) {
	user, err := s.repo.UserById(ctx, userId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, apperror.New(apperror.KindNotFound, "User "+userId+" not found")
		}
		return model.User{}, err
	}
	return user, nil
}

func (s Service) canResetPassword(ctx context.Context, actor userdto.Actor, target model.User) error {
	if actor.UserId == target.Id || hasPermission(actor.Permissions, "role:write") {
		return nil
	}
	permissions, err := s.repo.UserPermissions(ctx, target.Id)
	if err != nil {
		return err
	}
	if hasPermission(permissions, "role:write") {
		return ErrPermissionDenied
	}
	return nil
}

func normalizeRoleIds(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		roleId := strings.TrimSpace(value)
		if roleId == "" {
			return nil, ErrInvalidRoleID
		}
		if _, ok := seen[roleId]; ok {
			return nil, ErrDuplicateRoleIDs
		}
		seen[roleId] = struct{}{}
		result = append(result, roleId)
	}
	return result, nil
}

func requirePermission(actor userdto.Actor, permission string) error {
	if !hasPermission(actor.Permissions, permission) {
		return ErrPermissionDenied
	}
	return nil
}

func hasPermission(permissions []string, permission string) bool {
	return slices.Contains(permissions, permission)
}

func emptyRoles(roles []model.Role) []model.Role {
	if roles == nil {
		return []model.Role{}
	}
	return roles
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func normalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*email)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

var (
	ErrInvalidUserFields        = apperror.New(apperror.KindValidation, "Invalid user fields")
	ErrInvalidUserStatus        = apperror.New(apperror.KindValidation, "Invalid user status")
	ErrCannotDisableCurrentUser = apperror.New(apperror.KindValidation, "Cannot disable current user")
	ErrCannotDeleteCurrentUser  = apperror.New(apperror.KindValidation, "Cannot delete current user")
	ErrInvalidRoleID            = apperror.New(apperror.KindValidation, "Invalid role id")
	ErrDuplicateRoleIDs         = apperror.New(apperror.KindValidation, "role_ids must be unique")
	ErrPermissionDenied         = apperror.New(apperror.KindForbidden, "Permission denied")
)
