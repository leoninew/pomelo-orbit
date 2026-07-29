package taskrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	tasksqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/task"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *tasksqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *tasksqlc.Queries {
		return tasksqlc.New(dbtx)
	})
}

func (r Repository) Enqueue(ctx context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error {
	if maxAttempts < 1 {
		return errors.New("maxAttempts must be at least 1")
	}
	now := time.Now().UTC()
	err := r.q(ctx).EnqueueTask(ctx, tasksqlc.EnqueueTaskParams{
		ID:          id,
		TaskType:    taskType,
		PayloadJson: payloadJSON,
		Status:      status.TaskPending,
		MaxAttempts: int64(maxAttempts),
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}
	return nil
}

func (r Repository) ClaimNext(ctx context.Context, workerId string, lockTimeout time.Duration) (*tasksvc.Task, error) {
	var claimedId string
	err := tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {
		q := r.q(txCtx)
		cutoff := time.Now().UTC().Add(-lockTimeout)
		task, err := q.TaskToClaim(txCtx, tasksqlc.TaskToClaimParams{
			Status:   status.TaskPending,
			Status_2: status.TaskRunning,
			LockedAt: sql.NullTime{Time: cutoff, Valid: true},
		})
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("select task to claim: %w", err)
		}
		now := time.Now().UTC()
		rows, err := q.ClaimTask(txCtx, tasksqlc.ClaimTaskParams{
			Status:     status.TaskRunning,
			LockedBy:   sql.NullString{String: workerId, Valid: true},
			LockedAt:   sql.NullTime{Time: now, Valid: true},
			StartedAt:  sql.NullTime{Time: now, Valid: true},
			UpdatedAt:  now,
			ID:         task.ID,
			Status_2:   status.TaskPending,
			Status_3:   status.TaskRunning,
			LockedAt_2: sql.NullTime{Time: cutoff, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("claim task %s: %w", task.ID, err)
		}
		if rows == 0 {
			return nil
		}
		claimedId = task.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	if claimedId == "" {
		return nil, nil
	}
	return r.FindById(ctx, claimedId)
}

func (r Repository) Complete(ctx context.Context, taskId string) error {
	now := time.Now().UTC()
	err := r.q(ctx).CompleteTask(ctx, tasksqlc.CompleteTaskParams{
		Status:     status.TaskSucceeded,
		FinishedAt: sql.NullTime{Time: now, Valid: true},
		UpdatedAt:  now,
		ID:         taskId,
	})
	if err != nil {
		return fmt.Errorf("complete task %s: %w", taskId, err)
	}
	return nil
}

func (r Repository) Fail(ctx context.Context, taskId string, message string) error {
	now := time.Now().UTC()
	err := r.q(ctx).FailTask(ctx, tasksqlc.FailTaskParams{
		Status:       status.TaskFailed,
		Status_2:     status.TaskPending,
		FinishedAt:   sql.NullTime{Time: now, Valid: true},
		UpdatedAt:    now,
		ErrorMessage: sql.NullString{String: message, Valid: true},
		ID:           taskId,
	})
	if err != nil {
		return fmt.Errorf("fail task %s: %w", taskId, err)
	}
	return nil
}

func (r Repository) FindById(ctx context.Context, id string) (*tasksvc.Task, error) {
	task, err := r.q(ctx).FindTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task %s: %w", id, err)
	}
	converted := taskFromSQLC(task)
	return &converted, nil
}

func taskFromSQLC(task tasksqlc.BackgroundTask) tasksvc.Task {
	return tasksvc.Task{
		Id:           task.ID,
		TaskType:     task.TaskType,
		PayloadJSON:  task.PayloadJson,
		Status:       task.Status,
		Attempts:     int(task.Attempts),
		MaxAttempts:  int(task.MaxAttempts),
		LockedBy:     dbmodel.StringPtr(task.LockedBy),
		LockedAt:     dbmodel.TimePtr(task.LockedAt),
		StartedAt:    dbmodel.TimePtr(task.StartedAt),
		FinishedAt:   dbmodel.TimePtr(task.FinishedAt),
		ErrorMessage: dbmodel.StringPtr(task.ErrorMessage),
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
	}
}
