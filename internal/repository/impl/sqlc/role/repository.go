package rolerepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	rolesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/role"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.RoleStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *rolesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *rolesqlc.Queries {
		return rolesqlc.New(dbtx)
	})
}

func (r Repository) RoleById(ctx context.Context, id string) (model.Role, error) {
	role, err := r.q(ctx).RoleByID(ctx, id)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role %s: %w", id, sqlcommon.TranslateError(err))
	}
	return roleFromSQLC(role), nil
}

func (r Repository) RoleByCode(ctx context.Context, code string) (model.Role, error) {
	role, err := r.q(ctx).RoleByCode(ctx, code)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return roleFromSQLC(role), nil
}

func (r Repository) RoleByName(ctx context.Context, name string) (model.Role, error) {
	role, err := r.q(ctx).RoleByName(ctx, name)
	if err != nil {
		return model.Role{}, fmt.Errorf("load role by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return roleFromSQLC(role), nil
}

func (r Repository) ListRoles(ctx context.Context, page int, perPage int, search string) (repository.Page[model.Role], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.LowerSearchPattern(search)
	q := r.q(ctx)
	total, err := q.CountRoles(ctx, rolesqlc.CountRolesParams{
		Column1:     raw,
		Code:        pattern,
		Name:        pattern,
		Description: sql.NullString{String: pattern, Valid: true},
	})
	if err != nil {
		return repository.Page[model.Role]{}, fmt.Errorf("count roles: %w", err)
	}
	rows, err := q.ListRoles(ctx, rolesqlc.ListRolesParams{
		Column1:     raw,
		Code:        pattern,
		Name:        pattern,
		Description: sql.NullString{String: pattern, Valid: true},
		Limit:       int64(perPage),
		Offset:      int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Role]{}, fmt.Errorf("list roles: %w", err)
	}
	items := make([]model.Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, roleFromSQLC(row))
	}
	return repository.Page[model.Role]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	rows, err := r.q(ctx).ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	permissions := make([]model.Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, model.Permission{
			Id:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			Description: dbmodel.StringPtr(row.Description),
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return permissions, nil
}

func (r Repository) PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error) {
	found := map[string]struct{}{}
	if len(codes) == 0 {
		return found, nil
	}
	rows, err := r.q(ctx).PermissionCodesByCodes(ctx, codes)
	if err != nil {
		return nil, fmt.Errorf("load permissions by code: %w", err)
	}
	for _, code := range rows {
		found[code] = struct{}{}
	}
	return found, nil
}

func (r Repository) RolePermissionCodesByRoleIds(ctx context.Context, roleIds []string) (map[string][]string, error) {
	permissionsByRoleId := make(map[string][]string, len(roleIds))
	for _, roleId := range roleIds {
		permissionsByRoleId[roleId] = []string{}
	}
	if len(roleIds) == 0 {
		return permissionsByRoleId, nil
	}
	rows, err := r.q(ctx).RolePermissionCodesByRoleIds(ctx, roleIds)
	if err != nil {
		return nil, fmt.Errorf("load role permissions: %w", err)
	}
	for _, row := range rows {
		permissionsByRoleId[row.RoleID] = append(permissionsByRoleId[row.RoleID], row.Code)
	}
	return permissionsByRoleId, nil
}

func (r Repository) CreateRole(ctx context.Context, role model.Role, permissionCodes []string) error {
	q := r.q(ctx)
	if err := q.CreateRole(ctx, rolesqlc.CreateRoleParams{
		ID:          role.Id,
		Code:        role.Code,
		Name:        role.Name,
		Description: dbmodel.NullString(role.Description),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("create role %s: %w", role.Code, err)
	}
	if err := setRolePermissions(ctx, q, role.Id, permissionCodes); err != nil {
		return err
	}
	return nil
}

func (r Repository) UpdateRole(ctx context.Context, role model.Role, permissionCodes []string) error {
	q := r.q(ctx)
	if err := q.UpdateRole(ctx, rolesqlc.UpdateRoleParams{
		Code:        role.Code,
		Name:        role.Name,
		Description: dbmodel.NullString(role.Description),
		UpdatedAt:   time.Now().UTC(),
		ID:          role.Id,
	}); err != nil {
		return fmt.Errorf("update role %s: %w", role.Id, err)
	}
	if err := setRolePermissions(ctx, q, role.Id, permissionCodes); err != nil {
		return err
	}
	return nil
}

func (r Repository) DeleteRole(ctx context.Context, roleId string) error {
	if err := r.q(ctx).DeleteRole(ctx, roleId); err != nil {
		return fmt.Errorf("delete role %s: %w", roleId, err)
	}
	return nil
}

func setRolePermissions(ctx context.Context, q *rolesqlc.Queries, roleId string, permissionCodes []string) error {
	if err := q.DeleteRolePermissions(ctx, roleId); err != nil {
		return fmt.Errorf("delete role permissions %s: %w", roleId, err)
	}
	now := time.Now().UTC()
	for _, code := range permissionCodes {
		permissionId, err := q.PermissionIDByCode(ctx, code)
		if err != nil {
			return fmt.Errorf("load permission %s: %w", code, err)
		}
		if err := q.InsertRolePermission(ctx, rolesqlc.InsertRolePermissionParams{
			RoleID:       roleId,
			PermissionID: permissionId,
			CreatedAt:    now,
		}); err != nil {
			return fmt.Errorf("insert role permission %s/%s: %w", roleId, code, err)
		}
	}
	return nil
}

func roleFromSQLC(role rolesqlc.Role) model.Role {
	return model.Role{
		Id:          role.ID,
		Code:        role.Code,
		Name:        role.Name,
		Description: dbmodel.StringPtr(role.Description),
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}
