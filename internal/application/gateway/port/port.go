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
	ListApplications(ctx context.Context, projectId string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error)
	Application(ctx context.Context, projectId string, id string) (model.Application, error)
	ApplicationByProjectAndName(ctx context.Context, projectId string, name string) (model.Application, error)
	ApplicationByProjectAndCode(ctx context.Context, projectId string, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, projectId string, app model.Application) error
	Version(ctx context.Context, projectId string, id string) (model.Version, error)
	CreateVersion(ctx context.Context, projectId string, version model.Version) error
	VersionComponentsByVersion(ctx context.Context, projectId string, versionId string) ([]model.VersionComponent, error)
	ReplaceVersionComponents(ctx context.Context, projectId string, versionId string, components []model.VersionComponent) error
}

// ConfigStore persists gateway configuration and resolves the active gateway.
type ConfigStore interface {
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	GatewayConfigByProject(ctx context.Context, projectId string) (model.GatewayConfig, error)
	ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error)
	UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error
	ReplaceGatewayVersionBindings(ctx context.Context, applicationId string, bindings []model.GatewayVersionBinding) error
}

// ServiceReader exposes only runtime bindings needed by gateway projections and conflict checks.
type EnvironmentStore interface {
	EnvironmentByProject(ctx context.Context, projectId string) (model.Environment, error)
	BindGatewayApplication(ctx context.Context, environmentId string, gatewayApplicationId string) (bool, error)
}

type ServiceReader interface {
	ListServicesByApplication(ctx context.Context, projectId string, applicationId string) ([]model.Service, error)
	ServiceByKey(ctx context.Context, projectId string, applicationId string, instanceKey string) (model.Service, error)
	ServiceComponentsByService(ctx context.Context, projectId string, serviceId string) ([]model.ServiceComponent, error)
	UpdateServiceConfiguration(ctx context.Context, projectId string, svc model.Service, components []model.ServiceComponent) error
}
