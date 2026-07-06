package taskrepo

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/db"
	"gitee.com/leoninew/PomeloOrbit-go/internal/status"
)

func openTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, "sqlite"); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestRepositoryClaimComplete(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database, "sqlite")
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "task-1", status.TaskTypeCIPipelineRunExecute, `{"pipeline_run_id":"run-1"}`, 3); err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil {
		t.Fatal("expected task to be claimed")
	}
	if claimed.Id != "task-1" {
		t.Fatalf("unexpected task id: %s", claimed.Id)
	}

	if err := repo.Complete(ctx, claimed.Id); err != nil {
		t.Fatal(err)
	}

	completed, err := repo.FindById(ctx, claimed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != status.TaskSucceeded {
		t.Fatalf("unexpected status: %s", completed.Status)
	}
}

func TestRepositoryFailRetriesUntilMaxAttempts(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database, "sqlite")
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "task-1", status.TaskTypeCIPipelineRunExecute, `{}`, 2); err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(ctx, claimed.Id, "first failure"); err != nil {
		t.Fatal(err)
	}
	failedOnce, err := repo.FindById(ctx, claimed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if failedOnce.Status != status.TaskPending {
		t.Fatalf("expected retry pending, got %s", failedOnce.Status)
	}

	claimedAgain, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(ctx, claimedAgain.Id, "second failure"); err != nil {
		t.Fatal(err)
	}
	failedTwice, err := repo.FindById(ctx, claimedAgain.Id)
	if err != nil {
		t.Fatal(err)
	}
	if failedTwice.Status != status.TaskFailed {
		t.Fatalf("expected final failure, got %s", failedTwice.Status)
	}
}
