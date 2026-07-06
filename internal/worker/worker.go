package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/task"
)

type Handler interface {
	Handle(ctx context.Context, task taskrepo.Task) error
}

type HandlerFunc func(ctx context.Context, task taskrepo.Task) error

func (f HandlerFunc) Handle(ctx context.Context, task taskrepo.Task) error {
	return f(ctx, task)
}

type Router struct {
	handlers map[string]Handler
}

func NewRouter() *Router {
	return &Router{handlers: map[string]Handler{}}
}

func (r *Router) Register(taskType string, handler Handler) {
	r.handlers[taskType] = handler
}

func (r *Router) Handle(ctx context.Context, item taskrepo.Task) error {
	handler, ok := r.handlers[item.TaskType]
	if !ok {
		return fmt.Errorf("no handler registered for task type: %s", item.TaskType)
	}
	return handler.Handle(ctx, item)
}

type Repository interface {
	ClaimNext(ctx context.Context, workerId string, lockTimeout time.Duration) (*taskrepo.Task, error)
	Complete(ctx context.Context, taskId string) error
	Fail(ctx context.Context, taskId string, message string) error
}

type Worker struct {
	repo          Repository
	router        Handler
	logger        *slog.Logger
	workerId      string
	pollInterval  time.Duration
	leaseDuration time.Duration
	concurrency   int
}

type Config struct {
	WorkerId      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	Concurrency   int
}

func New(repo Repository, router Handler, logger *slog.Logger, cfg Config) *Worker {
	return &Worker{
		repo:          repo,
		router:        router,
		logger:        logger,
		workerId:      cfg.WorkerId,
		pollInterval:  cfg.PollInterval,
		leaseDuration: cfg.LeaseDuration,
		concurrency:   cfg.Concurrency,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	if w.concurrency < 1 {
		return errors.New("worker concurrency must be at least 1")
	}

	w.logger.Info("background worker started", "worker_id", w.workerId, "concurrency", w.concurrency)
	var wg sync.WaitGroup
	for i := range w.concurrency {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			w.runSlot(ctx, slot)
		}(i)
	}

	<-ctx.Done()
	wg.Wait()
	w.logger.Info("background worker stopped", "worker_id", w.workerId)
	return ctx.Err()
}

func (w *Worker) runSlot(ctx context.Context, slot int) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		if err := w.runOnce(ctx, slot); err != nil {
			w.logger.Error("worker iteration failed", "slot", slot, "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (w *Worker) runOnce(ctx context.Context, slot int) error {
	taskItem, err := w.repo.ClaimNext(ctx, w.workerId, w.leaseDuration)
	if err != nil {
		return err
	}
	if taskItem == nil {
		return nil
	}

	logger := w.logger.With("slot", slot, "task_id", taskItem.Id, "task_type", taskItem.TaskType)
	logger.Info("task claimed")
	if err := w.router.Handle(ctx, *taskItem); err != nil {
		logger.Error("task failed", "error", err)
		if failErr := w.repo.Fail(ctx, taskItem.Id, err.Error()); failErr != nil {
			return errors.Join(err, failErr)
		}
		return nil
	}
	if err := w.repo.Complete(ctx, taskItem.Id); err != nil {
		return err
	}
	logger.Info("task completed")
	return nil
}
