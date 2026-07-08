package taskrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	dbsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
)

type Task struct {
	Id           string     `db:"id" json:"id"`
	TaskType     string     `db:"task_type" json:"task_type"`
	PayloadJSON  string     `db:"payload_json" json:"payload_json"`
	Status       string     `db:"status" json:"status"`
	Attempts     int        `db:"attempts" json:"attempts"`
	MaxAttempts  int        `db:"max_attempts" json:"max_attempts"`
	LockedBy     *string    `db:"locked_by" json:"locked_by"`
	LockedAt     *time.Time `db:"locked_at" json:"locked_at"`
	StartedAt    *time.Time `db:"started_at" json:"started_at"`
	FinishedAt   *time.Time `db:"finished_at" json:"finished_at"`
	ErrorMessage *string    `db:"error_message" json:"error_message"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

type Repository struct {
	db      *sqlx.DB
	driver  string
	queries *dbsqlc.Queries
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver, queries: dbsqlc.New(db)}
}

func (r Repository) Enqueue(ctx context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error {
	if maxAttempts < 1 {
		return errors.New("maxAttempts must be at least 1")
	}
	err := r.queries.EnqueueTask(ctx, dbsqlc.EnqueueTaskParams{ID: id, TaskType: taskType, PayloadJson: payloadJSON, Status: status.TaskPending, MaxAttempts: int64(maxAttempts)})
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}
	return nil
}

func (r Repository) ClaimNext(ctx context.Context, workerId string, lockTimeout time.Duration) (*Task, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	cutoff := time.Now().UTC().Add(-lockTimeout)
	queries := dbsqlc.New(tx)
	task, err := queries.TaskToClaim(ctx, dbsqlc.TaskToClaimParams{Status: status.TaskPending, Status_2: status.TaskRunning, LockedAt: sql.NullTime{Time: cutoff, Valid: true}})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select task to claim: %w", err)
	}

	now := db.NowExpr(r.driver)
	result, err := tx.ExecContext(ctx, fmt.Sprintf(`
				UPDATE background_task
				SET status = ?, attempts = attempts + 1, locked_by = ?, locked_at = %s,
				    started_at = COALESCE(started_at, %s), updated_at = %s, error_message = NULL
				WHERE id = ?
				  AND (status = ? OR (status = ? AND locked_at IS NOT NULL AND locked_at < ?))
			`, now, now, now), status.TaskRunning, workerId, task.ID, status.TaskPending, status.TaskRunning, cutoff)
	if err != nil {
		return nil, fmt.Errorf("claim task %s: %w", task.ID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read claimed rows: %w", err)
	}
	if rows == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim transaction: %w", err)
	}

	claimed, err := r.FindById(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	return claimed, nil
}

func (r Repository) Complete(ctx context.Context, taskId string) error {
	now := db.NowExpr(r.driver)
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
				UPDATE background_task
				SET status = ?, locked_by = NULL, locked_at = NULL, finished_at = %s,
				    updated_at = %s, error_message = NULL
				WHERE id = ?
			`, now, now), status.TaskSucceeded, taskId)
	if err != nil {
		return fmt.Errorf("complete task %s: %w", taskId, err)
	}
	return nil
}

func (r Repository) Fail(ctx context.Context, taskId string, message string) error {
	now := db.NowExpr(r.driver)
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`
				UPDATE background_task
				SET status = CASE WHEN attempts >= max_attempts THEN ? ELSE ? END,
				    locked_by = NULL, locked_at = NULL,
				    finished_at = CASE WHEN attempts >= max_attempts THEN %s ELSE finished_at END,
				    updated_at = %s, error_message = ?
				WHERE id = ?
			`, now, now), status.TaskFailed, status.TaskPending, message, taskId)
	if err != nil {
		return fmt.Errorf("fail task %s: %w", taskId, err)
	}
	return nil
}

func (r Repository) FindById(ctx context.Context, id string) (*Task, error) {
	task, err := r.queries.FindTaskByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find task %s: %w", id, err)
	}
	converted := taskFromSQLC(task)
	return &converted, nil
}

func taskFromSQLC(task dbsqlc.BackgroundTask) Task {
	return Task{Id: task.ID, TaskType: task.TaskType, PayloadJSON: task.PayloadJson, Status: task.Status, Attempts: int(task.Attempts), MaxAttempts: int(task.MaxAttempts), LockedBy: dbmodel.StringPtr(task.LockedBy), LockedAt: dbmodel.TimePtr(task.LockedAt), StartedAt: dbmodel.TimePtr(task.StartedAt), FinishedAt: dbmodel.TimePtr(task.FinishedAt), ErrorMessage: dbmodel.StringPtr(task.ErrorMessage), CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt}
}
