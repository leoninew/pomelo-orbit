package rolesvc

import (
	"context"
	"testing"

	roledto "github.com/leoninew/pomelo-orbit/internal/application/role/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type permissionFacts struct{ repository.RoleStore }

func (permissionFacts) RoleByCode(context.Context, string) (model.Role, error) {
	return model.Role{}, repository.ErrNotFound
}
func (permissionFacts) RoleByName(context.Context, string) (model.Role, error) {
	return model.Role{}, repository.ErrNotFound
}
func (permissionFacts) PermissionCodesExist(context.Context, []string) (map[string]struct{}, error) {
	return map[string]struct{}{"user:read": {}}, nil
}
func TestRoleServiceRejectsMissingPermissionAndDuplicatePermissions(t *testing.T) {
	service := New(permissionFacts{}, nil)
	ctx := context.Background()
	if _, err := service.Create(ctx, roledto.SaveInput{Code: "bad", Name: "Bad", PermissionCodes: []string{"missing:permission"}}); err == nil || !apperror.IsKind(err, apperror.KindNotFound) {
		t.Fatalf("expected missing permission to return 404, got %v", err)
	}
	if _, err := service.Create(ctx, roledto.SaveInput{Code: "bad", Name: "Bad", PermissionCodes: []string{"user:read", "user:read"}}); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected duplicate permissions to return 400, got %v", err)
	}
}
