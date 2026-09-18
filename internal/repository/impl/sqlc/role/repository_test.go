package rolerepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func setupTestRoleDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	return database
}

func TestCreateRoleRollsBackWhenPermissionFails(t *testing.T) {
	database := setupTestRoleDB(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)

	if _, err := database.ExecContext(ctx, `INSERT INTO permission (id, code, name) VALUES ('perm-1', 'perm:read', 'Read')`); err != nil {
		t.Fatal(err)
	}

	role := model.Role{
		Id:        "role-1",
		Code:      "tester",
		Name:      "Tester",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	sqlTx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	txCtx := tx.WithTx(ctx, sqlTx)

	err = repo.CreateRole(txCtx, role, []string{"perm:read", "perm:non_existent"})
	if err == nil {
		t.Fatal("expected CreateRole to fail for unknown permission code")
	}
	_ = sqlTx.Rollback()

	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM role WHERE id = 'role-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected role record to be rolled back, but found count = %d", count)
	}
}

func TestCreateRoleSucceedsAtomically(t *testing.T) {
	database := setupTestRoleDB(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)

	if _, err := database.ExecContext(ctx, `INSERT INTO permission (id, code, name) VALUES ('perm-1', 'perm:read', 'Read')`); err != nil {
		t.Fatal(err)
	}

	role := model.Role{
		Id:        "role-2",
		Code:      "developer",
		Name:      "Developer",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := repo.CreateRole(ctx, role, []string{"perm:read"}); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	created, err := repo.RoleById(ctx, "role-2")
	if err != nil {
		t.Fatalf("RoleById failed: %v", err)
	}
	if created.Code != "developer" {
		t.Fatalf("expected code developer, got %s", created.Code)
	}

	permCodes, err := repo.RolePermissionCodesByRoleIds(ctx, []string{"role-2"})
	if err != nil {
		t.Fatalf("RolePermissionCodesByRoleIds failed: %v", err)
	}
	if len(permCodes["role-2"]) != 1 || permCodes["role-2"][0] != "perm:read" {
		t.Fatalf("expected perm:read, got %v", permCodes["role-2"])
	}
}

func TestUpdateRoleRollsBackWhenPermissionFails(t *testing.T) {
	database := setupTestRoleDB(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)

	if _, err := database.ExecContext(ctx, `INSERT INTO permission (id, code, name) VALUES ('perm-1', 'perm:read', 'Read')`); err != nil {
		t.Fatal(err)
	}

	role := model.Role{
		Id:        "role-3",
		Code:      "manager",
		Name:      "Manager",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := repo.CreateRole(ctx, role, []string{"perm:read"}); err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	role.Name = "Senior Manager"
	sqlTx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	txCtx := tx.WithTx(ctx, sqlTx)

	err = repo.UpdateRole(txCtx, role, []string{"perm:invalid"})
	if err == nil {
		t.Fatal("expected UpdateRole to fail for unknown permission code")
	}
	_ = sqlTx.Rollback()

	unchanged, err := repo.RoleById(ctx, "role-3")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Name != "Manager" {
		t.Fatalf("expected role name to remain Manager after rollback, got %s", unchanged.Name)
	}

	permCodes, err := repo.RolePermissionCodesByRoleIds(ctx, []string{"role-3"})
	if err != nil {
		t.Fatal(err)
	}
	if len(permCodes["role-3"]) != 1 || permCodes["role-3"][0] != "perm:read" {
		t.Fatalf("expected perm:read preserved, got %v", permCodes["role-3"])
	}
}
