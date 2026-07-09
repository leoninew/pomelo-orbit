package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

type blockingRuntimeWorker struct {
	started chan struct{}
}

func (w blockingRuntimeWorker) Run(ctx context.Context) error {
	close(w.started)
	<-ctx.Done()
	return ctx.Err()
}

func TestRunHTTPServerAndWorkerReturnsServeErrorAndCancelsWorker(t *testing.T) {
	worker := blockingRuntimeWorker{started: make(chan struct{})}
	server := &http.Server{Addr: "invalid-address", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}

	err := runHTTPServerAndWorker(context.Background(), testLogger(), server.Addr, server, worker)
	if err == nil || !strings.Contains(err.Error(), "serve http") {
		t.Fatalf("expected serve http error, got %v", err)
	}
	select {
	case <-worker.started:
	case <-time.After(time.Second):
		t.Fatal("expected worker to start before cancellation")
	}
}

func TestRunHTTPServerAndWorkerReturnsContextErrorOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	worker := blockingRuntimeWorker{started: make(chan struct{})}
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runHTTPServerAndWorker(ctx, testLogger(), server.Addr, server, worker)
	}()

	select {
	case <-worker.started:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("expected worker to start")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expected runtime to stop after context cancellation")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
