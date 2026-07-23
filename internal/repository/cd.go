package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// CDStore persists application deployment configuration and routes.
type CDStore interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string, kind string) (Page[model.Application], error)
	Application(ctx context.Context, id string) (model.Application, error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error

	// Version / Component / Expose
	ListVersions(ctx context.Context, applicationId string) ([]model.Version, error)
	ListVersionsPage(ctx context.Context, applicationId string, page int, perPage int, search string) (Page[model.Version], error)
	Version(ctx context.Context, id string) (model.Version, error)
	CreateVersion(ctx context.Context, version model.Version) error
	UpdateVersion(ctx context.Context, version model.Version) error
	// DeleteVersion removes a version row and its components/exposes.
	// Callers must enforce unpublished-only and non-reference rules.
	DeleteVersion(ctx context.Context, id string) error
	// CountVersionRuntimeRefs counts service/deployment rows that still point at the version.
	CountVersionRuntimeRefs(ctx context.Context, versionId string) (int, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error)
	ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error
	ReplaceVersionExposes(ctx context.Context, versionId string, exposes []model.VersionExpose) error
	CreateVersionWithVersionComponentsAndExposes(ctx context.Context, version model.Version, components []model.VersionComponent, exposes []model.VersionExpose) error

	// Environment
	ListEnvironments(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Environment], error)
	Environment(ctx context.Context, id string) (model.Environment, error)
	EnvironmentByProjectCode(ctx context.Context, projectId string, code string) (model.Environment, error)
	CreateEnvironment(ctx context.Context, env model.Environment) error
	UpdateEnvironment(ctx context.Context, env model.Environment) error
	DeleteEnvironment(ctx context.Context, id string) error
	CountServicesByEnvironment(ctx context.Context, environmentId string) (int, error)

	// Runtime Service binding
	ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error)
	// ListServicesByProject lists runtime bindings for applications in a project, with display labels.
	ListServicesByProject(ctx context.Context, projectId string, applicationId string, environmentId string, status string, search string, page int, perPage int) (Page[model.ServiceListItem], error)
	// ServiceListItem loads one runtime binding with application / environment / version labels.
	ServiceListItem(ctx context.Context, id string) (model.ServiceListItem, error)
	ServiceByKey(ctx context.Context, applicationId string, environmentId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpsertService(ctx context.Context, svc model.Service) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error

	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.Deployment], error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	CancelDeployment(ctx context.Context, id string) error
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Route], error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
	// ListEnabledRoutes returns every enabled route across projects (platform rest snapshot).
	ListEnabledRoutes(ctx context.Context) ([]model.Route, error)
	// HasActiveGatewayService reports whether any gateway Service is deploying/running,
	// optionally excluding one application id (for redeploy of the same gateway).
	HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error)
	// Gateway config (kind=gateway only)
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	// ResolveActiveGatewayConfig returns config for the active gateway Service, or the sole gateway app.
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
	// ListGatewayApplications lists applications with kind=gateway for a project.
	ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error)
	CreateApplicationWithGatewayConfig(ctx context.Context, app model.Application, cfg model.GatewayConfig) error
	UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error
	Route(ctx context.Context, id string) (model.Route, error)
	RouteByDomain(ctx context.Context, domain string) (model.Route, error)
	CreateRoute(ctx context.Context, route model.Route) error
	UpdateRoute(ctx context.Context, route model.Route) error
	DeleteRoute(ctx context.Context, id string) error
}

// DeploymentExecutionStore persists state transitions and inputs of a deployment execution.
type DeploymentExecutionStore interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	Version(ctx context.Context, id string) (model.Version, error)
	VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error)
	VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error)
	Environment(ctx context.Context, id string) (model.Environment, error)
	ServiceByKey(ctx context.Context, applicationId string, environmentId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpsertService(ctx context.Context, svc model.Service) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
	// Gateway config resolution for compose Host / rest publish during deploy execution.
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
}
