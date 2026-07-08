package tasksvc

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

type Repository interface {
	Enqueue(ctx context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error
	FindById(ctx context.Context, id string) (*taskrepo.Task, error)
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
	MaxAttempts int
}

func New(repo Repository, defaultMaxAttempts int) Service {
	return Service{repo: repo, defaultMaxAttempts: defaultMaxAttempts}
}

func (s Service) Create(ctx context.Context, input CreateInput) (*taskrepo.Task, error) {
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

	id := strings.TrimSpace(input.Id)
	if id == "" {
		generated, err := NewId()
		if err != nil {
			return nil, fmt.Errorf("generate task id: %w", err)
		}
		id = generated
	}
	maxAttempts := input.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = s.defaultMaxAttempts
	}
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if maxAttempts < 1 {
		return nil, ErrInvalidMaxAttempts
	}

	if err := s.repo.Enqueue(ctx, id, taskType, payloadJSON, maxAttempts); err != nil {
		return nil, fmt.Errorf("enqueue task: %w", err)
	}
	item, err := s.repo.FindById(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read enqueued task: %w", err)
	}
	return item, nil
}

func (s Service) EnqueueTyped(ctx context.Context, taskType string, payload any) (*taskrepo.Task, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task payload: %w", err)
	}
	return s.Create(ctx, CreateInput{TaskType: taskType, PayloadJSON: string(payloadJSON)})
}

func (s Service) FindById(ctx context.Context, id string) (*taskrepo.Task, error) {
	return s.repo.FindById(ctx, strings.TrimSpace(id))
}

func NewId() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "task_" + hex.EncodeToString(b[:]), nil
}

var (
	ErrTaskTypeRequired   = apperror.New(apperror.KindValidation, "task_type is required")
	ErrInvalidPayload     = apperror.New(apperror.KindValidation, "payload must be valid JSON")
	ErrInvalidMaxAttempts = apperror.New(apperror.KindValidation, "max_attempts must be at least 1")
)
