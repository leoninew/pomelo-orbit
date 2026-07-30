package port

import (
	"context"
	"io"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// Dispatcher persists the existing asynchronous task contract for deployment commands.
type Dispatcher interface {
	DispatchDeploy(ctx context.Context, input deploymentdto.DeployDispatchInput) error
	DispatchRestart(ctx context.Context, input deploymentdto.RestartDispatchInput) error
	DispatchStop(ctx context.Context, input deploymentdto.StopDispatchInput) error
}

// CommandStore is the narrow persistence view used to create deployment commands.
type CommandStore interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	Application(ctx context.Context, id string) (model.Application, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	UpsertService(ctx context.Context, service model.Service) error
	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error)
}

// ExecutionStore is the worker's narrow persistence view. It deliberately
// exposes domain models rather than generated SQLC or transport types.
type ExecutionStore interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
}

type LogReader interface {
	Read(logPath string, offset int) ([]byte, int, error)
}

type CommandQueryRunner interface {
	Run(ctx context.Context, cwd string, name string, args ...string) (string, error)
}

type CommandRunner interface {
	Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error
}

type ExecutionLogStore interface {
	LogReader
	Writer(logPath string) (io.WriteCloser, error)
}

// Workspace is the deployment runtime filesystem boundary.
type Workspace interface {
	AppDir(appCode string) string
	ServiceDir(appCode string, instanceKey string) string
	ServiceDirExists(appCode string, instanceKey string) (bool, error)
	DeploymentLogPath(appCode string, instanceKey string, deploymentId string) string
	PhysicalDir(ctx context.Context) (string, error)
	PhysicalServiceDir(ctx context.Context, appCode string, instanceKey string) (string, error)
	WriteConfig(appCode string, instanceKey string, path string, content string) error
	RemoveAppDir(appCode string) error
}
