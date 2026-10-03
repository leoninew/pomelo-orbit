package operation

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

type logAttrsKey struct{}

func WithLogAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	current, _ := ctx.Value(logAttrsKey{}).([]slog.Attr)
	merged := append([]slog.Attr(nil), current...)
	for _, attr := range attrs {
		found := false
		for i := range merged {
			if merged[i].Key == attr.Key {
				merged[i] = attr
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, attr)
		}
	}
	return context.WithValue(ctx, logAttrsKey{}, merged)
}

func Logger(ctx context.Context) *slog.Logger {
	attrs, _ := ctx.Value(logAttrsKey{}).([]slog.Attr)
	values := make([]any, len(attrs))
	for i, attr := range attrs {
		values[i] = attr
	}
	return slog.Default().With(values...)
}

// StartStage measures an operation phase and optionally gives it its own deadline.
func StartStage(ctx context.Context, operation, phase string, timeout time.Duration, attrs ...any) (context.Context, func(error, ...any) error) {
	cancel := func() {}
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	}
	started := time.Now()
	attrs = append(attrs, "phase", phase)
	if timeout > 0 {
		attrs = append(attrs, "timeout_ms", timeout.Milliseconds())
	}
	logger := Logger(ctx)
	logger.InfoContext(ctx, operation+" stage started", attrs...)
	return ctx, func(err error, completedAttrs ...any) error {
		defer cancel()
		if err != nil && ctx.Err() != nil && !errors.Is(err, ctx.Err()) {
			err = errors.Join(err, ctx.Err())
		}
		status := "completed"
		if err != nil {
			status = "failed"
			if errors.Is(err, context.DeadlineExceeded) {
				status = "timed_out"
			} else if errors.Is(err, context.Canceled) {
				status = "canceled"
			}
		}
		attrs = append(attrs, completedAttrs...)
		attrs = append(attrs, "elapsed_ms", time.Since(started).Milliseconds(), "status", status)
		if err != nil {
			attrs = append(attrs, "error", err)
			logger.WarnContext(ctx, operation+" stage finished", attrs...)
		} else {
			logger.InfoContext(ctx, operation+" stage finished", attrs...)
		}
		return err
	}
}
