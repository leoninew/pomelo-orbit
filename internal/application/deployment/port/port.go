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
	Service(ctx context.Context, id string) (model.Service, error)
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	ServiceEnvByService(ctx context.Context, serviceId string) ([]model.ServiceEnv, error)
	ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error)
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	HasActiveDeployment(ctx context.Context, serviceId string) (bool, error)
	HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
}

// ExecutionStore is the worker's narrow persistence view. It deliberately
// exposes domain models rather than generated SQLC or transport types.
type ExecutionStore interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	ServiceEnvByService(ctx context.Context, serviceId string) ([]model.ServiceEnv, error)
	ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string) error
	BeginDeployment(ctx context.Context, id string) (bool, error)
	CompleteDeployment(ctx context.Context, id string, status string, message string) (bool, error)
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
	ServiceDir(serviceCode string) string
	ServiceDirExists(serviceCode string) (bool, error)
	DeploymentLogPath(serviceCode string, deploymentId string) string
	PhysicalDir(ctx context.Context) (string, error)
	PhysicalServiceDir(ctx context.Context, serviceCode string) (string, error)
	WriteConfig(serviceCode string, path string, content string) error
}
