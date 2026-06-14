package orbit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"backend/internal/db"
)

func (s Store) UserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, `SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
		email, auth_source, created_at, updated_at, last_login_at FROM user WHERE email = ?`, email)
	if err != nil {
		return User{}, fmt.Errorf("load user by email %s: %w", email, err)
	}
	return user, nil
}

func (s Store) ListUsers(ctx context.Context, page int, perPage int, search string) (Page[User], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := userSearchWhere(search)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM user`+where, args...); err != nil {
		return Page[User]{}, fmt.Errorf("count users: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []User
	err := s.db.SelectContext(ctx, &items, `SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
		email, auth_source, created_at, updated_at, last_login_at FROM user`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[User]{}, fmt.Errorf("list users: %w", err)
	}
	return Page[User]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) UserRolesByUserIds(ctx context.Context, userIds []string) (map[string][]Role, error) {
	rolesByUserId := make(map[string][]Role, len(userIds))
	if len(userIds) == 0 {
		return rolesByUserId, nil
	}
	for _, userId := range userIds {
		rolesByUserId[userId] = []Role{}
	}
	query, args, err := sqlIn(`SELECT user_role.user_id, role.id, role.code, role.name, role.description, role.created_at, role.updated_at FROM user_role
		JOIN role ON role.id = user_role.role_id
		WHERE user_role.user_id IN (?) ORDER BY user_role.user_id, role.code`, userIds)
	if err != nil {
		return nil, fmt.Errorf("build user roles query: %w", err)
	}
	var rows []struct {
		UserId string `db:"user_id"`
		Role
	}
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load users roles: %w", err)
	}
	for _, item := range rows {
		rolesByUserId[item.UserId] = append(rolesByUserId[item.UserId], item.Role)
	}
	return rolesByUserId, nil
}

func (s Store) UserRoleDetails(ctx context.Context, userId string) ([]Role, error) {
	var roles []Role
	err := s.db.SelectContext(ctx, &roles, `SELECT role.id, role.code, role.name, role.description, role.created_at, role.updated_at FROM role
		JOIN user_role ON user_role.role_id = role.id
		WHERE user_role.user_id = ? ORDER BY role.code`, userId)
	if err != nil {
		return nil, fmt.Errorf("load user role details %s: %w", userId, err)
	}
	return roles, nil
}

func (s Store) RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error) {
	permissionsByRoleId := make(map[string][]string, len(roleIds))
	if len(roleIds) == 0 {
		return permissionsByRoleId, nil
	}
	for _, roleId := range roleIds {
		permissionsByRoleId[roleId] = []string{}
	}
	query, args, err := sqlIn(`SELECT role_permission.role_id, permission.code FROM role_permission
		JOIN permission ON permission.id = role_permission.permission_id
		WHERE role_permission.role_id IN (?) ORDER BY role_permission.role_id, permission.code`, roleIds)
	if err != nil {
		return nil, fmt.Errorf("build role permissions query: %w", err)
	}
	var rows []struct {
		RoleId string `db:"role_id"`
		Code   string `db:"code"`
	}
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load role permissions: %w", err)
	}
	for _, row := range rows {
		permissionsByRoleId[row.RoleId] = append(permissionsByRoleId[row.RoleId], row.Code)
	}
	return permissionsByRoleId, nil
}

func (s Store) CreateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO user (id, username, password_hash, status, oauth_provider, oauth_provider_id, email, auth_source, created_at, updated_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, user.Id, user.Username, user.PasswordHash, user.Status, user.OAuthProvider, user.OAuthProviderId, user.Email, user.AuthSource, user.CreatedAt, user.UpdatedAt, user.LastLoginAt)
	if err != nil {
		return fmt.Errorf("create user %s: %w", user.Username, err)
	}
	return nil
}

func (s Store) UpdateUser(ctx context.Context, user User) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET username = ?, password_hash = ?, status = ?, email = ?, auth_source = ?, oauth_provider = ?, oauth_provider_id = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)),
		user.Username, user.PasswordHash, user.Status, user.Email, user.AuthSource, user.OAuthProvider, user.OAuthProviderId, user.Id)
	if err != nil {
		return fmt.Errorf("update user %s: %w", user.Id, err)
	}
	return nil
}

func (s Store) SetUserStatus(ctx context.Context, userId string, status string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), status, userId)
	if err != nil {
		return fmt.Errorf("set user status %s: %w", userId, err)
	}
	return nil
}

func (s Store) SetUserRoles(ctx context.Context, userId string, roleIds []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
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
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO user_role (user_id, role_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(s.driver)), userId, roleId); err != nil {
			return fmt.Errorf("insert user role %s/%s: %w", userId, roleId, err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE user SET updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), userId); err != nil {
		return fmt.Errorf("touch user %s: %w", userId, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit set user roles %s: %w", userId, err)
	}
	committed = true
	return nil
}

func (s Store) DeleteUser(ctx context.Context, userId string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user WHERE id = ?`, userId)
	if err != nil {
		return fmt.Errorf("delete user %s: %w", userId, err)
	}
	return nil
}

func (s Store) RoleById(ctx context.Context, id string) (Role, error) {
	var role Role
	err := s.db.GetContext(ctx, &role, `SELECT id, code, name, description, created_at, updated_at FROM role WHERE id = ?`, id)
	if err != nil {
		return Role{}, fmt.Errorf("load role %s: %w", id, err)
	}
	return role, nil
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
