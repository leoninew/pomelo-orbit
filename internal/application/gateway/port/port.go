package port

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// ProjectReader is the membership read surface required by gateway operations.
type ProjectReader interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
}

// ApplicationStore is the narrow application/version persistence surface used by gateway.
type ApplicationStore interface {
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error)
	Application(ctx context.Context, id string) (model.Application, error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error
	ListVersions(ctx context.Context, applicationId string) ([]model.Version, error)
	CreateVersion(ctx context.Context, version model.Version) error
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error
}

// ConfigStore persists gateway configuration and resolves the active gateway.
type ConfigStore interface {
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
	UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error
}

// ServiceReader exposes only runtime bindings needed by gateway projections and conflict checks.
type ServiceReader interface {
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	ServiceExposesByService(ctx context.Context, serviceId string) ([]model.ServiceExpose, error)
}

// Workspace owns the optional on-disk application cleanup operation.
type Workspace interface {
	RemoveAppDir(appCode string) error
}

// DeploymentCoordinator resolves the explicit Gateway configuration required to
// render a Service deployment. It never mutates or deploys Gateway resources.
type DeploymentCoordinator interface {
	EnsureGatewayRunning(ctx context.Context, app model.Application) error
	GatewayForDeployment(ctx context.Context, app model.Application, exposes []model.ServiceExpose) (*model.GatewayConfig, error)
}
