package port

import (
	"context"
	"errors"
	"io"
	"time"

	logport "github.com/leoninew/pomelo-orbit/internal/application/logstream/port"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

var ErrLogNotReady = errors.New("container log configuration is not ready")

// Dispatcher persists the existing asynchronous task contract for deployment commands.
type Dispatcher interface {
	DispatchDeploy(ctx context.Context, input deploymentdto.DeployDispatchInput) error
	DispatchRestart(ctx context.Context, input deploymentdto.RestartDispatchInput) error
	DispatchStop(ctx context.Context, input deploymentdto.StopDispatchInput) error
}

// CommandStore is the narrow persistence view used to create deployment commands.
type CommandStore interface {
	UpdateServiceDeploymentDirectory(ctx context.Context, projectId, id, directory string, targetRevision int64) error
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	Application(ctx context.Context, projectId string, id string) (model.Application, error)
	Version(ctx context.Context, projectId string, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error)
	Service(ctx context.Context, projectId string, id string) (model.Service, error)
	ListServicesByApplication(ctx context.Context, projectId string, applicationId string) ([]model.Service, error)
	ServiceEnvByService(ctx context.Context, projectId string, serviceId string) ([]model.ServiceEnv, error)
	ServiceComponentsByService(ctx context.Context, projectId string, serviceId string) ([]model.ServiceComponent, error)
	UpdateServiceStatus(ctx context.Context, projectId string, id string, status string) error
	CreateDeployment(ctx context.Context, projectId string, deployment model.Deployment) error
	HasActiveDeployment(ctx context.Context, projectId string, serviceId string) (bool, error)
}

// ExecutionStore is the worker's narrow persistence view. It deliberately
// exposes domain models rather than generated SQLC or transport types.
type ExecutionStore interface {
	DirectoryServices(ctx context.Context, projectId string) ([]model.Service, error)
	BindServiceRuntimeDirectory(ctx context.Context, projectId, id, deploymentId, directory string, targetRevision int64) error
	Application(ctx context.Context, projectId string, id string) (model.Application, error)
	Deployment(ctx context.Context, projectId string, id string) (model.Deployment, error)
	Version(ctx context.Context, projectId string, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error)
	ServiceEnvByService(ctx context.Context, projectId string, serviceId string) ([]model.ServiceEnv, error)
	ServiceComponentsByService(ctx context.Context, projectId string, serviceId string) ([]model.ServiceComponent, error)
	Service(ctx context.Context, projectId string, id string) (model.Service, error)
	UpdateServiceStatus(ctx context.Context, projectId string, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, projectId string, id string, status string, versionId string) error
	BeginDeployment(ctx context.Context, projectId string, id string) (bool, error)
	CompleteDeployment(ctx context.Context, projectId string, id string, status string, message string) (bool, error)
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
}

type ExecutionLogStore interface {
	OpenReader(ctx context.Context, serviceCode, deploymentId string) (logport.Reader, error)
	Read(serviceCode string, deploymentId string, offset int) ([]byte, int, error)
	Writer(serviceCode string, deploymentId string) (io.WriteCloser, error)
	Remove(serviceCode string, deploymentId string) error
}

type WorkspaceFile struct {
	Path           string
	Content        []byte
	Mode           uint32
	IgnoreIfExists bool
}

// WorkspaceFiles operates on explicit paths within the target workspace.
type WorkspaceFiles interface {
	ReadFile(context.Context, environmentport.Target, string) ([]byte, error)
	ListFiles(context.Context, environmentport.Target, string) ([]string, error)
	RemoveFile(context.Context, environmentport.Target, string) error
}

// RuntimeSessions scopes reusable target connections to one caller-owned operation.
type RuntimeSessions interface {
	OpenSession(context.Context, environmentport.Target) (context.Context, func(), error)
}

type ServiceLocation struct {
	Code      string
	Directory string
}

type ServiceVersionSelector interface {
	SelectServiceDeploymentVersion(ctx context.Context, userId, projectId, serviceId, versionId string) (model.Service, error)
}

type Workspace struct {
	Location      ServiceLocation
	AdoptExisting bool
	Directories   []string
	Files         []WorkspaceFile
	Compose       string
	DeploymentId  string
}

// Runtime is the only deployment execution boundary. Every operation receives
// an explicit Project Environment target and dispatches only by target type.
type Runtime interface {
	Stream(ctx context.Context, target environmentport.Target, location ServiceLocation, output io.Writer, name string, args ...string) error
	ServiceDir(target environmentport.Target, location ServiceLocation) (string, error)
	ServiceDirExists(ctx context.Context, target environmentport.Target, location ServiceLocation) (bool, error)
	ResolveDirectory(ctx context.Context, target environmentport.Target, location ServiceLocation) (string, error)
	ComposeMountSourceDir(ctx context.Context, target environmentport.Target, location ServiceLocation) (string, error)
	StageWorkspace(ctx context.Context, target environmentport.Target, workspace Workspace) error
	Run(ctx context.Context, target environmentport.Target, location ServiceLocation, log io.Writer, name string, args ...string) error
	Query(ctx context.Context, target environmentport.Target, location ServiceLocation, name string, args ...string) (string, error)
	QueryAtEnvironmentRoot(ctx context.Context, target environmentport.Target, name string, args ...string) (string, error)
	QueryAtEnvironmentRootInput(ctx context.Context, target environmentport.Target, stdin []byte, name string, args ...string) (string, error)
	SyncFiles(ctx context.Context, target environmentport.Target, directory string, files []WorkspaceFile, pruneSuffix string) error
}

type GatewayReadinessChecker interface {
	WaitUntilReady(ctx context.Context, projectId string, gateway model.GatewayConfig, timeout time.Duration) error
}

// GatewayDeploymentCoordinator resolves and selects Gateway state required by
// a deployment. The deployment domain owns this outbound port because both
// operations are mandatory before a Gateway deployment is snapshotted.
type GatewayDeploymentCoordinator interface {
	GatewayForDeployment(ctx context.Context, projectId string, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error)
	SelectGatewayDeploymentVersion(ctx context.Context, projectId string, app model.Application, service model.Service) (model.Service, error)
}
