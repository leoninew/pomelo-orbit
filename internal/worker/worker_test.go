package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"backend/internal/db"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
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

func TestWorkerCompletesTask(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := taskrepo.NewRepository(database, "sqlite")
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypeCIPipelineRunExecute, `{}`, 1); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypeCIPipelineRunExecute, HandlerFunc(func(ctx context.Context, item taskrepo.Task) error { return nil }))
	worker := New(repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

	if err := worker.runOnce(context.Background(), 0); err != nil {
		t.Fatal(err)
	}

	item, err := repo.FindById(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != status.TaskSucceeded {
		t.Fatalf("unexpected status: %s", item.Status)
	}
}

func TestWorkerRequeuesFailedTask(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := taskrepo.NewRepository(database, "sqlite")
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypeCIPipelineRunExecute, `{}`, 2); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypeCIPipelineRunExecute, HandlerFunc(func(ctx context.Context, item taskrepo.Task) error { return errors.New("boom") }))
	worker := New(repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

	if err := worker.runOnce(context.Background(), 0); err != nil {
		t.Fatal(err)
	}

	item, err := repo.FindById(context.Background(), "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Status != status.TaskPending {
		t.Fatalf("unexpected status: %s", item.Status)
	}
}
