package traefik

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
)

type readinessRuntime struct {
	deploymentport.Runtime
	block bool
	calls int
}

func (r *readinessRuntime) QueryAtEnvironmentRoot(ctx context.Context, _ environmentport.Target, _ string, _ ...string) (string, error) {
	r.calls++
	if r.block {
		<-ctx.Done()
		return "", ctx.Err()
	}
	return "", errors.New("gateway unavailable")
}

func TestGatewayReadinessBoundsRequestsAndRetryWait(t *testing.T) {
	for _, tc := range []struct {
		name      string
		block     bool
		cancel    bool
		budget    time.Duration
		wantTime  time.Duration
		wantCause error
	}{
		{"request exceeds total budget", true, false, 750 * time.Millisecond, 750 * time.Millisecond, context.DeadlineExceeded},
		{"retry wait exceeds remaining budget", false, false, 250 * time.Millisecond, 250 * time.Millisecond, context.DeadlineExceeded},
		{"caller cancels during retry wait", false, true, 3 * time.Second, 100 * time.Millisecond, context.Canceled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if tc.cancel {
					time.AfterFunc(tc.wantTime, cancel)
				}
				runtime := &readinessRuntime{block: tc.block}
				manager := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
				started := time.Now()
				err := manager.WaitUntilReady(ctx, "project-1", testGateway(), tc.budget)
				if !errors.Is(err, tc.wantCause) || time.Since(started) != tc.wantTime || runtime.calls != 1 {
					t.Fatalf("error=%v elapsed=%v requests=%d", err, time.Since(started), runtime.calls)
				}
			})
		})
	}
}
