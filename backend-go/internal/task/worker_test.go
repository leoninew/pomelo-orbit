package task

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"backend/internal/status"
)

func TestWorkerRunOnceCompletesTask(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database)
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypeCIPipelineRunExecute, `{}`, 1); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypeCIPipelineRunExecute, HandlerFunc(func(ctx context.Context, task Task) error { return nil }))
	worker := NewWorker(repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), WorkerConfig{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

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

func TestWorkerRunOnceRequeuesFailedTask(t *testing.T) {
	database := openTestDB(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database)
	if err := repo.Enqueue(context.Background(), "task-1", status.TaskTypeCIPipelineRunExecute, `{}`, 2); err != nil {
		t.Fatal(err)
	}
	router := NewRouter()
	router.Register(status.TaskTypeCIPipelineRunExecute, HandlerFunc(func(ctx context.Context, task Task) error { return errors.New("boom") }))
	worker := NewWorker(repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), WorkerConfig{WorkerId: "worker-1", PollInterval: time.Millisecond, LeaseDuration: time.Minute, Concurrency: 1})

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
