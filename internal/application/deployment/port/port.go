package port

import (
	"context"
	"io"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
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
}

type ExecutionLogStore interface {
	Read(serviceCode string, deploymentID string, offset int) ([]byte, int, error)
	Writer(serviceCode string, deploymentID string) (io.WriteCloser, error)
	Remove(serviceCode string, deploymentID string) error
}

type RemoteFile struct {
	Path           string
	Content        []byte
	Mode           uint32
	IgnoreIfExists bool
}

type RemoteWorkspace struct {
	ServiceCode  string
	Directories  []string
	Files        []RemoteFile
	Compose      string
	DeploymentID string
}

// RemoteRuntime is the only deployment execution boundary. Every operation
// receives an explicit Project Environment target; no local fallback exists.
type RemoteRuntime interface {
	ServiceDir(target environmentport.SSHTarget, serviceCode string) (string, error)
	ServiceDirExists(ctx context.Context, target environmentport.SSHTarget, serviceCode string) (bool, error)
	StageWorkspace(ctx context.Context, target environmentport.SSHTarget, workspace RemoteWorkspace) error
	Run(ctx context.Context, target environmentport.SSHTarget, serviceCode string, log io.Writer, name string, args ...string) error
	Query(ctx context.Context, target environmentport.SSHTarget, serviceCode string, name string, args ...string) (string, error)
	QueryAtEnvironmentRoot(ctx context.Context, target environmentport.SSHTarget, name string, args ...string) (string, error)
	SyncFiles(ctx context.Context, target environmentport.SSHTarget, directory string, files []RemoteFile, pruneSuffix string) error
}

// GatewayRoutePublisher restores the complete custom Route snapshot after a
// Gateway Compose deployment has replaced the Traefik REST provider state.
type GatewayRoutePublisher interface {
	PublishSnapshot(ctx context.Context, projectId string) error
}

// GatewayDeploymentCoordinator resolves and selects Gateway state required by
// a deployment. The deployment domain owns this outbound port because both
// operations are mandatory before a Gateway deployment is snapshotted.
type GatewayDeploymentCoordinator interface {
	GatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error)
	SelectGatewayDeploymentVersion(ctx context.Context, app model.Application, service model.Service) (model.Service, error)
}
