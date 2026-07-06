package rolerepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"gitee.com/leoninew/PomeloOrbit-go/internal/db"
	dbsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/db/sqlc"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
)

type Repository struct {
	db      *sqlx.DB
	driver  string
	queries *dbsqlc.Queries
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver, queries: dbsqlc.New(db)}
}

func (r Repository) RoleById(ctx context.Context, id string) (model.Role, error) {
	role, err := r.queries.RoleByID(ctx, id)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role %s: %w", id, err)
	}
	return dbmodel.RoleFromSQLC(role), nil
}

func (r Repository) RoleByCode(ctx context.Context, code string) (model.Role, error) {
	role, err := r.queries.RoleByCode(ctx, code)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role by code %s: %w", code, err)
	}
	return dbmodel.RoleFromSQLC(role), nil
}

func (r Repository) RoleByName(ctx context.Context, name string) (model.Role, error) {
	role, err := r.queries.RoleByName(ctx, name)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role by name %s: %w", name, err)
	}
	return dbmodel.RoleFromSQLC(role), nil
}

func (r Repository) ListRoles(ctx context.Context, page int, perPage int, search string) (repository.Page[model.Role], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := roleSearchWhere(search)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM role`+where, args...); err != nil {
		return repository.Page[model.Role]{}, fmt.Errorf("count roles: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Role
	err := r.db.SelectContext(ctx, &items, `SELECT id, code, name, description, created_at, updated_at FROM role`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Role]{}, fmt.Errorf("list roles: %w", err)
	}
	return repository.Page[model.Role]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	rows, err := r.queries.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	permissions := make([]model.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, dbmodel.PermissionFromSQLC(row))
	}
	return permissions, nil
}

func (r Repository) PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error) {
	found := map[string]struct{}{}
	if len(codes) == 0 {
		return found, nil
	}
	query, args, err := sqlx.In(`SELECT code FROM permission WHERE code IN (?)`, codes)
	if err != nil {
		return nil, fmt.Errorf("build permission query: %w", err)
	}
	var rows []string
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load permissions by code: %w", err)
	}
	for _, code := range rows {
		found[code] = struct{}{}
	}
	return found, nil
}

func (r Repository) RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error) {
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
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("load role permissions: %w", err)
	}
	for _, row := range rows {
		permissionsByRoleId[row.RoleId] = append(permissionsByRoleId[row.RoleId], row.Code)
	}
	return permissionsByRoleId, nil
}

func (r Repository) CreateRole(ctx context.Context, role model.Role, permissionCodes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
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
	if err := setRolePermissionsTx(ctx, tx, r.driver, role.Id, permissionCodes); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create role %s: %w", role.Code, err)
	}
	committed = true
	return nil
}

func (r Repository) UpdateRole(ctx context.Context, role model.Role, permissionCodes []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update role %s: %w", role.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE role SET code = ?, name = ?, description = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), role.Code, role.Name, role.Description, role.Id); err != nil {
		return fmt.Errorf("update role %s: %w", role.Id, err)
	}
	if err := setRolePermissionsTx(ctx, tx, r.driver, role.Id, permissionCodes); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update role %s: %w", role.Id, err)
	}
	committed = true
	return nil
}

func (r Repository) DeleteRole(ctx context.Context, roleId string) error {
	err := r.queries.DeleteRole(ctx, roleId)
	if err != nil {
		return fmt.Errorf("delete role %s: %w", roleId, err)
	}
	return nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func setRolePermissionsTx(ctx context.Context, tx *sqlx.Tx, driver string, roleId string, permissionCodes []string) error {
	queries := dbsqlc.New(tx)
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permission WHERE role_id = ?`, roleId); err != nil {
		return fmt.Errorf("delete role permissions %s: %w", roleId, err)
	}
	for _, code := range permissionCodes {
		permissionId, err := queries.PermissionIDByCode(ctx, code)
		if err != nil {
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
