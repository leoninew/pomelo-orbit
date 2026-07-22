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
	ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[model.Application], error)
	Application(ctx context.Context, id string) (model.Application, error)
	ApplicationByName(ctx context.Context, name string) (model.Application, error)
	ApplicationByCode(ctx context.Context, code string) (model.Application, error)
	CreateApplication(ctx context.Context, app model.Application) error
	UpdateApplication(ctx context.Context, app model.Application) error
	DeleteApplication(ctx context.Context, id string) error

	// Version / Component / Expose
	ListVersions(ctx context.Context, applicationId string) ([]model.Version, error)
	Version(ctx context.Context, id string) (model.Version, error)
	CreateVersion(ctx context.Context, version model.Version) error
	UpdateVersion(ctx context.Context, version model.Version) error
	ComponentsByVersion(ctx context.Context, versionId string) ([]model.Component, error)
	ExposesByVersion(ctx context.Context, versionId string) ([]model.Expose, error)
	ReplaceComponents(ctx context.Context, versionId string, components []model.Component) error
	ReplaceExposes(ctx context.Context, versionId string, exposes []model.Expose) error
	CreateVersionWithComponentsAndExposes(ctx context.Context, version model.Version, components []model.Component, exposes []model.Expose) error

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
	ServiceByKey(ctx context.Context, applicationId string, environmentId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpsertService(ctx context.Context, svc model.Service) error
	ClearIngressForAppEnv(ctx context.Context, applicationId string, environmentId string, exceptServiceId string) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error

	CreateDeployment(ctx context.Context, deployment model.Deployment) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
	ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.Deployment], error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	CancelDeployment(ctx context.Context, id string) error
	ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Route], error)
	ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error)
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
	ComponentsByVersion(ctx context.Context, versionId string) ([]model.Component, error)
	ExposesByVersion(ctx context.Context, versionId string) ([]model.Expose, error)
	Environment(ctx context.Context, id string) (model.Environment, error)
	ServiceByKey(ctx context.Context, applicationId string, environmentId string, instanceKey string) (model.Service, error)
	Service(ctx context.Context, id string) (model.Service, error)
	UpsertService(ctx context.Context, svc model.Service) error
	ClearIngressForAppEnv(ctx context.Context, applicationId string, environmentId string, exceptServiceId string) error
	UpdateServiceStatus(ctx context.Context, id string, status string) error
	UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
}
