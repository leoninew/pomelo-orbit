package worker

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

func openTestDb(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
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
	database := openTestDb(t)
	defer func() { _ = database.Close() }()

	repo := taskrepo.NewRepository(database)
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypePipelineRunExecute, `{}`, 1); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypePipelineRunExecute, HandlerFunc(func(ctx context.Context, item tasksvc.Task) error { return nil }))
	worker := New(database, repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

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
	database := openTestDb(t)
	defer func() { _ = database.Close() }()

	repo := taskrepo.NewRepository(database)
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypePipelineRunExecute, `{}`, 2); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypePipelineRunExecute, HandlerFunc(func(ctx context.Context, item tasksvc.Task) error { return errors.New("boom") }))
	worker := New(database, repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

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
