package rolesvc

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	roledto "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var codePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Service struct {
	repo repository.RoleStore
}

func New(repo repository.RoleStore) Service {
	return Service{repo: repo}
}

func (s Service) List(ctx context.Context, page int, perPage int, search string) (repository.Page[roledto.Detail], error) {
	roles, err := s.repo.ListRoles(ctx, page, perPage, search)
	if err != nil {
		return repository.Page[roledto.Detail]{}, err
	}
	roleIds := make([]string, 0, len(roles.Items))
	for _, role := range roles.Items {
		roleIds = append(roleIds, role.Id)
	}
	permissionsByRoleId, err := s.repo.RolePermissionCodesByRoleIds(ctx, roleIds)
	if err != nil {
		return repository.Page[roledto.Detail]{}, err
	}
	items := make([]roledto.Detail, 0, len(roles.Items))
	for _, role := range roles.Items {
		items = append(items, roledto.Detail{Role: role, PermissionCodes: emptyStrings(permissionsByRoleId[role.Id])})
	}
	return repository.Page[roledto.Detail]{Items: items, Total: roles.Total, Page: roles.Page, PerPage: roles.PerPage}, nil
}

func (s Service) Detail(ctx context.Context, roleId string) (roledto.Detail, error) {
	role, err := s.find(ctx, roleId)
	if err != nil {
		return roledto.Detail{}, err
	}
	permissionsByRoleId, err := s.repo.RolePermissionCodesByRoleIds(ctx, []string{role.Id})
	if err != nil {
		return roledto.Detail{}, err
	}
	return roledto.Detail{Role: role, PermissionCodes: emptyStrings(permissionsByRoleId[role.Id])}, nil
}

func (s Service) ListPermissions(ctx context.Context) ([]roledto.Permission, error) {
	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]roledto.Permission, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, roledto.Permission{Id: permission.Id, Code: permission.Code, Name: permission.Name, Description: permission.Description})
	}
	return items, nil
}

func (s Service) Create(ctx context.Context, input roledto.SaveInput) (model.Role, error) {
	code, name, description, permissions, err := s.normalizeAndValidate(input)
	if err != nil {
		return model.Role{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, ""); err != nil {
		return model.Role{}, err
	}
	if err := s.ensureNameAvailable(ctx, name, ""); err != nil {
		return model.Role{}, err
	}
	if err := s.ensurePermissionsExist(ctx, permissions); err != nil {
		return model.Role{}, err
	}
	now := time.Now().UTC()
	role := model.Role{Id: idutil.NewId(), Code: code, Name: name, Description: description, CreatedAt: now, UpdatedAt: now}
	if err := s.repo.CreateRole(ctx, role, permissions); err != nil {
		return model.Role{}, err
	}
	return role, nil
}

func (s Service) UpdateById(ctx context.Context, roleId string, input roledto.SaveInput) (roledto.Detail, error) {
	role, err := s.find(ctx, roleId)
	if err != nil {
		return roledto.Detail{}, err
	}
	updated, err := s.update(ctx, role, input)
	if err != nil {
		return roledto.Detail{}, err
	}
	return s.Detail(ctx, updated.Id)
}

func (s Service) Delete(ctx context.Context, roleId string) error {
	role, err := s.find(ctx, roleId)
	if err != nil {
		return err
	}
	return s.repo.DeleteRole(ctx, role.Id)
}

func (s Service) update(ctx context.Context, role model.Role, input roledto.SaveInput) (model.Role, error) {
	code, name, description, permissions, err := s.normalizeAndValidate(input)
	if err != nil {
		return model.Role{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, role.Id); err != nil {
		return model.Role{}, err
	}
	if err := s.ensureNameAvailable(ctx, name, role.Id); err != nil {
		return model.Role{}, err
	}
	if err := s.ensurePermissionsExist(ctx, permissions); err != nil {
		return model.Role{}, err
	}
	role.Code = code
	role.Name = name
	role.Description = description
	if err := s.repo.UpdateRole(ctx, role, permissions); err != nil {
		return model.Role{}, err
	}
	return s.repo.RoleById(ctx, role.Id)
}

func (s Service) find(ctx context.Context, roleId string) (model.Role, error) {
	roleId = strings.TrimSpace(roleId)
	role, err := s.repo.RoleById(ctx, roleId)
	if err == nil {
		return role, nil
	}
	if errors.Is(err, repository.ErrNotFound) {
		return model.Role{}, apperror.New(apperror.KindNotFound, "Role "+roleId+" not found")
	}
	return model.Role{}, err
}

func (s Service) normalizeAndValidate(input roledto.SaveInput) (string, string, *string, []string, error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	description := normalizeOptional(input.Description)
	if code == "" || len(code) > 50 || !codePattern.MatchString(code) || name == "" || len(name) > 100 || (description != nil && len(*description) > 500) {
		return "", "", nil, nil, ErrInvalidRoleFields
	}
	permissions, err := normalizePermissions(input.PermissionCodes)
	if err != nil {
		return "", "", nil, nil, err
	}
	return code, name, description, permissions, nil
}

func (s Service) ensureCodeAvailable(ctx context.Context, code string, currentRoleId string) error {
	existing, err := s.repo.RoleByCode(ctx, code)
	if err == nil {
		if existing.Id != currentRoleId {
			return apperror.New(apperror.KindConflict, "Role code "+code+" already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	return nil
}

func (s Service) ensureNameAvailable(ctx context.Context, name string, currentRoleId string) error {
	existing, err := s.repo.RoleByName(ctx, name)
	if err == nil {
		if existing.Id != currentRoleId {
			return apperror.New(apperror.KindConflict, "Role name "+name+" already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return err
	}
	return nil
}

func (s Service) ensurePermissionsExist(ctx context.Context, permissionCodes []string) error {
	found, err := s.repo.PermissionCodesExist(ctx, permissionCodes)
	if err != nil {
		return err
	}
	for _, code := range permissionCodes {
		if _, ok := found[code]; !ok {
			return apperror.New(apperror.KindNotFound, "Permission "+code+" not found")
		}
	}
	return nil
}

func normalizePermissions(values []string) ([]string, error) {
	if values == nil {
		return []string{}, nil
	}
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value)
		if code == "" {
			return nil, ErrInvalidPermissionCode
		}
		if _, ok := seen[code]; ok {
			return nil, ErrDuplicatePermissionCodes
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	sort.Strings(result)
	return result, nil
}

func emptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

var (
	ErrInvalidRoleFields        = apperror.New(apperror.KindValidation, "Invalid role fields")
	ErrInvalidPermissionCode    = apperror.New(apperror.KindValidation, "Invalid permission code")
	ErrDuplicatePermissionCodes = apperror.New(apperror.KindValidation, "permission_codes must be unique")
)
