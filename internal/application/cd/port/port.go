package port

import (
	"context"
	"io"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type ApplicationDispatcher interface {
	DispatchApplicationDeploy(ctx context.Context, input cdto.ApplicationDeployDispatchInput) error
	DispatchApplicationRestart(ctx context.Context, input cdto.ApplicationRestartDispatchInput) error
	DispatchApplicationStop(ctx context.Context, input cdto.ApplicationStopDispatchInput) error
}

type LogReader interface {
	Read(logPath string, offset int) ([]byte, int, error)
}

type ExecutionLogStore interface {
	LogReader
	Writer(logPath string) (io.WriteCloser, error)
}

type Workspace interface {
	// AppDir is the application root under data/cd/{appCode}.
	AppDir(appCode string) string
	// ServiceDir is data/cd/{appCode}/{envCode}/{instanceKey}.
	ServiceDir(appCode string, envCode string, instanceKey string) string
	DeploymentLogPath(appCode string, envCode string, instanceKey string, deploymentID string) string
	PhysicalDir(ctx context.Context) (string, error)
	PhysicalServiceDir(ctx context.Context, appCode string, envCode string, instanceKey string) (string, error)
	WriteConfig(appCode string, envCode string, instanceKey string, path string, content string) error
	RemoveAppDir(appCode string) error
}

type CommandRunner interface {
	Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error
}

type CommandQueryRunner interface {
	Run(ctx context.Context, cwd string, name string, args ...string) (string, error)
}

type RouteConfigPublisher interface {
	Sync(ctx context.Context, route model.Route) error
	Revoke(ctx context.Context, routeName string) error
	RevokeCertificate(ctx context.Context, routeName string) error
}

type RouteCertificateGenerator interface {
	Generate(ctx context.Context, domain string) (string, string, error)
}

// TraefikRouter is the router data exposed by the Traefik integration.
type TraefikRouter struct {
	Name        string
	Provider    string
	Status      string
	Rule        string
	Service     string
	Entrypoints []string
	TLS         bool
}

type TraefikRouterClient interface {
	ListRouters(ctx context.Context) ([]TraefikRouter, error)
	IsConnectionError(err error) bool
}
