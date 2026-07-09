package tasksvc

import "time"

type Task struct {
	Id           string
	TaskType     string
	PayloadJSON  string
	Status       string
	Attempts     int
	MaxAttempts  int
	LockedBy     *string
	LockedAt     *time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
