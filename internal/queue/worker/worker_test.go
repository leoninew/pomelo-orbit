package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

func TestWorkerCompletesClaimedTask(t *testing.T) {
	testWorkerDelivery(t, nil, nil)
}

func TestWorkerRecordsHandlerFailure(t *testing.T) {
	testWorkerDelivery(t, errors.New("execution failed"), nil)
}

func TestWorkerPropagatesResultPersistenceFailure(t *testing.T) {
	for _, handlerErr := range []error{nil, errors.New("execution failed")} {
		name := "complete"
		if handlerErr != nil {
			name = "fail"
		}
		t.Run(name, func(t *testing.T) {
			testWorkerDelivery(t, handlerErr, errors.New("result store unavailable"))
		})
	}
}

func testWorkerDelivery(t *testing.T, handlerErr, storeErr error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	claimed := tasksvc.Task{Id: "task-1", TaskType: status.TaskTypePipelineRunExecute, PayloadJSON: `{"pipeline_run_id":"run-1"}`, Attempts: 2, MaxAttempts: 3}
	repo := &deliveryStore{task: claimed, resultError: storeErr}
	router := NewRouter()
	calls := 0
	router.Register(claimed.TaskType, HandlerFunc(func(gotCtx context.Context, item tasksvc.Task) error {
		calls++
		if gotCtx != ctx || item.Id != claimed.Id || item.PayloadJSON != claimed.PayloadJSON || item.Attempts != claimed.Attempts {
			t.Fatalf("handler lost lifecycle context or claimed task: %+v", item)
		}
		if repo.completed != "" || repo.failed != "" {
			t.Fatal("result persisted before handler")
		}
		return handlerErr
	}))
	worker := New(repo, router, slog.New(slog.NewTextHandler(io.Discard, nil)), Config{WorkerId: "worker-1", PollInterval: time.Second, LeaseDuration: time.Minute, Concurrency: 1})
	err := worker.runOnce(ctx, 0)
	if storeErr == nil {
		if err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(err, storeErr) || (handlerErr != nil && !errors.Is(err, handlerErr)) {
		t.Fatalf("delivery error=%v, want storage error and original handler failure", err)
	}
	if calls != 1 || repo.workerId != "worker-1" || repo.lease != time.Minute || repo.claimCtx != ctx {
		t.Fatalf("claim worker=%s lease=%v handler calls=%d", repo.workerId, repo.lease, calls)
	}
	if handlerErr == nil {
		if repo.completed != claimed.Id || repo.failed != "" {
			t.Fatalf("complete=%s fail=%s", repo.completed, repo.failed)
		}
	} else if repo.failed != claimed.Id || repo.message != handlerErr.Error() || repo.completed != "" {
		t.Fatalf("complete=%s fail=%s message=%s", repo.completed, repo.failed, repo.message)
	}
}

type deliveryStore struct {
	task        tasksvc.Task
	claimCtx    context.Context
	workerId    string
	lease       time.Duration
	completed   string
	failed      string
	message     string
	resultError error
}

func (s *deliveryStore) ClaimNext(ctx context.Context, workerId string, lease time.Duration) (*tasksvc.Task, error) {
	s.claimCtx, s.workerId, s.lease = ctx, workerId, lease
	return &s.task, nil
}
func (s *deliveryStore) Complete(_ context.Context, id string) error {
	s.completed = id
	return s.resultError
}
func (s *deliveryStore) Fail(_ context.Context, id, message string) error {
	s.failed, s.message = id, message
	return s.resultError
}
