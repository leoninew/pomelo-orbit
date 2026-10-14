package deploymentsvc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestSupersededCancellationReturnsBeforeUnresponsiveStore(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release, exited := make(chan struct{}), make(chan struct{})
		defer close(release)
		service := ExecutionService{
			executionStore: blockedTakeoverCancellation{release: release, exited: exited},
			cancelTimeout:  30 * time.Millisecond,
		}
		service.cancelSupersededDeployments(context.Background(), "project-1", "service-1", "deployment-1")
		select {
		case <-exited:
			t.Fatal("cancellation waited for the unresponsive store")
		default:
		}
	})
}

func TestNewOperationCompletesBeforeOldRuntimeExitsDespiteCancellationFailure(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "late success", true: "late failure"}[lateFailure], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				service, store, logs := controlTestService()
				store.current.Store("first")
				started, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
				var releaseOnce sync.Once
				defer releaseOnce.Do(func() { close(release) })
				oldDone := make(chan error, 1)
				go func() {
					oldDone <- service.executeDeployment(context.Background(), "project-1", "app-1", "first", "stop", func(context.Context, model.Application, model.Deployment) error {
						close(started)
						defer close(exited)
						<-release
						if lateFailure {
							return errors.New("late Docker failure")
						}
						return nil
					})
				}()
				<-started
				// Supply the current-operation fact established by admission.
				// SQL admission and conditional writes are verified in operation_takeover_test.go.
				store.current.Store("second")
				ran := false
				if err := service.executeDeployment(context.Background(), "project-1", "app-1", "second", "stop", func(context.Context, model.Application, model.Deployment) error { ran = true; return nil }); err != nil {
					t.Fatal(err)
				}
				if !ran {
					t.Fatal("new operation did not execute")
				}
				if err := <-oldDone; err != nil {
					t.Fatalf("old cancellation became an error: %v", err)
				}
				select {
				case <-exited:
					t.Fatal("new operation waited for old runtime exit")
				default:
				}
				releaseOnce.Do(func() { close(release) })
				synctest.Wait()
				writes, completions := store.results()
				if len(writes) != 1 || writes[0] != "second" || len(completions) != 1 || completions[0].id != "second" || completions[0].status != status.WorkStatusRanToCompletion {
					t.Fatalf("writes=%v completions=%+v", writes, completions)
				}
				if output := logs.String(); !strings.Contains(output, "old cancellation write failed") || !strings.Contains(output, `"level":"WARN"`) || strings.Contains(output, `"level":"ERROR"`) {
					t.Fatalf("cancellation diagnostics: %s", output)
				}
			})
		})
	}
}

func TestDeploymentExecutionTimeoutReleasesWaitAndPreservesTerminalResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, store, _ := controlTestService()
		service.executionTimeout = 30 * time.Millisecond
		store.current.Store("deployment-1")
		release := make(chan struct{})
		var releaseOnce sync.Once
		defer releaseOnce.Do(func() { close(release) })
		err := service.executeDeployment(context.Background(), "project-1", "app-1", "deployment-1", "stop", func(ctx context.Context, _ model.Application, _ model.Deployment) error {
			setDeploymentStage(ctx, "compose_stop")
			<-release
			return nil
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout result=%v", err)
		}
		_, completions := store.results()
		if len(completions) != 1 || completions[0].status != status.WorkStatusFaulted || !strings.Contains(completions[0].message, "30ms") || !strings.Contains(completions[0].message, "compose_stop") {
			t.Fatalf("timeout completion=%+v", completions)
		}
		releaseOnce.Do(func() { close(release) })
		synctest.Wait()
		_, after := store.results()
		if len(after) != 1 || after[0] != completions[0] {
			t.Fatalf("late result changed completion: %+v", after)
		}
	})
}

func controlTestService() (ExecutionService, *controlExecutionStore, *controlLog) {
	store := &controlExecutionStore{}
	logs := &controlLog{}
	return NewExecutionService(store, slog.New(slog.NewJSONHandler(logs, nil)), nil, nil, nil, 5*time.Millisecond, time.Second, 200*time.Millisecond, controlTransaction{}, nil, nil, nil, nil), store, logs
}

type controlCompletion struct{ id, status, message string }
type controlExecutionStore struct {
	deploymentport.ExecutionStore
	current     atomic.Value
	mu          sync.Mutex
	writes      []string
	completions []controlCompletion
}

func (*controlExecutionStore) Application(context.Context, string, string) (model.Application, error) {
	return model.Application{Id: "app-1"}, nil
}
func (*controlExecutionStore) Deployment(_ context.Context, _, id string) (model.Deployment, error) {
	serviceId := "service-1"
	return model.Deployment{Id: id, ServiceId: &serviceId, Status: status.WorkStatusRunning}, nil
}
func (s *controlExecutionStore) Service(context.Context, string, string) (model.Service, error) {
	current := s.current.Load().(string)
	return model.Service{Id: "service-1", CurrentDeploymentId: &current}, nil
}
func (*controlExecutionStore) BeginDeployment(context.Context, string, string) (bool, error) {
	return true, nil
}
func (*controlExecutionStore) CancelSupersededDeployments(context.Context, string, string, string) ([]string, error) {
	return nil, errors.New("old cancellation write failed")
}
func (*controlExecutionStore) CancelObsoleteDeployment(context.Context, string, string) (bool, error) {
	return true, nil
}
func (s *controlExecutionStore) UpdateServiceDeploymentResult(_ context.Context, _, _, id, _ string, _ *string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writes = append(s.writes, id)
	return true, nil
}
func (s *controlExecutionStore) CompleteDeployment(_ context.Context, _, id, state, message string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completions = append(s.completions, controlCompletion{id: id, status: state, message: message})
	return true, nil
}
func (s *controlExecutionStore) results() ([]string, []controlCompletion) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.writes...), append([]controlCompletion(nil), s.completions...)
}

type controlTransaction struct{}

func (controlTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type controlLog struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (l *controlLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.Write(data)
}
func (l *controlLog) String() string { l.mu.Lock(); defer l.mu.Unlock(); return l.buffer.String() }

type blockedTakeoverCancellation struct {
	deploymentport.ExecutionStore
	release <-chan struct{}
	exited  chan<- struct{}
}

func (s blockedTakeoverCancellation) CancelSupersededDeployments(context.Context, string, string, string) ([]string, error) {
	defer close(s.exited)
	<-s.release
	return nil, nil
}
