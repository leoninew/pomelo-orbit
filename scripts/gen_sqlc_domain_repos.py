#!/usr/bin/env python3
"""One-shot generator for domain sqlc repository implementations. Not a runtime dependency."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SQLC = "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc"
REPO = "gitee.com/leoninew/PomeloOrbit-go/internal/repository"
DBMODEL = "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
SQLCOMMON = "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
TX = "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
MODEL = "gitee.com/leoninew/PomeloOrbit-go/internal/model"
STATUS = "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
IDUTIL = "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"


def write(rel: str, content: str) -> None:
    path = ROOT / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content.lstrip("\n"), encoding="utf-8")
    print("wrote", rel)


COMMON_HEADER = '''package {pkg}

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	gensqlc "{sqlc_pkg}"
	"{model}"
	"{repo}"
	"{dbmodel}"
	"{sqlcommon}"
	"{tx}"
)

type Repository struct {{
	db *sql.DB
}}

func NewRepository(db *sql.DB) Repository {{
	return Repository{{db: db}}
}}

func (r Repository) q(ctx context.Context) *gensqlc.Queries {{
	if dbtx, err := tx.DBTXFrom(ctx); err == nil {{
		return gensqlc.New(dbtx)
	}}
	return gensqlc.New(r.db)
}}

func likePattern(search string) (empty string, pattern string) {{
	search = strings.TrimSpace(search)
	if search == "" {{
		return "", ""
	}}
	return search, "%" + search + "%"
}}

func lowerLike(search string) (empty string, pattern string) {{
	search = strings.TrimSpace(search)
	if search == "" {{
		return "", ""
	}}
	return search, "%" + strings.ToLower(search) + "%"
}}
'''


def main() -> None:
    write(
        "internal/repository/impl/sqlc/dbmodel/convert.go",
        f'''
package dbmodel

import (
	"database/sql"
	"time"
)

func NullString(value *string) sql.NullString {{
	if value == nil {{
		return sql.NullString{{}}
	}}
	return sql.NullString{{String: *value, Valid: true}}
}}

func StringPtr(value sql.NullString) *string {{
	if !value.Valid {{
		return nil
	}}
	return &value.String
}}

func NullTime(value *time.Time) sql.NullTime {{
	if value == nil {{
		return sql.NullTime{{}}
	}}
	return sql.NullTime{{Time: *value, Valid: true}}
}}

func TimePtr(value sql.NullTime) *time.Time {{
	if !value.Valid {{
		return nil
	}}
	return &value.Time
}}

func BoolInt(value bool) int64 {{
	if value {{
		return 1
	}}
	return 0
}}

func IntBool(value int64) bool {{
	return value != 0
}}

func NullInt64FromIntPtr(value *int) sql.NullInt64 {{
	if value == nil {{
		return sql.NullInt64{{}}
	}}
	return sql.NullInt64{{Int64: int64(*value), Valid: true}}
}}

func IntPtrFromNullInt64(value sql.NullInt64) *int {{
	if !value.Valid {{
		return nil
	}}
	v := int(value.Int64)
	return &v
}}
''',
    )

    # task
    write(
        "internal/repository/impl/sqlc/task/repository.go",
        f'''
package taskrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	status "{STATUS}"
	gensqlc "{SQLC}/task"
	"{TX}"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	"{DBMODEL}"
)

type Repository struct {{
	db *sql.DB
}}

func NewRepository(db *sql.DB) Repository {{
	return Repository{{db: db}}
}}

func (r Repository) q(ctx context.Context) *gensqlc.Queries {{
	if dbtx, err := tx.DBTXFrom(ctx); err == nil {{
		return gensqlc.New(dbtx)
	}}
	return gensqlc.New(r.db)
}}

func (r Repository) Enqueue(ctx context.Context, id string, taskType string, payloadJSON string, maxAttempts int) error {{
	if maxAttempts < 1 {{
		return errors.New("maxAttempts must be at least 1")
	}}
	now := time.Now().UTC()
	err := r.q(ctx).EnqueueTask(ctx, gensqlc.EnqueueTaskParams{{
		ID: id, TaskType: taskType, PayloadJson: payloadJSON, Status: status.TaskPending,
		MaxAttempts: int64(maxAttempts), CreatedAt: now, UpdatedAt: now,
	}})
	if err != nil {{
		return fmt.Errorf("enqueue task: %w", err)
	}}
	return nil
}}

func (r Repository) ClaimNext(ctx context.Context, workerId string, lockTimeout time.Duration) (*tasksvc.Task, error) {{
	var claimed *tasksvc.Task
	err := tx.RunInTx(ctx, r.db, func(txCtx context.Context) error {{
		cutoff := time.Now().UTC().Add(-lockTimeout)
		q := gensqlc.New(mustDBTX(txCtx))
		task, err := q.TaskToClaim(txCtx, gensqlc.TaskToClaimParams{{
			Status: status.TaskPending, Status_2: status.TaskRunning, LockedAt: sql.NullTime{{Time: cutoff, Valid: true}},
		}})
		if errors.Is(err, sql.ErrNoRows) {{
			return nil
		}}
		if err != nil {{
			return fmt.Errorf("select task to claim: %w", err)
		}}
		now := time.Now().UTC()
		rows, err := q.ClaimTask(txCtx, gensqlc.ClaimTaskParams{{
			Status: status.TaskRunning, LockedBy: sql.NullString{{String: workerId, Valid: true}},
			LockedAt: sql.NullTime{{Time: now, Valid: true}}, StartedAt: sql.NullTime{{Time: now, Valid: true}},
			UpdatedAt: now, ID: task.ID, Status_2: status.TaskPending, Status_3: status.TaskRunning,
			LockedAt_2: sql.NullTime{{Time: cutoff, Valid: true}},
		}})
		if err != nil {{
			return fmt.Errorf("claim task %s: %w", task.ID, err)
		}}
		if rows == 0 {{
			return nil
		}}
		found, err := q.FindTaskByID(txCtx, task.ID)
		if err != nil {{
			return fmt.Errorf("find claimed task %s: %w", task.ID, err)
		}}
		converted := taskFromSQLC(found)
		claimed = &converted
		return nil
	}})
	if err != nil {{
		return nil, err
	}}
	return claimed, nil
}}

func (r Repository) Complete(ctx context.Context, taskId string) error {{
	now := time.Now().UTC()
	err := r.q(ctx).CompleteTask(ctx, gensqlc.CompleteTaskParams{{
		Status: status.TaskSucceeded, FinishedAt: sql.NullTime{{Time: now, Valid: true}}, UpdatedAt: now, ID: taskId,
	}})
	if err != nil {{
		return fmt.Errorf("complete task %s: %w", taskId, err)
	}}
	return nil
}}

func (r Repository) Fail(ctx context.Context, taskId string, message string) error {{
	now := time.Now().UTC()
	err := r.q(ctx).FailTask(ctx, gensqlc.FailTaskParams{{
		Status: status.TaskFailed, Status_2: status.TaskPending,
		FinishedAt: sql.NullTime{{Time: now, Valid: true}}, UpdatedAt: now,
		ErrorMessage: sql.NullString{{String: message, Valid: true}}, ID: taskId,
	}})
	if err != nil {{
		return fmt.Errorf("fail task %s: %w", taskId, err)
	}}
	return nil
}}

func (r Repository) FindById(ctx context.Context, id string) (*tasksvc.Task, error) {{
	task, err := r.q(ctx).FindTaskByID(ctx, id)
	if err != nil {{
		return nil, fmt.Errorf("find task %s: %w", id, err)
	}}
	converted := taskFromSQLC(task)
	return &converted, nil
}}

func taskFromSQLC(task gensqlc.BackgroundTask) tasksvc.Task {{
	return tasksvc.Task{{
		Id: task.ID, TaskType: task.TaskType, PayloadJSON: task.PayloadJson, Status: task.Status,
		Attempts: int(task.Attempts), MaxAttempts: int(task.MaxAttempts),
		LockedBy: dbmodel.StringPtr(task.LockedBy), LockedAt: dbmodel.TimePtr(task.LockedAt),
		StartedAt: dbmodel.TimePtr(task.StartedAt), FinishedAt: dbmodel.TimePtr(task.FinishedAt),
		ErrorMessage: dbmodel.StringPtr(task.ErrorMessage), CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt,
	}}
}}

func mustDBTX(ctx context.Context) tx.DBTX {{
	dbtx, err := tx.DBTXFrom(ctx)
	if err != nil {{
		panic(err)
	}}
	return dbtx
}}
''',
    )

    print("partial generator done — remaining domains written by follow-up script body")


if __name__ == "__main__":
    main()
