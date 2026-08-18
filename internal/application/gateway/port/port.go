package port

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// ProjectReader is the membership read surface required by gateway operations.
type ProjectReader interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
}

// TransactionRunner groups Gateway resource creation into one database unit of
// work while reusing an existing request transaction when present.
type TransactionRunner interface {
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
}

// PhysicalPathResolver maps a path visible to Orbit to the path visible to the
// Docker daemon. For a native Orbit process, both paths are identical.
type PhysicalPathResolver func(ctx context.Context, logicalPath string) (string, error)

// ApplicationStore is the narrow application/version persistence surface used by gateway.
type ApplicationStore interface {
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error)
	Application(ctx context.Context, id string) (model.Application, error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error
	DeleteGatewayApplication(ctx context.Context, id string) error
	ListVersions(ctx context.Context, applicationId string) ([]model.Version, error)
	Version(ctx context.Context, id string) (model.Version, error)
	CreateVersion(ctx context.Context, version model.Version) error
	UpdateVersion(ctx context.Context, version model.Version) error
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
	ServiceByKey(ctx context.Context, applicationId string, instanceKey string) (model.Service, error)
	ServiceComponentsByService(ctx context.Context, serviceId string) ([]model.ServiceComponent, error)
}

// DeploymentCoordinator resolves the explicit Gateway configuration required to
// render a Service deployment. It never mutates or deploys Gateway resources.
type DeploymentCoordinator interface {
	GatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error)
}
