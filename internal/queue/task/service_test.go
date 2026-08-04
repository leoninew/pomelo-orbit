package tasksvc

import (
	"context"
	"testing"

	"github.com/oklog/ulid/v2"
)

type taskRepositoryStub struct {
	task Task
}

func (s *taskRepositoryStub) Enqueue(_ context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error {
	s.task = Task{Id: id, TaskType: taskType, PayloadJSON: payloadJSON, MaxAttempts: maxAttempts}
	return nil
}

func (s *taskRepositoryStub) FindById(_ context.Context, _ string) (*Task, error) {
	return &s.task, nil
}

func TestCreateGeneratesULID(t *testing.T) {
	repo := &taskRepositoryStub{}
	service := New(repo, 3)

	task, err := service.Create(context.Background(), CreateInput{TaskType: "test"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := ulid.ParseStrict(task.Id); err != nil {
		t.Fatalf("expected ULID task ID, got %q: %v", task.Id, err)
	}
}
