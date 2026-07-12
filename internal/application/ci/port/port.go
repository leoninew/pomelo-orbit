package port

import (
	"context"
	"io"

	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type TaskService interface {
	EnqueueTyped(ctx context.Context, taskType string, payload any) (*tasksvc.Task, error)
}

type LogReader interface {
	Read(logPath string, offset int) ([]byte, int, error)
}

type ExecutionLogStore interface {
	LogReader
	Writer(logPath string) (io.WriteCloser, error)
}

type RunOptions struct {
	Image       string
	Script      string
	Environment []string
	Volumes     []ciworkspace.VolumeMount
	LogWriter   io.Writer
}

type ContainerRunner interface {
	Run(ctx context.Context, opts RunOptions) (int, string, error)
}
