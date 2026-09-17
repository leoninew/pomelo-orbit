package userrepo

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

func setupTestUserDB(t *testing.T) *sql.DB {
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

func TestSetUserRolesSucceedsAtomically(t *testing.T) {
	database := setupTestUserDB(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)

	userId := "user-test-1"
	user := model.User{
		Id:           userId,
		Username:     "testuser1",
		PasswordHash: "hash",
		Email:        "test1@example.com",
		Status:       "enabled",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// 插入合法角色
	if _, err := database.ExecContext(ctx, `INSERT INTO role (id, code, name) VALUES ('role-u1', 'viewer', 'Viewer'), ('role-u2', 'editor', 'Editor')`); err != nil {
		t.Fatal(err)
	}

	// 设置为 viewer
	if err := repo.SetUserRoles(ctx, userId, []string{"role-u1"}); err != nil {
		t.Fatalf("SetUserRoles failed: %v", err)
	}

	roles, err := repo.UserRoles(ctx, userId)
	if err != nil {
		t.Fatalf("UserRoles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "viewer" {
		t.Fatalf("expected [viewer], got %v", roles)
	}

	// 更新为 editor
	if err := repo.SetUserRoles(ctx, userId, []string{"role-u2"}); err != nil {
		t.Fatalf("SetUserRoles failed: %v", err)
	}

	roles, err = repo.UserRoles(ctx, userId)
	if err != nil {
		t.Fatalf("UserRoles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "editor" {
		t.Fatalf("expected [editor], got %v", roles)
	}
}

func TestSetUserRolesRollsBackOnError(t *testing.T) {
	database := setupTestUserDB(t)
	defer func() { _ = database.Close() }()

	ctx := context.Background()
	repo := NewRepository(database)

	userId := "user-test-2"
	user := model.User{
		Id:           userId,
		Username:     "testuser2",
		PasswordHash: "hash",
		Email:        "test2@example.com",
		Status:       "enabled",
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if _, err := database.ExecContext(ctx, `INSERT INTO role (id, code, name) VALUES ('role-init', 'initial_role', 'Initial Role')`); err != nil {
		t.Fatal(err)
	}

	// 初始分配 initial_role
	if err := repo.SetUserRoles(ctx, userId, []string{"role-init"}); err != nil {
		t.Fatalf("initial SetUserRoles failed: %v", err)
	}

	// 在外部事务上下文中执行，并在出错时回滚，验证仓储在事务下的正确性
	sqlTx, err := database.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	txCtx := tx.WithTx(ctx, sqlTx)

	// 传入重复的 roleId 触发主键冲突错误
	err = repo.SetUserRoles(txCtx, userId, []string{"role-init", "role-init"})
	if err == nil {
		t.Fatal("expected SetUserRoles to fail on duplicate roleId")
	}
	_ = sqlTx.Rollback()

	// 验证回滚：原角色 initial_role 仍完整保留
	roles, err := repo.UserRoles(ctx, userId)
	if err != nil {
		t.Fatalf("UserRoles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "initial_role" {
		t.Fatalf("expected [initial_role] to be preserved after rollback, got %v", roles)
	}
}
