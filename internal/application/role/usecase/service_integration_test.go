package rolesvc

import (
	"context"
	"testing"

	"database/sql"

	_ "modernc.org/sqlite"

	roledto "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	testseed "gitee.com/leoninew/PomeloOrbit-go/internal/testutil/seed"
)

func TestRoleServiceCreateUpdateAndDelete(t *testing.T) {
	service, database := newRoleIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	description := "Audit users"

	created, err := service.Create(ctx, roledto.SaveInput{Code: "auditor", Name: "Auditor", Description: &description, PermissionCodes: []string{"user:read"}})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "auditor" || created.Name != "Auditor" || created.Description == nil || *created.Description != description {
		t.Fatalf("unexpected created role: %+v", created)
	}
	updated, err := service.UpdateById(ctx, created.Id, roledto.SaveInput{Code: "auditor", Name: "Auditor Updated", PermissionCodes: []string{"role:read", "user:read"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Role.Name != "Auditor Updated" || updated.Role.Description != nil {
		t.Fatalf("unexpected updated role: %+v", updated.Role)
	}
	permissions, err := rolerepo.NewRepository(database).RolePermissionCodesByRoleIds(ctx, []string{updated.Role.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(permissions[updated.Role.Id]) != 2 || permissions[updated.Role.Id][0] != "role:read" || permissions[updated.Role.Id][1] != "user:read" {
		t.Fatalf("unexpected role permissions: %+v", permissions)
	}
	if err := service.Delete(ctx, updated.Role.Id); err != nil {
		t.Fatal(err)
	}
}

func TestRoleServiceRejectsMissingPermissionAndDuplicatePermissions(t *testing.T) {
	service, database := newRoleIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	if _, err := service.Create(ctx, roledto.SaveInput{Code: "bad", Name: "Bad", PermissionCodes: []string{"missing:permission"}}); err == nil || apperror.StatusCode(err) != 404 {
		t.Fatalf("expected missing permission to return 404, got %v", err)
	}
	if _, err := service.Create(ctx, roledto.SaveInput{Code: "bad", Name: "Bad", PermissionCodes: []string{"user:read", "user:read"}}); err == nil || apperror.StatusCode(err) != 400 {
		t.Fatalf("expected duplicate permissions to return 400, got %v", err)
	}
}

func newRoleIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	testseed.ApplySQLiteSystemSeed(t, database)
	return New(rolerepo.NewRepository(database)), database
}
