package userrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	usersqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/user"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.UserStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *usersqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *usersqlc.Queries {
		return usersqlc.New(dbtx)
	})
}

func (r Repository) UserByUsername(ctx context.Context, username string) (model.User, error) {
	user, err := r.q(ctx).UserByUsername(ctx, username)
	if err != nil {
		return model.User{}, fmt.Errorf("load user by username %s: %w", username, sqlcommon.TranslateError(err))
	}
	return userFromRow(user.ID, user.Username, user.PasswordHash, user.Status, user.OauthProvider, user.OauthProviderID, user.Email, user.AuthSource, user.CreatedAt, user.UpdatedAt, user.LastLoginAt), nil
}

func (r Repository) UserById(ctx context.Context, id string) (model.User, error) {
	user, err := r.q(ctx).UserByID(ctx, id)
	if err != nil {
		return model.User{}, fmt.Errorf("load user %s: %w", id, sqlcommon.TranslateError(err))
	}
	return userFromRow(user.ID, user.Username, user.PasswordHash, user.Status, user.OauthProvider, user.OauthProviderID, user.Email, user.AuthSource, user.CreatedAt, user.UpdatedAt, user.LastLoginAt), nil
}

func (r Repository) UserByEmail(ctx context.Context, email string) (model.User, error) {
	user, err := r.q(ctx).UserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return model.User{}, fmt.Errorf("load user by email %s: %w", email, sqlcommon.TranslateError(err))
	}
	return userFromRow(user.ID, user.Username, user.PasswordHash, user.Status, user.OauthProvider, user.OauthProviderID, user.Email, user.AuthSource, user.CreatedAt, user.UpdatedAt, user.LastLoginAt), nil
}

func (r Repository) UserRoles(ctx context.Context, userId string) ([]string, error) {
	roles, err := r.q(ctx).UserRoles(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user roles %s: %w", userId, err)
	}
	return roles, nil
}

func (r Repository) UserPermissions(ctx context.Context, userId string) ([]string, error) {
	permissions, err := r.q(ctx).UserPermissions(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user permissions %s: %w", userId, err)
	}
	return permissions, nil
}

func (r Repository) ListUsers(ctx context.Context, page int, perPage int, search string) (repository.Page[model.User], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.LowerSearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	total, err := q.CountUsers(ctx, usersqlc.CountUsersParams{SearchPattern: searchPattern})
	if err != nil {
		return repository.Page[model.User]{}, fmt.Errorf("count users: %w", err)
	}
	rows, err := q.ListUsers(ctx, usersqlc.ListUsersParams{
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.User]{}, fmt.Errorf("list users: %w", err)
	}
	items := make([]model.User, 0, len(rows))
	for _, row := range rows {
		items = append(items, userFromRow(row.ID, row.Username, row.PasswordHash, row.Status, row.OauthProvider, row.OauthProviderID, row.Email, row.AuthSource, row.CreatedAt, row.UpdatedAt, row.LastLoginAt))
	}
	return repository.Page[model.User]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) UserRolesByUserIds(ctx context.Context, userIds []string) (map[string][]model.Role, error) {
	rolesByUserId := make(map[string][]model.Role, len(userIds))
	for _, userId := range userIds {
		rolesByUserId[userId] = []model.Role{}
	}
	if len(userIds) == 0 {
		return rolesByUserId, nil
	}
	rows, err := r.q(ctx).UserRolesByUserIds(ctx, userIds)
	if err != nil {
		return nil, fmt.Errorf("load users roles: %w", err)
	}
	for _, row := range rows {
		rolesByUserId[row.UserID] = append(rolesByUserId[row.UserID], model.Role{
			Id:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			Description: dbmodel.StringPtr(row.Description),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return rolesByUserId, nil
}

func (r Repository) UserRoleDetails(ctx context.Context, userId string) ([]model.Role, error) {
	rows, err := r.q(ctx).UserRoleDetails(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user role details %s: %w", userId, err)
	}
	roles := make([]model.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, model.Role{
			Id:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			Description: dbmodel.StringPtr(row.Description),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return roles, nil
}

func (r Repository) CreateUser(ctx context.Context, user model.User) error {
	err := r.q(ctx).CreateUser(ctx, usersqlc.CreateUserParams{
		ID:              user.Id,
		Username:        user.Username,
		PasswordHash:    user.PasswordHash,
		Status:          user.Status,
		OauthProvider:   user.OAuthProvider,
		OauthProviderID: user.OAuthProviderId,
		Email:           dbmodel.NullString(user.Email),
		AuthSource:      user.AuthSource,
		CreatedAt:       user.CreatedAt,
		UpdatedAt:       user.UpdatedAt,
		LastLoginAt:     dbmodel.NullTime(user.LastLoginAt),
	})
	if err != nil {
		return fmt.Errorf("create user %s: %w", user.Username, err)
	}
	return nil
}

func (r Repository) UpdateUser(ctx context.Context, user model.User) error {
	err := r.q(ctx).UpdateUser(ctx, usersqlc.UpdateUserParams{
		Username:        user.Username,
		PasswordHash:    user.PasswordHash,
		Status:          user.Status,
		Email:           dbmodel.NullString(user.Email),
		AuthSource:      user.AuthSource,
		OauthProvider:   user.OAuthProvider,
		OauthProviderID: user.OAuthProviderId,
		UpdatedAt:       time.Now().UTC(),
		ID:              user.Id,
	})
	if err != nil {
		return fmt.Errorf("update user %s: %w", user.Id, err)
	}
	return nil
}

func (r Repository) SetUserStatus(ctx context.Context, userId string, status string) error {
	err := r.q(ctx).SetUserStatus(ctx, usersqlc.SetUserStatusParams{
		Status:    status,
		UpdatedAt: time.Now().UTC(),
		ID:        userId,
	})
	if err != nil {
		return fmt.Errorf("set user status %s: %w", userId, err)
	}
	return nil
}

func (r Repository) SetUserRoles(ctx context.Context, userId string, roleIds []string) error {
	q := r.q(ctx)
	if err := q.DeleteUserRoles(ctx, userId); err != nil {
		return fmt.Errorf("delete user roles %s: %w", userId, err)
	}
	now := time.Now().UTC()
	for _, roleId := range roleIds {
		if err := q.InsertUserRole(ctx, usersqlc.InsertUserRoleParams{
			UserID:    userId,
			RoleID:    roleId,
			CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("insert user role %s/%s: %w", userId, roleId, err)
		}
	}
	if err := q.TouchUserUpdatedAt(ctx, usersqlc.TouchUserUpdatedAtParams{
		UpdatedAt: now,
		ID:        userId,
	}); err != nil {
		return fmt.Errorf("touch user %s: %w", userId, err)
	}
	return nil
}

func (r Repository) DeleteUser(ctx context.Context, userId string) error {
	if err := r.q(ctx).DeleteUser(ctx, userId); err != nil {
		return fmt.Errorf("delete user %s: %w", userId, err)
	}
	return nil
}

func (r Repository) MarkUserLoggedIn(ctx context.Context, userId string) error {
	now := time.Now().UTC()
	err := r.q(ctx).MarkUserLoggedIn(ctx, usersqlc.MarkUserLoggedInParams{
		LastLoginAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:   now,
		ID:          userId,
	})
	if err != nil {
		return fmt.Errorf("mark user logged in %s: %w", userId, err)
	}
	return nil
}

func userFromRow(
	id, username, passwordHash, status, oauthProvider, oauthProviderId string,
	email sql.NullString,
	authSource string,
	createdAt, updatedAt time.Time,
	lastLoginAt sql.NullTime,
) model.User {
	return model.User{
		Id:              id,
		Username:        username,
		PasswordHash:    passwordHash,
		Status:          status,
		OAuthProvider:   oauthProvider,
		OAuthProviderId: oauthProviderId,
		Email:           dbmodel.StringPtr(email),
		AuthSource:      authSource,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
		LastLoginAt:     dbmodel.TimePtr(lastLoginAt),
	}
}
