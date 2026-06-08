package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Handler interface {
	Handle(ctx context.Context, task Task) error
}

type HandlerFunc func(ctx context.Context, task Task) error

func (f HandlerFunc) Handle(ctx context.Context, task Task) error {
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

func (r *Router) Handle(ctx context.Context, task Task) error {
	handler, ok := r.handlers[task.TaskType]
	if !ok {
		return fmt.Errorf("no handler registered for task type: %s", task.TaskType)
	}
	return handler.Handle(ctx, task)
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

type WorkerConfig struct {
	WorkerId      string
	PollInterval  time.Duration
	LeaseDuration time.Duration
	Concurrency   int
}

func NewWorker(repo Repository, router Handler, logger *slog.Logger, cfg WorkerConfig) *Worker {
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
	task, err := w.repo.ClaimNext(ctx, w.workerId, w.leaseDuration)
	if err != nil {
		return err
	}
	if task == nil {
		return nil
	}

	logger := w.logger.With("slot", slot, "task_id", task.Id, "task_type", task.TaskType)
	logger.Info("task claimed")
	if err := w.router.Handle(ctx, *task); err != nil {
		logger.Error("task failed", "error", err)
		if failErr := w.repo.Fail(ctx, task.Id, err.Error()); failErr != nil {
			return errors.Join(err, failErr)
		}
		return nil
	}
	if err := w.repo.Complete(ctx, task.Id); err != nil {
		return err
	}
	logger.Info("task completed")
	return nil
}
