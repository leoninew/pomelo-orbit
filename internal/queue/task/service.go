package tasksvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
)

type Repository interface {
	Enqueue(ctx context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error
	FindById(ctx context.Context, id string) (*Task, error)
}

type Service struct {
	repo               Repository
	defaultMaxAttempts int
}

type CreateInput struct {
	Id          string
	TaskType    string
	Payload     json.RawMessage
	PayloadJSON string
}

func New(repo Repository, defaultMaxAttempts int) Service {
	return Service{repo: repo, defaultMaxAttempts: defaultMaxAttempts}
}

func (s Service) Create(ctx context.Context, input CreateInput) (*Task, error) {
	taskType := strings.TrimSpace(input.TaskType)
	if taskType == "" {
		return nil, ErrTaskTypeRequired
	}
	payloadJSON := strings.TrimSpace(input.PayloadJSON)
	if payloadJSON == "" && len(input.Payload) > 0 {
		payloadJSON = string(input.Payload)
	}
	if payloadJSON == "" {
		payloadJSON = "{}"
	}
	if !json.Valid([]byte(payloadJSON)) {
		return nil, ErrInvalidPayload
	}
	if s.defaultMaxAttempts < 1 {
		return nil, ErrInvalidMaxAttempts
	}

	id := strings.TrimSpace(input.Id)
	if id == "" {
		id = idutil.NewId()
	}
	if err := s.repo.Enqueue(ctx, id, taskType, payloadJSON, s.defaultMaxAttempts); err != nil {
		return nil, fmt.Errorf("enqueue task: %w", err)
	}
	item, err := s.repo.FindById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read enqueued task: %w", err)
	}
	return item, nil
}

func (s Service) EnqueueTyped(ctx context.Context, taskType string, payload any) (*Task, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}
	return s.Create(ctx, CreateInput{TaskType: taskType, PayloadJSON: string(payloadJSON)})
}

func (s Service) FindById(ctx context.Context, id string) (*Task, error) {
	return s.repo.FindById(ctx, strings.TrimSpace(id))
}

var (
	ErrTaskTypeRequired   = apperror.New(apperror.KindValidation, "task_type is required")
	ErrInvalidPayload     = apperror.New(apperror.KindValidation, "payload must be valid JSON")
	ErrInvalidMaxAttempts = apperror.New(apperror.KindValidation, "default max_attempts must be at least 1")
)
