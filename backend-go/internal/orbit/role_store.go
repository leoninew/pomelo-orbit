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

func (s Store) RoleByCode(ctx context.Context, code string) (Role, error) {
	var role Role
	err := s.db.GetContext(ctx, &role, `SELECT id, code, name, description, created_at, updated_at FROM role WHERE code = ?`, code)
	if err != nil {
		return Role{}, fmt.Errorf("load role by code %s: %w", code, err)
	}
	return role, nil
}

func (s Store) RoleByName(ctx context.Context, name string) (Role, error) {
	var role Role
	err := s.db.GetContext(ctx, &role, `SELECT id, code, name, description, created_at, updated_at FROM role WHERE name = ?`, name)
	if err != nil {
		return Role{}, fmt.Errorf("load role by name %s: %w", name, err)
	}
	return role, nil
}

func (s Store) ListRoles(ctx context.Context, page int, perPage int, search string) (Page[Role], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := roleSearchWhere(search)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM role`+where, args...); err != nil {
		return Page[Role]{}, fmt.Errorf("count roles: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Role
	err := s.db.SelectContext(ctx, &items, `SELECT id, code, name, description, created_at, updated_at FROM role`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Role]{}, fmt.Errorf("list roles: %w", err)
	}
	return Page[Role]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListPermissions(ctx context.Context) ([]Permission, error) {
	var permissions []Permission
	err := s.db.SelectContext(ctx, &permissions, `SELECT id, code, name, description, created_at, updated_at FROM permission ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	return permissions, nil
}

func (s Store) PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error) {
	found := map[string]struct{}{}
	if len(codes) == 0 {
		return found, nil
	}
	query, args, err := sqlx.In(`SELECT code FROM permission WHERE code IN (?)`, codes)
	if err != nil {
		return nil, fmt.Errorf("build permission query: %w", err)
	}
	var rows []string
	if err := s.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load permissions by code: %w", err)
	}
	for _, code := range rows {
		found[code] = struct{}{}
	}
	return found, nil
}

func (s Store) CreateRole(ctx context.Context, role Role, permissionCodes []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create role %s: %w", role.Code, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `INSERT INTO role (id, code, name, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, role.Id, role.Code, role.Name, role.Description, role.CreatedAt, role.UpdatedAt); err != nil {
		return fmt.Errorf("create role %s: %w", role.Code, err)
	}
	if err := setRolePermissionsTx(ctx, tx, s.driver, role.Id, permissionCodes); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create role %s: %w", role.Code, err)
	}
	committed = true
	return nil
}

func (s Store) UpdateRole(ctx context.Context, role Role, permissionCodes []string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update role %s: %w", role.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE role SET code = ?, name = ?, description = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), role.Code, role.Name, role.Description, role.Id); err != nil {
		return fmt.Errorf("update role %s: %w", role.Id, err)
	}
	if err := setRolePermissionsTx(ctx, tx, s.driver, role.Id, permissionCodes); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update role %s: %w", role.Id, err)
	}
	committed = true
	return nil
}

func (s Store) DeleteRole(ctx context.Context, roleId string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM role WHERE id = ?`, roleId)
	if err != nil {
		return fmt.Errorf("delete role %s: %w", roleId, err)
	}
	return nil
}

func RoleIsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func setRolePermissionsTx(ctx context.Context, tx *sqlx.Tx, driver string, roleId string, permissionCodes []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permission WHERE role_id = ?`, roleId); err != nil {
		return fmt.Errorf("delete role permissions %s: %w", roleId, err)
	}
	for _, code := range permissionCodes {
		var permissionId string
		if err := tx.GetContext(ctx, &permissionId, `SELECT id FROM permission WHERE code = ?`, code); err != nil {
			return fmt.Errorf("load permission %s: %w", code, err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO role_permission (role_id, permission_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(driver)), roleId, permissionId); err != nil {
			return fmt.Errorf("insert role permission %s/%s: %w", roleId, code, err)
		}
	}
	return nil
}

func roleSearchWhere(search string) (string, []any) {
	search = strings.TrimSpace(search)
	if search == "" {
		return "", nil
	}
	like := "%" + strings.ToLower(search) + "%"
	return " WHERE LOWER(code) LIKE ? OR LOWER(name) LIKE ? OR LOWER(COALESCE(description, '')) LIKE ?", []any{like, like, like}
}
