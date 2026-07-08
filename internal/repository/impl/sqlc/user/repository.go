package userrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	dbsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
)

type Repository struct {
	db      *sqlx.DB
	driver  string
	queries *dbsqlc.Queries
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver, queries: dbsqlc.New(db)}
}

func (r Repository) UserByUsername(ctx context.Context, username string) (model.User, error) {
	user, err := r.queries.UserByUsername(ctx, username)
	if err != nil {
		return model.User{}, fmt.Errorf("load user by username %s: %w", username, err)
	}
	return dbmodel.UserFromByUsername(user), nil
}

func (r Repository) UserById(ctx context.Context, id string) (model.User, error) {
	user, err := r.queries.UserByID(ctx, id)
	if err != nil {
		return model.User{}, fmt.Errorf("load user %s: %w", id, err)
	}
	return dbmodel.UserFromByID(user), nil
}

func (r Repository) UserByEmail(ctx context.Context, email string) (model.User, error) {
	user, err := r.queries.UserByEmail(ctx, sql.NullString{String: email, Valid: true})
	if err != nil {
		return model.User{}, fmt.Errorf("load user by email %s: %w", email, err)
	}
	return dbmodel.UserFromByEmail(user), nil
}

func (r Repository) UserRoles(ctx context.Context, userId string) ([]string, error) {
	roles, err := r.queries.UserRoles(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user roles %s: %w", userId, err)
	}
	return roles, nil
}

func (r Repository) UserPermissions(ctx context.Context, userId string) ([]string, error) {
	permissions, err := r.queries.UserPermissions(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user permissions %s: %w", userId, err)
	}
	return permissions, nil
}

func (r Repository) ListUsers(ctx context.Context, page int, perPage int, search string) (repository.Page[model.User], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := userSearchWhere(search)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM user`+where, args...); err != nil {
		return repository.Page[model.User]{}, fmt.Errorf("count users: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.User
	err := r.db.SelectContext(ctx, &items, `SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
		email, auth_source, created_at, updated_at, last_login_at FROM user`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.User]{}, fmt.Errorf("list users: %w", err)
	}
	return repository.Page[model.User]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) UserRolesByUserIds(ctx context.Context, userIds []string) (map[string][]model.Role, error) {
	rolesByUserId := make(map[string][]model.Role, len(userIds))
	if len(userIds) == 0 {
		return rolesByUserId, nil
	}
	for _, userId := range userIds {
		rolesByUserId[userId] = []model.Role{}
	}
	query, args, err := sqlIn(`SELECT user_role.user_id, role.id, role.code, role.name, role.description, role.created_at, role.updated_at FROM user_role
		JOIN role ON role.id = user_role.role_id
		WHERE user_role.user_id IN (?) ORDER BY user_role.user_id, role.code`, userIds)
	if err != nil {
		return nil, fmt.Errorf("build user roles query: %w", err)
	}
	var rows []struct {
		UserId string `db:"user_id"`
		model.Role
	}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load users roles: %w", err)
	}
	for _, item := range rows {
		rolesByUserId[item.UserId] = append(rolesByUserId[item.UserId], item.Role)
	}
	return rolesByUserId, nil
}

func (r Repository) UserRoleDetails(ctx context.Context, userId string) ([]model.Role, error) {
	rows, err := r.queries.UserRoleDetails(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("load user role details %s: %w", userId, err)
	}
	roles := make([]model.Role, 0, len(rows))
	for _, row := range rows {
		roles = append(roles, dbmodel.RoleFromSQLC(row))
	}
	return roles, nil
}

func (r Repository) CreateUser(ctx context.Context, user model.User) error {
	err := r.queries.CreateUser(ctx, dbsqlc.CreateUserParams{
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
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET username = ?, password_hash = ?, status = ?, email = ?, auth_source = ?, oauth_provider = ?, oauth_provider_id = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		user.Username, user.PasswordHash, user.Status, user.Email, user.AuthSource, user.OAuthProvider, user.OAuthProviderId, user.Id)
	if err != nil {
		return fmt.Errorf("update user %s: %w", user.Id, err)
	}
	return nil
}

func (r Repository) SetUserStatus(ctx context.Context, userId string, status string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), status, userId)
	if err != nil {
		return fmt.Errorf("set user status %s: %w", userId, err)
	}
	return nil
}

func (r Repository) SetUserRoles(ctx context.Context, userId string, roleIds []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin set user roles %s: %w", userId, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_role WHERE user_id = ?`, userId); err != nil {
		return fmt.Errorf("delete user roles %s: %w", userId, err)
	}
	for _, roleId := range roleIds {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO user_role (user_id, role_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(r.driver)), userId, roleId); err != nil {
			return fmt.Errorf("insert user role %s/%s: %w", userId, roleId, err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), userId); err != nil {
		return fmt.Errorf("touch user %s: %w", userId, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set user roles %s: %w", userId, err)
	}
	committed = true
	return nil
}

func (r Repository) DeleteUser(ctx context.Context, userId string) error {
	err := r.queries.DeleteUser(ctx, userId)
	if err != nil {
		return fmt.Errorf("delete user %s: %w", userId, err)
	}
	return nil
}

func (r Repository) MarkUserLoggedIn(ctx context.Context, userId string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET last_login_at = %s, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver), db.NowExpr(r.driver)), userId)
	if err != nil {
		return fmt.Errorf("mark user logged in %s: %w", userId, err)
	}
	return nil
}

func (r Repository) SaveLoginHistory(ctx context.Context, history model.LoginHistory) error {
	err := r.queries.SaveLoginHistory(ctx, dbsqlc.SaveLoginHistoryParams{
		ID:        history.Id,
		UserID:    history.UserId,
		Username:  history.Username,
		IpAddress: dbmodel.NullString(history.IpAddress),
		UserAgent: dbmodel.NullString(history.UserAgent),
		LoginAt:   history.LoginAt,
		Success:   dbmodel.BoolInt(history.Success),
	})
	if err != nil {
		return fmt.Errorf("save login history %s: %w", history.Id, err)
	}
	return nil
}

func (r Repository) ListLoginHistory(ctx context.Context, page int, perPage int, search string) (repository.Page[model.LoginHistory], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := loginHistorySearchWhere(search)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM login_history`+where, args...); err != nil {
		return repository.Page[model.LoginHistory]{}, fmt.Errorf("count login history: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.LoginHistory
	err := r.db.SelectContext(ctx, &items, `SELECT id, user_id, username, ip_address, user_agent, login_at, success FROM login_history`+where+` ORDER BY login_at DESC, id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.LoginHistory]{}, fmt.Errorf("list login history: %w", err)
	}
	return repository.Page[model.LoginHistory]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func userSearchWhere(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}
	like := "%" + strings.ToLower(search) + "%"
	return " WHERE LOWER(username) LIKE ? OR LOWER(COALESCE(email, '')) LIKE ?", []any{like, like}
}

func loginHistorySearchWhere(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}
	like := "%" + strings.ToLower(search) + "%"
	return " WHERE LOWER(username) LIKE ?", []any{like}
}

func sqlIn(query string, values []string) (string, []any, error) {
	args := make([]any, len(values))
	for i, value := range values {
		args[i] = value
	}
	return sqlxIn(query, args...)
}

func sqlxIn(query string, args ...any) (string, []any, error) {
	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return "", nil, err
	}
	return expanded, expandedArgs, nil
}
