package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

type runtimeWorker interface {
	Run(context.Context) error
}

func runHTTPServerAndWorker(ctx context.Context, logger *slog.Logger, addr string, httpServer *http.Server, backgroundWorker runtimeWorker) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var shutdownOnce sync.Once
	shutdownHTTP := func() {
		shutdownOnce.Do(func() {
			if err := httpServer.Shutdown(context.Background()); err != nil {
				logger.Error("http server shutdown failed", "error", err)
			}
		})
	}
	go func() {
		<-runCtx.Done()
		shutdownHTTP()
	}()

	errCh := make(chan error, 2)
	go func() {
		logger.Info("http server started", "addr", addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve http: %w", err)
			return
		}
		errCh <- nil
	}()
	go func() {
		if err := backgroundWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("run background worker: %w", err)
			return
		}
		errCh <- nil
	}()

	var firstErr error
	for completed := 0; completed < 2; completed++ {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			cancel()
			shutdownHTTP()
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}
