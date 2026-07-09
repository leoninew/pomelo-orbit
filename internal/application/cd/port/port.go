package port

import (
	"context"
	"io"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type TaskService interface {
	EnqueueTyped(ctx context.Context, taskType string, payload any) (*tasksvc.Task, error)
}

type LogReader interface {
	Read(logPath string, offset int) ([]byte, int, error)
}

type CommandRunner interface {
	Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error
}

type RouteConfigPublisher interface {
	Sync(ctx context.Context, route model.Route) error
	Revoke(ctx context.Context, routeName string) error
	RevokeCertificate(ctx context.Context, routeName string) error
}

type RouteCertificateGenerator interface {
	Generate(ctx context.Context, domain string) (string, string, error)
}

type TraefikRouterClient interface {
	ListRouters(ctx context.Context) ([]model.TraefikRouter, error)
	IsConnectionError(err error) bool
}
