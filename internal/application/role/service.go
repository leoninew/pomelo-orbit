package rolesvc

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

var codePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Repository interface {
	RoleById(ctx context.Context, id string) (model.Role, error)
	RoleByCode(ctx context.Context, code string) (model.Role, error)
	RoleByName(ctx context.Context, name string) (model.Role, error)
	PermissionCodesExist(ctx context.Context, codes []string) (map[string]struct{}, error)
	CreateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	UpdateRole(ctx context.Context, role model.Role, permissionCodes []string) error
	DeleteRole(ctx context.Context, roleId string) error
}

type Service struct {
	repo Repository
}

type SaveInput struct {
	Role            model.Role
	Code            string
	Name            string
	Description     *string
	PermissionCodes []string
}

func New(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) Create(ctx context.Context, input SaveInput) (model.Role, error) {
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

func (s Service) Update(ctx context.Context, input SaveInput) (model.Role, error) {
	code, name, description, permissions, err := s.normalizeAndValidate(input)
	if err != nil {
		return model.Role{}, err
	}
	if err := s.ensureCodeAvailable(ctx, code, input.Role.Id); err != nil {
		return model.Role{}, err
	}
	if err := s.ensureNameAvailable(ctx, name, input.Role.Id); err != nil {
		return model.Role{}, err
	}
	if err := s.ensurePermissionsExist(ctx, permissions); err != nil {
		return model.Role{}, err
	}
	role := input.Role
	role.Code = code
	role.Name = name
	role.Description = description
	if err := s.repo.UpdateRole(ctx, role, permissions); err != nil {
		return model.Role{}, err
	}
	return s.repo.RoleById(ctx, role.Id)
}

func (s Service) Delete(ctx context.Context, roleId string) error {
	return s.repo.DeleteRole(ctx, roleId)
}

func (s Service) normalizeAndValidate(input SaveInput) (string, string, *string, []string, error) {
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
	if !errors.Is(err, sql.ErrNoRows) {
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
	if !errors.Is(err, sql.ErrNoRows) {
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
